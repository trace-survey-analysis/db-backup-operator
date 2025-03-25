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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BackupStatusType string

const (
	BackupStatusRunning BackupStatusType = "Running"
	BackupStatusSuccess BackupStatusType = "Success"
	BackupStatusFailed  BackupStatusType = "Failed"
)

// BackupDatabaseSchemaSpec defines the desired state of BackupDatabaseSchema.
type BackupDatabaseSchemaSpec struct {
	DbHost                    string `json:"dbHost"`
	DbUser                    string `json:"dbUser"`
	DbPasswordSecretName      string `json:"dbPasswordSecretName"`
	DbPasswordSecretNamespace string `json:"dbPasswordSecretNamespace"`
	DbPasswordSecretKey       string `json:"dbPasswordSecretKey"`
	DbName                    string `json:"dbName"`
	DbSchema                  string `json:"dbSchema"`
	DbPort                    int    `json:"dbPort"`
	GcsBucket                 string `json:"gcsBucket"`
	KubeServiceAccount        string `json:"kubeServiceAccount"`
	GcpServiceAccount         string `json:"gcpServiceAccount"`
	BackupJobNamespace        string `json:"backupJobNamespace"`
}

// BackupDatabaseSchemaStatus defines the observed state of BackupDatabaseSchema.
type BackupDatabaseSchemaStatus struct {
	LastBackupTime *metav1.Time     `json:"lastBackupTime,omitempty"`
	BackupLocation *string          `json:"backupLocation,omitempty"`
	Status         BackupStatusType `json:"status,omitempty"`
	JobName        string           `json:"jobName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BackupDatabaseSchema is the Schema for the backupdatabaseschemas API.
type BackupDatabaseSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupDatabaseSchemaSpec   `json:"spec,omitempty"`
	Status BackupDatabaseSchemaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupDatabaseSchemaList contains a list of BackupDatabaseSchema.
type BackupDatabaseSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupDatabaseSchema `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupDatabaseSchema{}, &BackupDatabaseSchemaList{})
}
