package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// PipelineConfigSource specifies where the pipeline YAML comes from.
type PipelineConfigSource struct {
	// Inline is the raw pipeline YAML.
	// +optional
	Inline string `json:"inline,omitempty"`

	// ConfigMapRef references a ConfigMap containing the pipeline YAML.
	// +optional
	ConfigMapRef *xpv1.SecretKeySelector `json:"configMapRef,omitempty"`
}

// PipelineVar is a key-value variable passed to the pipeline.
type PipelineVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PipelineParameters are the configurable fields of a Pipeline.
type PipelineParameters struct {
	// TeamRef is the name of the Team MR this pipeline belongs to.
	// +optional
	TeamRef *xpv1.Reference `json:"teamRef,omitempty"`

	// +optional
	TeamSelector *xpv1.Selector `json:"teamSelector,omitempty"`

	// TeamName is the resolved Concourse team name.
	// +optional
	TeamName string `json:"teamName,omitempty"`

	// PipelineName is the Concourse pipeline name. If omitted, metadata.name is used.
	// +optional
	PipelineName string `json:"pipelineName,omitempty"`

	// Config is the pipeline configuration source.
	Config PipelineConfigSource `json:"config"`

	// Vars are template variables passed to the pipeline config.
	// +optional
	Vars []PipelineVar `json:"vars,omitempty"`

	// Paused sets whether the pipeline is paused.
	// +optional
	Paused *bool `json:"paused,omitempty"`

	// Exposed sets whether the pipeline is publicly visible.
	// +optional
	Exposed *bool `json:"exposed,omitempty"`
}

// PipelineObservation are the observable fields of a Pipeline.
type PipelineObservation struct {
	// ID is the Concourse-assigned pipeline ID.
	ID int `json:"id,omitempty"`

	// ConfigHash is the SHA256 of the last-applied config.
	ConfigHash string `json:"configHash,omitempty"`

	// Paused indicates current pause state.
	Paused bool `json:"paused,omitempty"`

	// Exposed indicates current expose state.
	Exposed bool `json:"exposed,omitempty"`
}

// PipelineSpec defines the desired state of a Pipeline.
type PipelineSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PipelineParameters `json:"forProvider"`
}

// PipelineStatus represents the observed state of a Pipeline.
type PipelineStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PipelineObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,concourse}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type=string,JSONPath=`.metadata.annotations.crossplane\.io/external-name`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Pipeline is a managed resource that represents a Concourse CI pipeline.
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec"`
	Status PipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineList contains a list of Pipeline.
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}
