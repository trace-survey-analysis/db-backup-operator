package controller

import (
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// JobCleanupReconciler cleans up completed/failed jobs
type JobCleanupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;delete
func (r *JobCleanupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	job := &batchv1.Job{}
	if err := r.Get(ctx, req.NamespacedName, job); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to fetch Job")
		return ctrl.Result{}, err
	}
	if _, exists := job.Labels["backup-database"]; !exists {
		log.Info("Skipping cleanup for non-backup job", "job", job.Name)
		return ctrl.Result{}, nil
	}
	if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
		log.Info("Cleaning up completed/failed job", "job", job.Name)

		log.Info("Job is queued for deletion", "job", job.Name)
		if len(job.GetFinalizers()) > 0 {
			log.Info("Finalizer present, performing cleanup", "job", job.Name)

			// List all pods in the same namespace as the job
			podList := &corev1.PodList{}
			labelSelector := metav1.LabelSelector{
				MatchLabels: map[string]string{
					"job-name": job.Name,
				},
			}
			selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
			if err != nil {
				log.Error(err, "Failed to convert label selector")
				return ctrl.Result{}, err
			}
			opts := client.ListOptions{LabelSelector: selector, Namespace: job.Namespace}
			if err := r.List(ctx, podList, &opts); err != nil {
				log.Error(err, "Failed to list pods")
				return ctrl.Result{}, err
			}
			for _, pod := range podList.Items {
				log.Info("Deleting pod", "pod", pod.Name)
				if err := r.Delete(ctx, &pod); err != nil {
					log.Error(err, "Failed to delete pod", "pod", pod.Name)
					return ctrl.Result{}, err
				}
			}
			job.SetFinalizers(nil)
			if err := r.Update(ctx, job); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}

		}
		if err := r.Delete(ctx, job); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Job not found. It might have already been deleted.")
				return ctrl.Result{}, nil
			}
			log.Error(err, "Failed to delete job")
			return ctrl.Result{}, err
		}

		log.Info("Job deleted successfully")
		return ctrl.Result{}, nil

	}
	// Requeue periodically (e.g., every 1 minute)
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *JobCleanupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Named("jobcleanup").
		Complete(r)
}
