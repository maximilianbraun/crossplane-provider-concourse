package team

import (
	"context"
	"testing"

	"github.com/concourse/concourse/atc"
	goconcourse "github.com/concourse/concourse/go-concourse/concourse"

	civ1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/ci/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockClient struct {
	goconcourse.Client
	teams     []atc.Team
	teamCalls map[string]*mockTeam
}

func (m *mockClient) ListTeams() ([]atc.Team, error) {
	return m.teams, nil
}

func (m *mockClient) Team(name string) goconcourse.Team {
	if m.teamCalls == nil {
		m.teamCalls = map[string]*mockTeam{}
	}
	if _, ok := m.teamCalls[name]; !ok {
		m.teamCalls[name] = &mockTeam{name: name}
	}
	return m.teamCalls[name]
}

type mockTeam struct {
	goconcourse.Team
	name      string
	created   *atc.Team
	destroyed bool
}

func (m *mockTeam) CreateOrUpdate(team atc.Team) (atc.Team, bool, bool, []goconcourse.ConfigWarning, error) {
	m.created = &team
	return team, true, false, nil, nil
}

func (m *mockTeam) DestroyTeam(name string) error {
	m.destroyed = true
	return nil
}

func newTeamMR(name, teamName string) *civ1alpha1.Team { //nolint:unparam
	return &civ1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: civ1alpha1.TeamSpec{
			ForProvider: civ1alpha1.TeamParameters{
				TeamName: teamName,
				Roles: []civ1alpha1.TeamRole{
					{Name: "owner", Users: []string{"admin"}},
				},
			},
		},
	}
}

func TestObserve_TeamExists(t *testing.T) {
	mc := &mockClient{
		teams: []atc.Team{
			{ID: 1, Name: "my-team", Auth: atc.TeamAuth{
				"owner": {"users": {"admin"}, "groups": {}},
			}},
		},
	}
	e := &external{concourse: mc}
	mg := newTeamMR("my-team", "my-team")

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if mg.Status.AtProvider.ID != 1 {
		t.Fatalf("expected ID=1, got %d", mg.Status.AtProvider.ID)
	}
}

func TestObserve_TeamNotFound(t *testing.T) {
	mc := &mockClient{teams: []atc.Team{}}
	e := &external{concourse: mc}
	mg := newTeamMR("my-team", "my-team")

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}
}

func TestObserve_RoleDrift(t *testing.T) {
	mc := &mockClient{
		teams: []atc.Team{
			{ID: 1, Name: "my-team", Auth: atc.TeamAuth{
				"owner": {"users": {"someone-else"}, "groups": {}},
			}},
		},
	}
	e := &external{concourse: mc}
	mg := newTeamMR("my-team", "my-team")

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false due to role drift")
	}
}

func TestCreate(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newTeamMR("my-team", "my-team")

	_, err := e.Create(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["my-team"]
	if mt == nil || mt.created == nil {
		t.Fatal("expected CreateOrUpdate to be called")
	}
	if mt.created.Name != "my-team" {
		t.Fatalf("expected team name 'my-team', got %q", mt.created.Name)
	}
}

func TestDelete(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newTeamMR("my-team", "my-team")

	_, err := e.Delete(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["my-team"]
	if mt == nil || !mt.destroyed {
		t.Fatal("expected DestroyTeam to be called")
	}
}
