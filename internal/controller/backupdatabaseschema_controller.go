/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1 "github.com/cyse7125-sp25-team03/db-backup-operator/api/v1"
)

// BackupDatabaseSchemaReconciler reconciles a BackupDatabaseSchema object
type BackupDatabaseSchemaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.csyeteam03.xyz,resources=backupdatabaseschemas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.csyeteam03.xyz,resources=backupdatabaseschemas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.csyeteam03.xyz,resources=backupdatabaseschemas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BackupDatabaseSchema object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *BackupDatabaseSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling BackupDatabaseSchema", "name", req.NamespacedName)

	backup := &backupv1.BackupDatabaseSchema{}
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to fetch BackupDatabaseSchema")
		return ctrl.Result{}, err
	}

	if backup.Spec.BackupJobNamespace == "" {
		log.Info("Backup job namespace is not defined")
		return ctrl.Result{}, nil
	}

	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: "backup-database-schema-job", Namespace: backup.Spec.BackupJobNamespace}, job)
	if err == nil {
		// Job exists, check its status
		if job.Status.Active > 0 {
			log.Info("Backup job is still running, skipping job creation")
			return ctrl.Result{}, nil
		} else if job.Status.Succeeded > 0 {
			log.Info("Backup job completed successfully, deleting it to allow a new one")
			backup.Status.Status = "Completed"
		} else if job.Status.Failed > 0 {
			log.Info("Backup job failed, deleting it to retry")
			backup.Status.Status = "Failed"
		}
		// Update the status before deleting the job
		log.Info("Updating BackupDatabaseSchema status")
		if err := r.Status().Update(ctx, backup); err != nil {
			log.Error(err, "Failed to update BackupDatabaseSchema status")
			return ctrl.Result{}, err
		}

	} else if !errors.IsNotFound(err) {
		// Unexpected error fetching the job
		log.Error(err, "Failed to check for existing job")
		return ctrl.Result{}, err
	} else {
		// Generate timestamped backup file name
		timestamp := time.Now().UTC().Format("20060102-150405")
		backupFileName := fmt.Sprintf("backup-%s.sql", timestamp)
		log.Info("Creating new backup job")
		finalizerName := "backup.csyeteam03.xyz/finalizer"

		newJob := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "backup-database-schema-job",
				Namespace:  backup.Spec.BackupJobNamespace,
				Labels:     map[string]string{"backup-database": "true"},
				Finalizers: []string{finalizerName},
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: backup.Spec.KubeServiceAccount,
						Containers: []corev1.Container{
							{
								Name:  "pg-dump",
								Image: "postgres:17.4",
								Command: []string{
									"sh", "-c",
									`set -e
									apt-get update && apt-get install -y curl gnupg && \
									curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
									echo "deb http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
									apt-get update && apt-get install -y google-cloud-cli && \
									rm -rf /var/lib/apt/lists/*			

									echo "checking gsutil version..."
									gsutil --version

									echo "Starting database backup..."
									BACKUP_FILE="/tmp/backup.sql"
									PGPASSWORD=$DB_PASSWORD pg_dump -h $DB_HOST -U $DB_USER -p $DB_PORT -d $DB_NAME -n $DB_SCHEMA > $BACKUP_FILE

									if [ $? -eq 0 ]; then
										echo "Backup successful! Uploading to GCS..."
										GCS_PATH="gs://$GCS_BUCKET/backups/backup-$(date +%Y%m%d-%H%M%S).sql"
										gsutil cp $BACKUP_FILE $GCS_PATH
										
										if [ $? -eq 0 ]; then
											echo "Upload successful! Backup saved at: $GCS_PATH"
										else
											echo "ERROR: Upload to GCS failed!" >&2
											exit 1
										fi
									else
										echo "ERROR: pg_dump failed!" >&2
										exit 1
									fi
								`},
								Env: []corev1.EnvVar{
									{Name: "DB_HOST", Value: backup.Spec.DbHost},
									{Name: "DB_USER", Value: backup.Spec.DbUser},
									{Name: "DB_PORT", Value: fmt.Sprintf("%d", backup.Spec.DbPort)},
									{Name: "DB_NAME", Value: backup.Spec.DbName},
									{Name: "DB_SCHEMA", Value: backup.Spec.DbSchema},
									{Name: "GCS_BUCKET", Value: backup.Spec.GcsBucket},
									{
										Name: "DB_PASSWORD",
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: backup.Spec.DbPasswordSecretName,
												},
												Key: backup.Spec.DbPasswordSecretKey,
											},
										},
									},
								},
							},
						},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}
		// Create the backup job
		if err := r.Create(ctx, newJob); err != nil {
			log.Error(err, "Failed to create backup job")
			return ctrl.Result{}, err
		}

		// Update the CRD status
		now := metav1.Now()
		backup.Status.LastBackupTime = &now
		location := fmt.Sprintf("gs://%s/backups/%s", backup.Spec.GcsBucket, backupFileName)
		backup.Status.BackupLocation = &location
		backup.Status.Status = "Running"
		backup.Status.JobName = newJob.Name
		log.Info("Updating BackupDatabaseSchema status123")
		if err := r.Status().Update(ctx, backup); err != nil {
			log.Error(err, "Failed to update BackupDatabaseSchema status")
			return ctrl.Result{}, err
		}
	}
	// Reconcile every 5 minutes
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupDatabaseSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1.BackupDatabaseSchema{}).
		Named("backupdatabaseschema").
		Complete(r)
}
