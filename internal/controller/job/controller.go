package job

import (
	"context"
	"fmt"

	"github.com/concourse/concourse/atc"
	goconcourse "github.com/concourse/concourse/go-concourse/concourse"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	civ1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/ci/v1alpha1"
	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/clients"
)

// Setup adds the Job controller to the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("job").
		For(&civ1alpha1.Job{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(civ1alpha1.SchemeGroupVersion.WithKind("Job")),
			managed.WithTypedExternalConnector[*civ1alpha1.Job](&connector{
				kube:  mgr.GetClient(),
				cache: cache,
			}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *civ1alpha1.Job) (managed.TypedExternalClient[*civ1alpha1.Job], error) {
	pc := &providerv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, fmt.Errorf("getting ProviderConfig: %w", err)
	}

	cl, err := c.cache.GetOrBuild(ctx, c.kube, pc)
	if err != nil {
		return nil, fmt.Errorf("building Concourse client: %w", err)
	}

	return &external{concourse: cl}, nil
}

type external struct {
	concourse goconcourse.Client
}

func (e *external) Observe(ctx context.Context, mg *civ1alpha1.Job) (managed.ExternalObservation, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := mg.Spec.ForProvider.PipelineName
	jobName := mg.Spec.ForProvider.JobName

	job, found, err := e.concourse.Team(teamName).Job(atc.PipelineRef{Name: pipelineName}, jobName)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("getting job: %w", err)
	}
	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	mg.Status.AtProvider.ID = job.ID
	mg.Status.AtProvider.Paused = job.Paused
	if job.FinishedBuild != nil {
		mg.Status.AtProvider.FinishedBuild = job.FinishedBuild.ID
	}
	if job.NextBuild != nil {
		mg.Status.AtProvider.NextBuild = job.NextBuild.ID
	}

	upToDate := true
	if mg.Spec.ForProvider.Paused != nil && *mg.Spec.ForProvider.Paused != job.Paused {
		upToDate = false
	}

	mg.SetConditions(xpv1.Available())
	meta.SetExternalName(mg, jobName)

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
}

func (e *external) Create(ctx context.Context, mg *civ1alpha1.Job) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *civ1alpha1.Job) (managed.ExternalUpdate, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := mg.Spec.ForProvider.PipelineName
	jobName := mg.Spec.ForProvider.JobName

	ref := atc.PipelineRef{Name: pipelineName}

	if mg.Spec.ForProvider.Paused != nil {
		if *mg.Spec.ForProvider.Paused {
			if _, err := e.concourse.Team(teamName).PauseJob(ref, jobName); err != nil {
				return managed.ExternalUpdate{}, fmt.Errorf("pausing job: %w", err)
			}
		} else {
			if _, err := e.concourse.Team(teamName).UnpauseJob(ref, jobName); err != nil {
				return managed.ExternalUpdate{}, fmt.Errorf("unpausing job: %w", err)
			}
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *civ1alpha1.Job) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }
