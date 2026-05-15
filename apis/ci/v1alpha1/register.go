package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

const (
	Group   = "ci.concourse.crossplane.io"
	Version = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme        = SchemeBuilder.AddToScheme
)

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func Kind(kind string) schema.GroupKind {
	return schema.GroupKind{Group: Group, Kind: kind}
}

func init() {
	SchemeBuilder.Register(
		&Team{}, &TeamList{},
		&Pipeline{}, &PipelineList{},
		&Job{}, &JobList{},
		&Build{}, &BuildList{},
	)
}

// TeamKind is the kind for Team.
var TeamKind = reflect.TypeOf(Team{}).Name()

// TeamGroupKind is the group-kind for Team.
var TeamGroupKind = schema.GroupKind{Group: Group, Kind: TeamKind}.String()

// TeamGroupVersionKind is the GVK for Team.
var TeamGroupVersionKind = SchemeGroupVersion.WithKind(TeamKind)

// PipelineKind is the kind for Pipeline.
var PipelineKind = reflect.TypeOf(Pipeline{}).Name()

// PipelineGroupVersionKind is the GVK for Pipeline.
var PipelineGroupVersionKind = SchemeGroupVersion.WithKind(PipelineKind)
