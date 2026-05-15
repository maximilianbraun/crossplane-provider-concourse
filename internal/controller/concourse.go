package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/maximilianbraun/crossplane-provider-concourse/internal/clients"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/build"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/job"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/pipeline"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/team"
)

// Setup registers all controllers with the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	for _, setup := range []func(ctrl.Manager, *clients.Cache) error{
		team.Setup,
		pipeline.Setup,
		job.Setup,
		build.Setup,
	} {
		if err := setup(mgr, cache); err != nil {
			return err
		}
	}
	return nil
}
