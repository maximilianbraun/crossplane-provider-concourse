package worker

import (
	"context"
	"fmt"

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
		Named("worker").
		For(&civ1alpha1.Worker{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(civ1alpha1.SchemeGroupVersion.WithKind("Worker")),
			managed.WithTypedExternalConnector[*civ1alpha1.Worker](&connector{
				kube:  mgr.GetClient(),
				cache: cache,
			}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *civ1alpha1.Worker) (managed.TypedExternalClient[*civ1alpha1.Worker], error) {
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

func (e *external) Observe(ctx context.Context, mg *civ1alpha1.Worker) (managed.ExternalObservation, error) {
	workers, err := e.concourse.ListWorkers()
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("listing workers: %w", err)
	}

	for _, w := range workers {
		if w.Name == mg.Spec.ForProvider.WorkerName {
			mg.Status.AtProvider.State = string(w.State)
			mg.Status.AtProvider.Version = w.Version
			mg.Status.AtProvider.Platform = w.Platform
			mg.Status.AtProvider.ActiveContainers = w.ActiveContainers
			mg.Status.AtProvider.ActiveVolumes = w.ActiveVolumes

			upToDate := true
			if mg.Spec.ForProvider.DesiredState != "" && mg.Spec.ForProvider.DesiredState != string(w.State) {
				upToDate = false
			}

			mg.SetConditions(xpv1.Available())
			return managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: upToDate,
			}, nil
		}
	}

	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (e *external) Create(_ context.Context, _ *civ1alpha1.Worker) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *civ1alpha1.Worker) (managed.ExternalUpdate, error) {
	switch mg.Spec.ForProvider.DesiredState {
	case "landing":
		if err := e.concourse.LandWorker(mg.Spec.ForProvider.WorkerName); err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("landing worker %q: %w", mg.Spec.ForProvider.WorkerName, err)
		}
	case "retiring":
		if err := e.concourse.PruneWorker(mg.Spec.ForProvider.WorkerName); err != nil {
			return managed.ExternalUpdate{}, fmt.Errorf("retiring worker %q: %w", mg.Spec.ForProvider.WorkerName, err)
		}
	}
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *civ1alpha1.Worker) (managed.ExternalDelete, error) {
	if err := e.concourse.PruneWorker(mg.Spec.ForProvider.WorkerName); err != nil {
		return managed.ExternalDelete{}, fmt.Errorf("pruning worker %q: %w", mg.Spec.ForProvider.WorkerName, err)
	}
	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }
