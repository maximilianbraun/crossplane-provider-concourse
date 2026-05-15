package team

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

// Setup adds the Team controller to the Manager.
func Setup(mgr ctrl.Manager, cache *clients.Cache) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("team").
		For(&civ1alpha1.Team{}).
		Complete(managed.NewReconciler(mgr,
			xpresource.ManagedKind(civ1alpha1.TeamGroupVersionKind),
			managed.WithTypedExternalConnector[*civ1alpha1.Team](&connector{
				kube:  mgr.GetClient(),
				cache: cache,
			}),
		))
}

type connector struct {
	kube  client.Client
	cache *clients.Cache
}

func (c *connector) Connect(ctx context.Context, mg *civ1alpha1.Team) (managed.TypedExternalClient[*civ1alpha1.Team], error) {
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

func (e *external) Observe(ctx context.Context, mg *civ1alpha1.Team) (managed.ExternalObservation, error) {
	teamName := externalName(mg)

	teams, err := e.concourse.ListTeams()
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("listing teams: %w", err)
	}

	for _, t := range teams {
		if t.Name == teamName {
			mg.Status.AtProvider.ID = t.ID

			upToDate := rolesMatch(mg.Spec.ForProvider.Roles, t.Auth)
			mg.SetConditions(xpv1.Available())

			return managed.ExternalObservation{
				ResourceExists:   true,
				ResourceUpToDate: upToDate,
			}, nil
		}
	}

	return managed.ExternalObservation{
		ResourceExists: false,
	}, nil
}

func (e *external) Create(ctx context.Context, mg *civ1alpha1.Team) (managed.ExternalCreation, error) {
	teamName := externalName(mg)

	team := atc.Team{
		Name: teamName,
		Auth: buildAuth(mg.Spec.ForProvider.Roles),
	}

	_, _, _, _, err := e.concourse.Team(teamName).CreateOrUpdate(team)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("creating team %q: %w", teamName, err)
	}

	meta.SetExternalName(mg, teamName)
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg *civ1alpha1.Team) (managed.ExternalUpdate, error) {
	teamName := externalName(mg)

	team := atc.Team{
		Name: teamName,
		Auth: buildAuth(mg.Spec.ForProvider.Roles),
	}

	_, _, _, _, err := e.concourse.Team(teamName).CreateOrUpdate(team)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("updating team %q: %w", teamName, err)
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg *civ1alpha1.Team) (managed.ExternalDelete, error) {
	teamName := externalName(mg)

	if err := e.concourse.Team(teamName).DestroyTeam(teamName); err != nil {
		return managed.ExternalDelete{}, fmt.Errorf("deleting team %q: %w", teamName, err)
	}

	return managed.ExternalDelete{}, nil
}

func (e *external) Disconnect(_ context.Context) error { return nil }

func externalName(mg *civ1alpha1.Team) string {
	if en := meta.GetExternalName(mg); en != "" && en != mg.Name {
		return en
	}
	if mg.Spec.ForProvider.TeamName != "" {
		return mg.Spec.ForProvider.TeamName
	}
	return mg.Name
}

func buildAuth(roles []civ1alpha1.TeamRole) atc.TeamAuth {
	auth := atc.TeamAuth{}
	for _, r := range roles {
		auth[r.Name] = map[string][]string{
			"users":  r.Users,
			"groups": r.Groups,
		}
	}
	return auth
}

func rolesMatch(desired []civ1alpha1.TeamRole, actual atc.TeamAuth) bool {
	desiredAuth := buildAuth(desired)
	if len(desiredAuth) != len(actual) {
		return false
	}
	for role, dMap := range desiredAuth {
		aMap, ok := actual[role]
		if !ok {
			return false
		}
		if !stringSliceEqual(dMap["users"], aMap["users"]) || !stringSliceEqual(dMap["groups"], aMap["groups"]) {
			return false
		}
	}
	return true
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, s := range a {
		set[s] = struct{}{}
	}
	for _, s := range b {
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}
