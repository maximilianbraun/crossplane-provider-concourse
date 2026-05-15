package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// JobParameters are the configurable fields of a Job.
type JobParameters struct {
	// PipelineRef is the name of the Pipeline MR this job belongs to.
	// +optional
	PipelineRef *xpv1.Reference `json:"pipelineRef,omitempty"`

	// +optional
	PipelineSelector *xpv1.Selector `json:"pipelineSelector,omitempty"`

	// PipelineName is the resolved Concourse pipeline name.
	// +optional
	PipelineName string `json:"pipelineName,omitempty"`

	// TeamName is the resolved Concourse team name.
	// +optional
	TeamName string `json:"teamName,omitempty"`

	// JobName is the Concourse job name.
	// +kubebuilder:validation:Required
	JobName string `json:"jobName"`

	// Paused sets whether the job is paused.
	// +optional
	Paused *bool `json:"paused,omitempty"`
}

// JobObservation are the observable fields of a Job.
type JobObservation struct {
	// ID is the Concourse-assigned job ID.
	ID int `json:"id,omitempty"`

	// Paused indicates current pause state.
	Paused bool `json:"paused,omitempty"`

	// FinishedBuild is the number of the last finished build.
	FinishedBuild int `json:"finishedBuild,omitempty"`

	// NextBuild is the number of the next pending build.
	NextBuild int `json:"nextBuild,omitempty"`
}

// JobSpec defines the desired state of a Job.
type JobSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       JobParameters `json:"forProvider"`
}

// JobStatus represents the observed state of a Job.
type JobStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          JobObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,concourse}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type=string,JSONPath=`.metadata.annotations.crossplane\.io/external-name`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Job is a managed resource that represents a Concourse CI job.
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobSpec   `json:"spec"`
	Status JobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JobList contains a list of Job.
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}
