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
	SchemeBuilder      = &scheme.Builder{GroupVersion: SchemeGroupVersion} //nolint:staticcheck // standard Crossplane pattern
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
		&Worker{}, &WorkerList{},
		&PipelineResource{}, &PipelineResourceList{},
	)
}

var TeamKind = reflect.TypeFor[Team]().Name()

var TeamGroupKind = schema.GroupKind{Group: Group, Kind: TeamKind}.String()

var TeamGroupVersionKind = SchemeGroupVersion.WithKind(TeamKind)

var PipelineKind = reflect.TypeFor[Pipeline]().Name()

var PipelineGroupVersionKind = SchemeGroupVersion.WithKind(PipelineKind)
