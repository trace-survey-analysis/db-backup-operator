# Database Backup Operator

![Go](https://img.shields.io/badge/Go-00ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![Kubebuilder](https://img.shields.io/badge/Kubebuilder-326CE5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED.svg?style=for-the-badge&logo=docker&logoColor=white)
![Make](https://img.shields.io/badge/Make-427819.svg?style=for-the-badge&logo=gnu&logoColor=white)
![Jenkins](https://img.shields.io/badge/Jenkins-D24939.svg?style=for-the-badge&logo=jenkins&logoColor=white)
![Semantic Release](https://img.shields.io/badge/Semantic_Release-494949.svg?style=for-the-badge&logo=semantic-release&logoColor=white)

A Kubernetes operator for automating PostgreSQL database schema backups to Google Cloud Storage (GCS).

## Overview

The Database Backup Operator creates and manages the process of backing up PostgreSQL database schemas to Google Cloud Storage. It uses the Kubernetes Operator pattern to define a custom resource that declares what database needs to be backed up, performs the backup operation, and tracks the backup status.

### Key Features

- **Automated Backups**: Schedule regular database schema backups
- **Cloud Storage Integration**: Uploads backups to Google Cloud Storage
- **Secure Credentials Management**: Uses Kubernetes secrets for database credentials
- **Status Tracking**: Maintains backup history and status
- **Self-Healing**: Automatically retries failed backups
- **Job Cleanup**: Automatically cleans up completed/failed jobs
- **Service Account Integration**: Uses Kubernetes and GCP service accounts for proper authentication

## Custom Resource Definition (CRD)

The operator introduces a new custom resource `BackupDatabaseSchema` with the following structure:

```yaml
apiVersion: backup.csyeteam03.xyz/v1
kind: BackupDatabaseSchema
metadata:
  name: database-backup
  namespace: backup-job-namespace
spec:
  # Database connection details
  dbHost: "pg-postgresql.postgres.svc.cluster.local"
  dbUser: "team03"
  dbPasswordSecretName: "backup-secret"
  dbPasswordSecretNamespace: "backup-job-namespace"
  dbPasswordSecretKey: "password"
  dbName: "api"
  dbSchema: "api"
  dbPort: 5432
  
  # Storage details
  gcsBucket: "operator-db-backups-bucket"
  
  # Service accounts for authorization
  kubeServiceAccount: "backup-sa"
  gcpServiceAccount: "db-operator-gsa"
  backupJobNamespace: "backup-job-namespace"
```

### Spec Fields

| Field | Description |
|-------|-------------|
| `dbHost` | PostgreSQL database hostname |
| `dbUser` | Database username |
| `dbPasswordSecretName` | Name of the Kubernetes secret containing the database password |
| `dbPasswordSecretNamespace` | Namespace of the password secret |
| `dbPasswordSecretKey` | Key in the secret that contains the password |
| `dbName` | Database name |
| `dbSchema` | Database schema to backup |
| `dbPort` | Database port number |
| `gcsBucket` | Google Cloud Storage bucket name |
| `kubeServiceAccount` | Kubernetes service account for the backup job |
| `gcpServiceAccount` | GCP service account for GCS access |
| `backupJobNamespace` | Namespace where backup jobs will be created |

### Status Fields

| Field | Description |
|-------|-------------|
| `lastBackupTime` | Timestamp of the last backup attempt |
| `backupLocation` | GCS path where the backup is stored |
| `status` | Status of the last backup (Running/Success/Failed) |
| `jobName` | Name of the Kubernetes job performing the backup |

## Architecture

The operator consists of two main controllers:

1. **BackupDatabaseSchemaReconciler**: Manages the lifecycle of database backups
2. **JobCleanupReconciler**: Cleans up completed or failed backup jobs

### Reconciliation Process

When a `BackupDatabaseSchema` resource is created or updated, the controller:

1. Checks if a backup job is already running
2. If a job is running, it skips creating a new one
3. If a job is completed or failed, it updates the status
4. If no job exists, it creates a new backup job with the following steps:
   - Uses the PostgreSQL image to run pg_dump
   - Installs the Google Cloud SDK
   - Dumps the specified schema to a file
   - Uploads the file to Google Cloud Storage
   - Updates the backup status with details
5. Schedules the next reconciliation after 5 minutes

### Job Cleanup Process

The JobCleanup controller:

1. Watches for completed or failed jobs with the label `backup-database: true`
2. Removes associated pods
3. Removes finalizers from the job
4. Deletes the job to allow new backups to run

## Installation

### Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured to communicate with your cluster
- Helm 3.0+
- Google Cloud Storage bucket
- Kubernetes service account with permissions to create jobs
- GCP service account with permissions to write to the GCS bucket

### Using Helm

The operator can be deployed using Helm from the [helm-charts repository](https://github.com/cyse7125-sp25-team03/helm-charts.git):

```bash
# Clone the helm-charts repository
git clone https://github.com/cyse7125-sp25-team03/helm-charts.git
cd helm-charts

# Install the operator
helm install db-backup-operator ./db-backup-operator -n operator-ns
```

### Creating a Custom Resource

After the operator is installed, create a BackupDatabaseSchema resource:

```bash
kubectl apply -f db-backup-operator/backup-job-cr.yaml
```

Example `backup-job-cr.yaml`:

```yaml
apiVersion: backup.csyeteam03.xyz/v1
kind: BackupDatabaseSchema
metadata:
  name: database-backup
  namespace: backup-job-namespace
spec:
  dbHost: "pg-postgresql.postgres.svc.cluster.local"
  dbUser: "team03"
  dbPasswordSecretName: "backup-secret"
  dbPasswordSecretNamespace: "backup-job-namespace"
  dbPasswordSecretKey: "password"
  dbName: "api"
  dbSchema: "api"
  dbPort: 5432
  gcsBucket: "operator-db-backups-bucket"
  kubeServiceAccount: "backup-sa"
  gcpServiceAccount: "db-operator-gsa"
  backupJobNamespace: "backup-job-namespace"
```

### Required Secrets

Create a secret for the database password:

```bash
kubectl create secret -n backup-job-namespace generic backup-secret --from-literal=password=your-password
```

## Development

### Prerequisites

- Go 1.21+
- Kubebuilder
- Docker
- Kind or Minikube for local testing
- GCP account for testing GCS integration

### Building

```bash
# Clone the repository
git clone https://github.com/cyse7125-sp25-team03/db-backup-operator.git
cd db-backup-operator

# Build the operator
make build

# Build the Docker image
make docker-build IMG=your-registry/db-backup-operator:tag

# Push the Docker image
make docker-push IMG=your-registry/db-backup-operator:tag
```

### Testing

```bash
# Run unit tests
make test

# Install CRDs in the cluster
make install

# Deploy the operator locally (outside the cluster)
make run
```

## Monitoring Backups

Check the status of your backup:

```bash
kubectl get backupdatabaseschema -n backup-job-namespace
```

View detailed backup information:

```bash
kubectl describe backupdatabaseschema database-backup -n backup-job-namespace
```

Check backup jobs:

```bash
kubectl get jobs -n backup-job-namespace
```

## Backup Files

Backup files are stored in the specified GCS bucket with the naming convention:

```
gs://[gcsBucket]/backups/backup-YYYYMMDD-HHMMSS.sql
```

## CI/CD

This project uses Jenkins for continuous integration and Semantic Release for versioning:

- When a pull request is successfully merged, a Docker image is built
- The Semantic Versioning bot creates a release on GitHub with a tag
- The tagged release is used for the Docker image, which is then pushed to Docker Hub

## Troubleshooting

### Common Issues

1. **Backup job fails with authentication errors**
   - Check if the Kubernetes service account exists and has necessary permissions
   - Verify the GCP service account has permissions to write to the GCS bucket

2. **Database connection issues**
   - Verify the database host, port, and credentials are correct
   - Check if the database is accessible from the Kubernetes cluster

3. **Job doesn't start**
   - Check if there are any resource constraints in the namespace
   - Verify the service account has permissions to create pods

To check job logs:

```bash
kubectl logs -n backup-job-namespace -l job-name=backup-database-schema-job
```

## License

This project is licensed under the GNU General Public License v3.0. See the [LICENSE](LICENSE) file for details.