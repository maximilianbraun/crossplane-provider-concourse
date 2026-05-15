package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// BuildParameters are the configurable fields of a Build.
type BuildParameters struct {
	// JobRef is the name of the Job MR to trigger a build for.
	// +optional
	JobRef *xpv1.Reference `json:"jobRef,omitempty"`

	// +optional
	JobSelector *xpv1.Selector `json:"jobSelector,omitempty"`

	// JobName is the resolved Concourse job name.
	// +optional
	JobName string `json:"jobName,omitempty"`

	// PipelineName is the resolved Concourse pipeline name.
	// +optional
	PipelineName string `json:"pipelineName,omitempty"`

	// TeamName is the resolved Concourse team name.
	// +optional
	TeamName string `json:"teamName,omitempty"`

	// Abort signals that a running build should be aborted.
	// +optional
	Abort bool `json:"abort,omitempty"`
}

// BuildObservation are the observable fields of a Build.
type BuildObservation struct {
	// BuildID is the Concourse-assigned build ID. Once set, prevents re-triggering.
	BuildID int `json:"buildId,omitempty"`

	// BuildName is the human-readable build number (e.g. "42").
	BuildName string `json:"buildName,omitempty"`

	// Status is the current build status (pending, started, succeeded, failed, errored, aborted).
	Status string `json:"status,omitempty"`

	// StartTime is when the build started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// EndTime is when the build finished.
	// +optional
	EndTime *metav1.Time `json:"endTime,omitempty"`
}

// BuildSpec defines the desired state of a Build.
type BuildSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BuildParameters `json:"forProvider"`
}

// BuildStatus represents the observed state of a Build.
type BuildStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          BuildObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,concourse}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="BUILD-ID",type=integer,JSONPath=`.status.atProvider.buildId`
// +kubebuilder:printcolumn:name="STATUS",type=string,JSONPath=`.status.atProvider.status`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Build is a managed resource that triggers and tracks a Concourse CI build.
// It follows a run-to-completion pattern: once triggered, the build cannot be
// re-triggered. The resource becomes immutable after reaching a terminal state.
type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec"`
	Status BuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BuildList contains a list of Build.
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Build `json:"items"`
}
