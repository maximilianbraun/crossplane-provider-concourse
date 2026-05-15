package build

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
	builds    map[string]atc.Build
	aborted   []string
	teamCalls map[string]*mockTeam
}

func (m *mockClient) Build(buildID string) (atc.Build, bool, error) {
	b, ok := m.builds[buildID]
	return b, ok, nil
}

func (m *mockClient) AbortBuild(buildID string) error {
	m.aborted = append(m.aborted, buildID)
	return nil
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
	buildID int
}

func (m *mockTeam) CreateJobBuild(ref atc.PipelineRef, jobName string) (atc.Build, error) {
	m.buildID++
	return atc.Build{
		ID:     m.buildID,
		Name:   "1",
		Status: atc.StatusPending,
	}, nil
}

func newBuildMR(team, pipeline, job string) *civ1alpha1.Build { //nolint:unparam
	return &civ1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{Name: "test-build"},
		Spec: civ1alpha1.BuildSpec{
			ForProvider: civ1alpha1.BuildParameters{
				TeamName:     team,
				PipelineName: pipeline,
				JobName:      job,
			},
		},
	}
}

func TestObserve_NoBuildID(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false when buildID=0")
	}
}

func TestObserve_BuildRunning(t *testing.T) {
	mc := &mockClient{
		builds: map[string]atc.Build{
			"42": {ID: 42, Name: "1", Status: atc.StatusStarted, StartTime: 1000},
		},
	}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")
	mg.Status.AtProvider.BuildID = 42

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true for running build without abort")
	}
	if mg.Status.AtProvider.Status != "started" {
		t.Fatalf("expected status 'started', got %q", mg.Status.AtProvider.Status)
	}
}

func TestObserve_BuildTerminal(t *testing.T) {
	mc := &mockClient{
		builds: map[string]atc.Build{
			"42": {ID: 42, Name: "1", Status: atc.StatusSucceeded, StartTime: 1000, EndTime: 2000},
		},
	}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")
	mg.Status.AtProvider.BuildID = 42

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Fatal("terminal build should be exists=true, upToDate=true")
	}
}

func TestObserve_AbortRequested(t *testing.T) {
	mc := &mockClient{
		builds: map[string]atc.Build{
			"42": {ID: 42, Name: "1", Status: atc.StatusStarted},
		},
	}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")
	mg.Status.AtProvider.BuildID = 42
	mg.Spec.ForProvider.Abort = true

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false when abort is requested on running build")
	}
}

func TestCreate_TriggersBuild(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")

	_, err := e.Create(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mg.Status.AtProvider.BuildID == 0 {
		t.Fatal("expected buildID to be set after Create")
	}
	if mg.Status.AtProvider.Status != "pending" {
		t.Fatalf("expected status 'pending', got %q", mg.Status.AtProvider.Status)
	}
}

func TestUpdate_AbortsRunningBuild(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")
	mg.Status.AtProvider.BuildID = 42
	mg.Spec.ForProvider.Abort = true

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.aborted) != 1 || mc.aborted[0] != "42" {
		t.Fatalf("expected AbortBuild(42), got %v", mc.aborted)
	}
}

func TestUpdate_NoAbortIfNotRequested(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")
	mg.Status.AtProvider.BuildID = 42

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.aborted) != 0 {
		t.Fatal("expected no abort call")
	}
}

func TestDelete_AbortsIfRunning(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")
	mg.Status.AtProvider.BuildID = 42
	mg.Status.AtProvider.Status = "started"

	_, err := e.Delete(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.aborted) != 1 {
		t.Fatal("expected abort on delete of running build")
	}
}

func TestDelete_NoAbortIfTerminal(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newBuildMR("team", "pipeline", "job")
	mg.Status.AtProvider.BuildID = 42
	mg.Status.AtProvider.Status = "succeeded"

	_, err := e.Delete(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.aborted) != 0 {
		t.Fatal("expected no abort for terminal build")
	}
}
