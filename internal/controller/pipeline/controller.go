package pipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/concourse/concourse/atc"
	goconcourse "github.com/concourse/concourse/go-concourse/concourse"
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	civ1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/ci/v1alpha1"
	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
	"github.com/maximilianbraun/crossplane-provider-concourse/internal/clients"
)

// Setup adds the Pipeline controller to the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("pipeline").
		For(&civ1alpha1.Pipeline{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(civ1alpha1.PipelineGroupVersionKind),
			managed.WithTypedExternalConnector[*civ1alpha1.Pipeline](&connector{
				kube:  mgr.GetClient(),
				cache: cache,
			}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *civ1alpha1.Pipeline) (managed.TypedExternalClient[*civ1alpha1.Pipeline], error) {
	pc := &providerv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: mg.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, fmt.Errorf("getting ProviderConfig: %w", err)
	}

	cl, err := c.cache.GetOrBuild(ctx, c.kube, pc)
	if err != nil {
		return nil, fmt.Errorf("building Concourse client: %w", err)
	}

	return &external{concourse: cl, kube: c.kube}, nil
}

type external struct {
	concourse goconcourse.Client
	kube      client.Client
}

func (e *external) Observe(ctx context.Context, mg *civ1alpha1.Pipeline) (managed.ExternalObservation, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := pipelineExternalName(mg)

	pipeline, found, err := e.concourse.Team(teamName).Pipeline(atc.PipelineRef{Name: pipelineName})
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("getting pipeline: %w", err)
	}
	if !found {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	mg.Status.AtProvider.ID = pipeline.ID
	mg.Status.AtProvider.Paused = pipeline.Paused
	mg.Status.AtProvider.Exposed = pipeline.Public

	desiredConfig, err := e.resolveConfig(ctx, mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	configHash := sha256Hash(desiredConfig)
	mg.Status.AtProvider.ConfigHash = configHash

	upToDate := true
	if mg.Spec.ForProvider.Paused != nil && *mg.Spec.ForProvider.Paused != pipeline.Paused {
		upToDate = false
	}
	if mg.Spec.ForProvider.Exposed != nil && *mg.Spec.ForProvider.Exposed != pipeline.Public {
		upToDate = false
	}

	mg.SetConditions(xpv1.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
}

func (e *external) Create(ctx context.Context, mg *civ1alpha1.Pipeline) (managed.ExternalCreation, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := pipelineExternalName(mg)

	config, err := e.resolveConfig(ctx, mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	_, _, _, err = e.concourse.Team(teamName).CreateOrUpdatePipelineConfig(
		atc.PipelineRef{Name: pipelineName},
		"0",
		[]byte(config),
		false,
	)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("creating pipeline: %w", err)
	}

	meta.SetExternalName(mg, pipelineName)

	if err := e.syncPauseExpose(mg, teamName, pipelineName); err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *civ1alpha1.Pipeline) (managed.ExternalUpdate, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := pipelineExternalName(mg)

	config, err := e.resolveConfig(ctx, mg)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	_, _, _, err = e.concourse.Team(teamName).CreateOrUpdatePipelineConfig(
		atc.PipelineRef{Name: pipelineName},
		"0",
		[]byte(config),
		false,
	)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("updating pipeline config: %w", err)
	}

	if err := e.syncPauseExpose(mg, teamName, pipelineName); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *civ1alpha1.Pipeline) (managed.ExternalDelete, error) {
	teamName := mg.Spec.ForProvider.TeamName
	pipelineName := pipelineExternalName(mg)

	_, err := e.concourse.Team(teamName).DeletePipeline(atc.PipelineRef{Name: pipelineName})
	if err != nil {
		return managed.ExternalDelete{}, fmt.Errorf("deleting pipeline: %w", err)
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func (e *external) syncPauseExpose(mg *civ1alpha1.Pipeline, teamName, pipelineName string) error {
	ref := atc.PipelineRef{Name: pipelineName}
	team := e.concourse.Team(teamName)

	if mg.Spec.ForProvider.Paused != nil {
		if *mg.Spec.ForProvider.Paused {
			if _, err := team.PausePipeline(ref); err != nil {
				return fmt.Errorf("pausing pipeline: %w", err)
			}
		} else {
			if _, err := team.UnpausePipeline(ref); err != nil {
				return fmt.Errorf("unpausing pipeline: %w", err)
			}
		}
	}

	if mg.Spec.ForProvider.Exposed != nil {
		if *mg.Spec.ForProvider.Exposed {
			if _, err := team.ExposePipeline(ref); err != nil {
				return fmt.Errorf("exposing pipeline: %w", err)
			}
		} else {
			if _, err := team.HidePipeline(ref); err != nil {
				return fmt.Errorf("hiding pipeline: %w", err)
			}
		}
	}

	return nil
}

func (e *external) resolveConfig(ctx context.Context, mg *civ1alpha1.Pipeline) (string, error) {
	cfg := mg.Spec.ForProvider.Config

	if cfg.Inline != "" {
		return cfg.Inline, nil
	}

	if cfg.ConfigMapRef != nil {
		cm := &corev1.ConfigMap{}
		nn := types.NamespacedName{
			Namespace: cfg.ConfigMapRef.SecretReference.Namespace,
			Name:      cfg.ConfigMapRef.SecretReference.Name,
		}
		if err := e.kube.Get(ctx, nn, cm); err != nil {
			return "", fmt.Errorf("getting pipeline configmap: %w", err)
		}
		data, ok := cm.Data[cfg.ConfigMapRef.Key]
		if !ok {
			return "", fmt.Errorf("key %q not found in configmap %s", cfg.ConfigMapRef.Key, nn)
		}
		return data, nil
	}

	return "", fmt.Errorf("no pipeline config source specified")
}

func pipelineExternalName(mg *civ1alpha1.Pipeline) string {
	if en := meta.GetExternalName(mg); en != "" && en != mg.Name {
		return en
	}
	if mg.Spec.ForProvider.PipelineName != "" {
		return mg.Spec.ForProvider.PipelineName
	}
	return mg.Name
}

func sha256Hash(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}
