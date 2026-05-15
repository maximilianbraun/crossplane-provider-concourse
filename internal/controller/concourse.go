package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/maximilianbraun/crossplane-provider-concourse/internal/clients"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/build"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/job"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/pipeline"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/providerconfig"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/resource"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/team"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/controller/worker"
)

// Setup registers all controllers with the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	if err := providerconfig.Setup(mgr); err != nil {
		return err
	}

	for _, setup := range []func(ctrl.Manager, *clients.Cache) error{
		team.Setup,
		pipeline.Setup,
		job.Setup,
		build.Setup,
		worker.Setup,
		resource.Setup,
	} {
		if err := setup(mgr, cache); err != nil {
			return err
		}
	}
	return nil
}
