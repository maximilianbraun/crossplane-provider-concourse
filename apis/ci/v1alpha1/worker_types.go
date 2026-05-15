package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// WorkerParameters are the configurable fields of a Worker.
type WorkerParameters struct {
	// WorkerName is the name of the Concourse worker.
	// +kubebuilder:validation:Required
	WorkerName string `json:"workerName"`

	// DesiredState is the target state for the worker (running, landing, retiring).
	// +kubebuilder:validation:Enum=running;landing;retiring
	// +optional
	DesiredState string `json:"desiredState,omitempty"`
}

// WorkerObservation are the observable fields of a Worker.
type WorkerObservation struct {
	// State is the current worker state.
	State string `json:"state,omitempty"`

	// Version is the worker's reported version.
	Version string `json:"version,omitempty"`

	// Platform is the worker's platform (linux, darwin, windows).
	Platform string `json:"platform,omitempty"`

	// ActiveContainers is the number of containers on this worker.
	ActiveContainers int `json:"activeContainers,omitempty"`

	// ActiveVolumes is the number of volumes on this worker.
	ActiveVolumes int `json:"activeVolumes,omitempty"`
}

// WorkerSpec defines the desired state of a Worker.
type WorkerSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       WorkerParameters `json:"forProvider"`
}

// WorkerStatus represents the observed state of a Worker.
type WorkerStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          WorkerObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,concourse}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="STATE",type=string,JSONPath=`.status.atProvider.state`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Worker is a managed resource that represents a Concourse CI worker.
// Workers self-register; this resource observes and manages their lifecycle state.
type Worker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkerSpec   `json:"spec"`
	Status WorkerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkerList contains a list of Worker.
type WorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Worker `json:"items"`
}
