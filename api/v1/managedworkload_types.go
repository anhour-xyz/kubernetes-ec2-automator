package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ManagedWorkloadSpec defines the desired state of ManagedWorkload
type ManagedWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of ManagedWorkload. Edit managedworkload_types.go to remove/update
	// +optional
	Image    string            `json:"image"`
	Replicas int32             `json:"replicas"`
	Port     int32             `json:"port,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
}

// ManagedWorkloadStatus defines the observed state of ManagedWorkload.
type ManagedWorkloadStatus struct {
	Phase                       string             `json:"phase,omitempty"`
	ReadyReplicas               int32              `json:"readyReplicas,omitempty"`
	AvailableReplicas           int32              `json:"availableReplicas,omitempty"`
	RecoveryCount               int64              `json:"recoveryCount,omitempty"`
	RecoveryStartedAt           *metav1.Time       `json:"recoveryStartedAt,omitempty"`
	LastRecoveryTime            *metav1.Time       `json:"lastRecoveryTime,omitempty"`
	LastRecoveryDurationSeconds int64              `json:"lastRecoveryDurationSeconds,omitempty"`
	Conditions                  []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ManagedWorkload is the Schema for the managedworkloads API
type ManagedWorkload struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ManagedWorkload
	// +required
	Spec ManagedWorkloadSpec `json:"spec"`

	// status defines the observed state of ManagedWorkload
	// +optional
	Status ManagedWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedWorkloadList contains a list of ManagedWorkload
type ManagedWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ManagedWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ManagedWorkload{}, &ManagedWorkloadList{})
		return nil
	})
}
