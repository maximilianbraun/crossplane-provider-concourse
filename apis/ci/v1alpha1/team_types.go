package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// TeamRole maps a Concourse role to a list of user/group identifiers.
type TeamRole struct {
	// Name is the role name (owner, member, pipeline-operator, viewer).
	// +kubebuilder:validation:Enum=owner;member;pipeline-operator;viewer
	Name string `json:"name"`

	// Users are local usernames assigned this role.
	// +optional
	Users []string `json:"users,omitempty"`

	// Groups are group identifiers (e.g. GitHub org:team) assigned this role.
	// +optional
	Groups []string `json:"groups,omitempty"`
}

// TeamParameters are the configurable fields of a Team.
type TeamParameters struct {
	// TeamName is the Concourse team name. If omitted, metadata.name is used.
	// +optional
	TeamName string `json:"teamName,omitempty"`

	// Roles defines the RBAC roles for this team.
	// +optional
	Roles []TeamRole `json:"roles,omitempty"`
}

// TeamObservation are the observable fields of a Team.
type TeamObservation struct {
	// ID is the Concourse-assigned team ID.
	ID int `json:"id,omitempty"`
}

// TeamSpec defines the desired state of a Team.
type TeamSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TeamParameters `json:"forProvider"`
}

// TeamStatus represents the observed state of a Team.
type TeamStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TeamObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,concourse}
// +kubebuilder:printcolumn:name="SYNCED",type=string,JSONPath=`.status.conditions[?(@.type=='Synced')].status`
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type=string,JSONPath=`.metadata.annotations.crossplane\.io/external-name`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// A Team is a managed resource that represents a Concourse CI team.
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team.
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}
