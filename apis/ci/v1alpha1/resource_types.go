package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// ResourceParameters are the configurable fields of a Resource.
type ResourceParameters struct {
	// TeamName is the Concourse team name.
	// +optional
	TeamName string `json:"teamName,omitempty"`

	// PipelineName is the Concourse pipeline name.
	// +optional
	PipelineName string `json:"pipelineName,omitempty"`

	// ResourceName is the Concourse resource name within the pipeline.
	// +kubebuilder:validation:Required
	ResourceName string `json:"resourceName"`

	// PinnedVersion pins the resource to a specific version.
	// +optional
	PinnedVersion map[string]string `json:"pinnedVersion,omitempty"`
}

// ResourceObservation are the observable fields of a Resource.
type ResourceObservation struct {
	// Type is the resource type (e.g. git, registry-image).
	Type string `json:"type,omitempty"`

	// PinnedVersion is the currently pinned version, if any.
	PinnedVersion map[string]string `json:"pinnedVersion,omitempty"`

	// LastChecked is the time of the last resource check.
	// +optional
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`
}

// ResourceSpec defines the desired state of a Resource.
type ResourceSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ResourceParameters `json:"forProvider"`
}

// ResourceStatus represents the observed state of a Resource.
type ResourceStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ResourceObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,concourse}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="TYPE",type=string,JSONPath=`.status.atProvider.type`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A PipelineResource is a managed resource that represents a Concourse CI pipeline resource.
// It can pin/unpin resource versions.
type PipelineResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceSpec   `json:"spec"`
	Status ResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineResourceList contains a list of PipelineResource.
type PipelineResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineResource `json:"items"`
}
