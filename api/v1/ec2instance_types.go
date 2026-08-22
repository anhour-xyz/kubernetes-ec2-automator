package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EC2InstanceSpec defines the desired state of an EC2 instance.
type EC2InstanceSpec struct {
	AmiID             string            `json:"amiID"`
	SshKey            string            `json:"sshKey,omitempty"`
	InstanceType      string            `json:"instanceType,omitempty"`
	Subnet            string            `json:"subnet,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	Storage           StorageConfig     `json:"storage,omitempty"`
	AdditionalStorage []StorageConfig   `json:"additionalStorage,omitempty"`
	InstanceName      string            `json:"instanceName"`
	Region            string            `json:"region"`
}

type StorageConfig struct {
	Size int    `json:"size"`
	Type string `json:"type,omitempty"`
}

// EC2InstanceStatus defines the observed state of an EC2 instance.
type EC2InstanceStatus struct {
	Phase      string `json:"phase,omitempty"`
	InstanceID string `json:"instanceID,omitempty"`
	PublicIP   string `json:"publicIP,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EC2Instance is the Schema for the EC2 instances API.
type EC2Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`
	Spec              EC2InstanceSpec   `json:"spec"`
	Status            EC2InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EC2InstanceList contains a list of EC2Instance resources.
type EC2InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []EC2Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &EC2Instance{}, &EC2InstanceList{})
		return nil
	})
}
