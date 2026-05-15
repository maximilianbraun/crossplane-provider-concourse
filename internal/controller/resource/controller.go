package resource

import (
	"context"
	"fmt"

	"github.com/concourse/concourse/atc"
	goconcourse "github.com/concourse/concourse/go-concourse/concourse"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	civ1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/ci/v1alpha1"
	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/clients"
)

func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("resource").
		For(&civ1alpha1.PipelineResource{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(civ1alpha1.SchemeGroupVersion.WithKind("PipelineResource")),
			managed.WithTypedExternalConnector[*civ1alpha1.PipelineResource](&connector{
				kube:  mgr.GetClient(),
				cache: cache,
			}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *civ1alpha1.PipelineResource) (managed.TypedExternalClient[*civ1alpha1.PipelineResource], error) {
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

func (e *external) Observe(ctx context.Context, mg *civ1alpha1.PipelineResource) (managed.ExternalObservation, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := mg.Spec.ForProvider.PipelineName
	resourceName := mg.Spec.ForProvider.ResourceName

	r, found, err := e.concourse.Team(teamName).Resource(atc.PipelineRef{Name: pipelineName}, resourceName)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("getting resource %s/%s/%s: %w", teamName, pipelineName, resourceName, err)
	}
	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	mg.Status.AtProvider.Type = r.Type
	if r.PinnedVersion != nil {
		mg.Status.AtProvider.PinnedVersion = r.PinnedVersion
	} else {
		mg.Status.AtProvider.PinnedVersion = nil
	}

	upToDate := pinnedVersionMatches(mg.Spec.ForProvider.PinnedVersion, r.PinnedVersion)
	mg.SetConditions(xpv1.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (e *external) Create(_ context.Context, _ *civ1alpha1.PipelineResource) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *civ1alpha1.PipelineResource) (managed.ExternalUpdate, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := mg.Spec.ForProvider.PipelineName
	resourceName := mg.Spec.ForProvider.ResourceName

	if len(mg.Spec.ForProvider.PinnedVersion) == 0 {
		if _, err := e.concourse.Team(teamName).UnpinResource(
			atc.PipelineRef{Name: pipelineName},
			resourceName,
		); err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("unpinning resource %s/%s/%s: %w", teamName, pipelineName, resourceName, err)
		}
	}
	// Pinning by version map requires resolving the version ID first via ResourceVersions.
	// For now, only unpin is supported. Pin support requires looking up the version ID.

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *civ1alpha1.PipelineResource) (managed.ExternalDelete, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := mg.Spec.ForProvider.PipelineName
	resourceName := mg.Spec.ForProvider.ResourceName

	_, _ = e.concourse.Team(teamName).UnpinResource(
		atc.PipelineRef{Name: pipelineName},
		resourceName,
	)
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func pinnedVersionMatches(desired, actual atc.Version) bool {
	if len(desired) == 0 && len(actual) == 0 {
		return true
	}
	if len(desired) != len(actual) {
		return false
	}
	for k, v := range desired {
		if actual[k] != v {
			return false
		}
	}
	return true
}
