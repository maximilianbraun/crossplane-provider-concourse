package resource

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
	teamCalls map[string]*mockTeam
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
	resources map[string]atc.Resource
	unpinned  []string
}

func (m *mockTeam) Resource(ref atc.PipelineRef, resourceName string) (atc.Resource, bool, error) {
	key := ref.Name + "/" + resourceName
	r, ok := m.resources[key]
	return r, ok, nil
}

func (m *mockTeam) UnpinResource(ref atc.PipelineRef, resourceName string) (bool, error) {
	m.unpinned = append(m.unpinned, ref.Name+"/"+resourceName)
	return true, nil
}

func newResourceMR(team, pipeline, resource string, pinned map[string]string) *civ1alpha1.PipelineResource {
	return &civ1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{Name: "test-resource"},
		Spec: civ1alpha1.ResourceSpec{
			ForProvider: civ1alpha1.ResourceParameters{
				TeamName:      team,
				PipelineName:  pipeline,
				ResourceName:  resource,
				PinnedVersion: pinned,
			},
		},
	}
}

func TestObserve_ResourceFound(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {
				name: "team",
				resources: map[string]atc.Resource{
					"pipeline/my-resource": {
						Name:          "my-resource",
						Type:          "git",
						PinnedVersion: atc.Version{"ref": "abc123"},
					},
				},
			},
		},
	}
	e := &external{concourse: mc}
	mg := newResourceMR("team", "pipeline", "my-resource", map[string]string{"ref": "abc123"})

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true when pinned versions match")
	}
	if mg.Status.AtProvider.Type != "git" {
		t.Fatalf("expected type 'git', got %q", mg.Status.AtProvider.Type)
	}
}

func TestObserve_ResourceNotFound(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {name: "team", resources: map[string]atc.Resource{}},
		},
	}
	e := &external{concourse: mc}
	mg := newResourceMR("team", "pipeline", "my-resource", nil)

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}
}

func TestObserve_PinDrift(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {
				name: "team",
				resources: map[string]atc.Resource{
					"pipeline/my-resource": {
						Name:          "my-resource",
						Type:          "git",
						PinnedVersion: atc.Version{"ref": "old-sha"},
					},
				},
			},
		},
	}
	e := &external{concourse: mc}
	mg := newResourceMR("team", "pipeline", "my-resource", map[string]string{"ref": "new-sha"})

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false when pinned versions differ")
	}
}

func TestUpdate_Unpin(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newResourceMR("team", "pipeline", "my-resource", nil)

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["team"]
	if len(mt.unpinned) != 1 || mt.unpinned[0] != "pipeline/my-resource" {
		t.Fatalf("expected UnpinResource(pipeline/my-resource), got %v", mt.unpinned)
	}
}

func TestDelete_Unpins(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newResourceMR("team", "pipeline", "my-resource", map[string]string{"ref": "abc"})

	_, err := e.Delete(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["team"]
	if len(mt.unpinned) != 1 {
		t.Fatal("expected UnpinResource to be called on delete")
	}
}
