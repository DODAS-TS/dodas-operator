package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CloudAuthFields fields for cloud provider
type CloudAuthFields struct {
	Host string	`json:"host"`
}


// IMAuthFields fields for cloud provider
type IMAuthFields struct {
	Host string	`json:"host"`
}

// InfrastructureSpec defines the desired state of Infrastructure
type InfrastructureSpec struct {
	Name      string             `json:"name"`
	Image     string             `json:"image"`
	CloudAuth CloudAuthFields `json:"cloud"`
	ImAuth    IMAuthFields `json:"im,omitempty"`
	Template string `json:"template"`
	AuthFile  string `json:"authFile,omitempty"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// InfrastructureStatus defines the observed state of Infrastructure
type InfrastructureStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	InfID      string             `json:"infID"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Infrastructure is the Schema for the infrastructures API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=infrastructures,scope=Namespaced
type Infrastructure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InfrastructureSpec   `json:"spec,omitempty"`
	Status InfrastructureStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureList contains a list of Infrastructure
type InfrastructureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Infrastructure `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Infrastructure{}, &InfrastructureList{})
}
