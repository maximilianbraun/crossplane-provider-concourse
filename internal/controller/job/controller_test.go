package job

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
	name    string
	jobs    map[string]atc.Job
	paused  []string
	resumed []string
}

func (m *mockTeam) Job(ref atc.PipelineRef, jobName string) (atc.Job, bool, error) {
	key := ref.Name + "/" + jobName
	j, ok := m.jobs[key]
	return j, ok, nil
}

func (m *mockTeam) PauseJob(ref atc.PipelineRef, jobName string) (bool, error) {
	m.paused = append(m.paused, ref.Name+"/"+jobName)
	return true, nil
}

func (m *mockTeam) UnpauseJob(ref atc.PipelineRef, jobName string) (bool, error) {
	m.resumed = append(m.resumed, ref.Name+"/"+jobName)
	return true, nil
}

func newJobMR(team, pipeline, job string, paused *bool) *civ1alpha1.Job { //nolint:unparam
	return &civ1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job"},
		Spec: civ1alpha1.JobSpec{
			ForProvider: civ1alpha1.JobParameters{
				TeamName:     team,
				PipelineName: pipeline,
				JobName:      job,
				Paused:       paused,
			},
		},
	}
}

func TestObserve_JobFound(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {
				name: "team",
				jobs: map[string]atc.Job{
					"pipeline/my-job": {ID: 10, Name: "my-job", Paused: false},
				},
			},
		},
	}
	e := &external{concourse: mc}
	mg := newJobMR("team", "pipeline", "my-job", ptr.To(false))

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true when paused matches")
	}
	if mg.Status.AtProvider.ID != 10 {
		t.Fatalf("expected ID=10, got %d", mg.Status.AtProvider.ID)
	}
}

func TestObserve_JobNotFound(t *testing.T) {
	mc := &mockClient{
		teamCalls: map[string]*mockTeam{
			"team": {name: "team", jobs: map[string]atc.Job{}},
		},
	}
	e := &external{concourse: mc}
	mg := newJobMR("team", "pipeline", "my-job", nil)

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
				jobs: map[string]atc.Job{
					"pipeline/my-job": {ID: 10, Name: "my-job", Paused: false},
				},
			},
		},
	}
	e := &external{concourse: mc}
	mg := newJobMR("team", "pipeline", "my-job", ptr.To(true))

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false when pause state differs")
	}
}

func TestUpdate_PauseJob(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newJobMR("team", "pipeline", "my-job", ptr.To(true))

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["team"]
	if len(mt.paused) != 1 || mt.paused[0] != "pipeline/my-job" {
		t.Fatalf("expected PauseJob, got paused=%v", mt.paused)
	}
}

func TestUpdate_UnpauseJob(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newJobMR("team", "pipeline", "my-job", ptr.To(false))

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mt := mc.teamCalls["team"]
	if len(mt.resumed) != 1 || mt.resumed[0] != "pipeline/my-job" {
		t.Fatalf("expected UnpauseJob, got resumed=%v", mt.resumed)
	}
}
