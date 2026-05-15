package build

import (
	"context"
	"fmt"
	"time"

	"github.com/concourse/concourse/atc"
	goconcourse "github.com/concourse/concourse/go-concourse/concourse"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	civ1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/ci/v1alpha1"
	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/clients"
)

// Setup adds the Build controller to the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("build").
		For(&civ1alpha1.Build{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(civ1alpha1.SchemeGroupVersion.WithKind("Build")),
			managed.WithTypedExternalConnector[*civ1alpha1.Build](&connector{
				kube:  mgr.GetClient(),
				cache: cache,
			}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *civ1alpha1.Build) (managed.TypedExternalClient[*civ1alpha1.Build], error) {
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

func (e *external) Observe(ctx context.Context, mg *civ1alpha1.Build) (managed.ExternalObservation, error) {
	buildID := mg.Status.AtProvider.BuildID

	if buildID == 0 {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	b, found, err := e.concourse.Build(fmt.Sprintf("%d", buildID))
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("getting build %d: %w", buildID, err)
	}
	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	mg.Status.AtProvider.Status = string(b.Status)
	mg.Status.AtProvider.BuildName = b.Name
	if b.StartTime > 0 {
		t := metav1.NewTime(time.Unix(b.StartTime, 0))
		mg.Status.AtProvider.StartTime = &t
	}
	if b.EndTime > 0 {
		t := metav1.NewTime(time.Unix(b.EndTime, 0))
		mg.Status.AtProvider.EndTime = &t
	}

	if isTerminal(b.Status) {
		mg.SetConditions(xpv1.Available())
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
	}

	if mg.Spec.ForProvider.Abort {
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, nil
	}

	mg.SetConditions(xpv1.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *external) Create(ctx context.Context, mg *civ1alpha1.Build) (managed.ExternalCreation, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := mg.Spec.ForProvider.PipelineName
	jobName := mg.Spec.ForProvider.JobName

	b, err := e.concourse.Team(teamName).CreateJobBuild(
		atc.PipelineRef{Name: pipelineName},
		jobName,
	)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("triggering build for %s/%s/%s: %w", teamName, pipelineName, jobName, err)
	}

	mg.Status.AtProvider.BuildID = b.ID
	mg.Status.AtProvider.BuildName = b.Name
	mg.Status.AtProvider.Status = string(b.Status)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *civ1alpha1.Build) (managed.ExternalUpdate, error) {
	if !mg.Spec.ForProvider.Abort {
		return managed.ExternalUpdate{}, nil
	}

	buildID := mg.Status.AtProvider.BuildID
	if err := e.concourse.AbortBuild(fmt.Sprintf("%d", buildID)); err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("aborting build %d: %w", buildID, err)
	}

	mg.Status.AtProvider.Status = "aborted"
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *civ1alpha1.Build) (managed.ExternalDelete, error) {
	if mg.Status.AtProvider.BuildID != 0 && !isTerminal(atc.BuildStatus(mg.Status.AtProvider.Status)) {
		_ = e.concourse.AbortBuild(fmt.Sprintf("%d", mg.Status.AtProvider.BuildID))
	}
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func isTerminal(status atc.BuildStatus) bool {
	switch status {
	case atc.StatusSucceeded, atc.StatusFailed, atc.StatusErrored, atc.StatusAborted:
		return true
	}
	return false
}
