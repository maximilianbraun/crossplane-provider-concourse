package pipeline

import (
	"context"
	"testing"

	"github.com/concourse/concourse/atc"
	goconcourse "github.com/concourse/concourse/go-concourse/concourse"
	"k8s.io/utils/ptr"

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
	pipelines map[string]atc.Pipeline
	configs   [][]byte
	deleted   []string
	paused    []string
	unpaused  []string
	exposed   []string
	hidden    []string
}

func (m *mockTeam) Pipeline(ref atc.PipelineRef) (atc.Pipeline, bool, error) {
	p, ok := m.pipelines[ref.Name]
	return p, ok, nil
}

func (m *mockTeam) CreateOrUpdatePipelineConfig(ref atc.PipelineRef, _ string, config []byte, _ bool) (bool, bool, []goconcourse.ConfigWarning, error) {
	m.configs = append(m.configs, config)
	return true, false, nil, nil
}

func (m *mockTeam) DeletePipeline(ref atc.PipelineRef) (bool, error) {
	m.deleted = append(m.deleted, ref.Name)
	return true, nil
}

func (m *mockTeam) PausePipeline(ref atc.PipelineRef) (bool, error) {
	m.paused = append(m.paused, ref.Name)
	return true, nil
}

func (m *mockTeam) UnpausePipeline(ref atc.PipelineRef) (bool, error) {
	m.unpaused = append(m.unpaused, ref.Name)
	return true, nil
}

func (m *mockTeam) ExposePipeline(ref atc.PipelineRef) (bool, error) {
	m.exposed = append(m.exposed, ref.Name)
	return true, nil
}

func (m *mockTeam) HidePipeline(ref atc.PipelineRef) (bool, error) {
	m.hidden = append(m.hidden, ref.Name)
	return true, nil
}

func newPipelineMR(team, pipeline, inlineConfig string, paused, exposed *bool) *civ1alpha1.Pipeline { //nolint:unparam
	return &civ1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pipeline"},
		Spec: civ1alpha1.PipelineSpec{
			ForProvider: civ1alpha1.PipelineParameters{
				TeamName:     team,
				PipelineName: pipeline,
				Config:       civ1alpha1.PipelineConfigSource{Inline: inlineConfig},
				Paused:       paused,
				Exposed:      exposed,
			},
		},
	}
}

func TestObserve_PipelineFound(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {
				name: "team",
				pipelines: map[string]atc.Pipeline{
					"my-pipeline": {ID: 5, Name: "my-pipeline", Paused: false, Public: true},
				},
			},
		},
	}
	e := &external{concourse: mc, kube: nil}
	mg := newPipelineMR("team", "my-pipeline", "jobs: []", ptr.To(false), ptr.To(true))

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true when paused/exposed match")
	}
	if mg.Status.AtProvider.ID != 5 {
		t.Fatalf("expected ID=5, got %d", mg.Status.AtProvider.ID)
	}
}

func TestObserve_PipelineNotFound(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {name: "team", pipelines: map[string]atc.Pipeline{}},
		},
	}
	e := &external{concourse: mc, kube: nil}
	mg := newPipelineMR("team", "my-pipeline", "jobs: []", nil, nil)

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}
}

func TestObserve_PauseDrift(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {
				name: "team",
				pipelines: map[string]atc.Pipeline{
					"my-pipeline": {ID: 5, Name: "my-pipeline", Paused: true, Public: false},
				},
			},
		},
	}
	e := &external{concourse: mc, kube: nil}
	mg := newPipelineMR("team", "my-pipeline", "jobs: []", ptr.To(false), nil)

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false when paused state differs")
	}
}

func TestCreate_InlineConfig(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc, kube: nil}
	mg := newPipelineMR("team", "my-pipeline", "jobs:\n- name: test", ptr.To(true), ptr.To(true))

	_, err := e.Create(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["team"]
	if len(mt.configs) != 1 {
		t.Fatal("expected CreateOrUpdatePipelineConfig to be called")
	}
	if string(mt.configs[0]) != "jobs:\n- name: test" {
		t.Fatalf("unexpected config: %s", mt.configs[0])
	}
	if len(mt.paused) != 1 {
		t.Fatal("expected PausePipeline to be called")
	}
	if len(mt.exposed) != 1 {
		t.Fatal("expected ExposePipeline to be called")
	}
}

func TestDelete_Pipeline(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc, kube: nil}
	mg := newPipelineMR("team", "my-pipeline", "", nil, nil)

	_, err := e.Delete(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["team"]
	if len(mt.deleted) != 1 || mt.deleted[0] != "my-pipeline" {
		t.Fatalf("expected DeletePipeline(my-pipeline), got %v", mt.deleted)
	}
}

func TestUpdate_SyncsPauseAndExpose(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc, kube: nil}
	mg := newPipelineMR("team", "my-pipeline", "jobs: []", ptr.To(false), ptr.To(false))

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["team"]
	if len(mt.unpaused) != 1 {
		t.Fatal("expected UnpausePipeline to be called")
	}
	if len(mt.hidden) != 1 {
		t.Fatal("expected HidePipeline to be called")
	}
}
