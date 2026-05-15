package worker

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
	workers []atc.Worker
	landed  []string
	pruned  []string
}

func (m *mockClient) ListWorkers() ([]atc.Worker, error) {
	return m.workers, nil
}

func (m *mockClient) LandWorker(name string) error {
	m.landed = append(m.landed, name)
	return nil
}

func (m *mockClient) PruneWorker(name string) error {
	m.pruned = append(m.pruned, name)
	return nil
}

func newWorkerMR(name, desiredState string) *civ1alpha1.Worker {
	return &civ1alpha1.Worker{
		ObjectMeta: metav1.ObjectMeta{Name: "test-worker"},
		Spec: civ1alpha1.WorkerSpec{
			ForProvider: civ1alpha1.WorkerParameters{
				WorkerName:   name,
				DesiredState: desiredState,
			},
		},
	}
}

func TestObserve_WorkerFound(t *testing.T) {
	mc := &mockClient{
		workers: []atc.Worker{
			{Name: "w1", State: "running", Platform: "linux", Version: "2.1", ActiveContainers: 5, ActiveVolumes: 10},
		},
	}
	e := &external{concourse: mc}
	mg := newWorkerMR("w1", "running")

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=true when desired==actual")
	}
	if mg.Status.AtProvider.State != "running" {
		t.Fatalf("expected state 'running', got %q", mg.Status.AtProvider.State)
	}
	if mg.Status.AtProvider.Platform != "linux" {
		t.Fatalf("expected platform 'linux', got %q", mg.Status.AtProvider.Platform)
	}
}

func TestObserve_WorkerNotFound(t *testing.T) {
	mc := &mockClient{workers: []atc.Worker{}}
	e := &external{concourse: mc}
	mg := newWorkerMR("w1", "running")

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs.ResourceExists {
		t.Fatal("expected ResourceExists=false")
	}
}

func TestObserve_StateDrift(t *testing.T) {
	mc := &mockClient{
		workers: []atc.Worker{
			{Name: "w1", State: "running"},
		},
	}
	e := &external{concourse: mc}
	mg := newWorkerMR("w1", "landing")

	obs, err := e.Observe(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Fatal("expected ResourceUpToDate=false when desired!=actual")
	}
}

func TestUpdate_Land(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newWorkerMR("w1", "landing")

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.landed) != 1 || mc.landed[0] != "w1" {
		t.Fatalf("expected LandWorker(w1), got landed=%v", mc.landed)
	}
}

func TestUpdate_Retire(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newWorkerMR("w1", "retiring")

	_, err := e.Update(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.pruned) != 1 || mc.pruned[0] != "w1" {
		t.Fatalf("expected PruneWorker(w1), got pruned=%v", mc.pruned)
	}
}

func TestDelete_Prunes(t *testing.T) {
	mc := &mockClient{}
	e := &external{concourse: mc}
	mg := newWorkerMR("w1", "")

	_, err := e.Delete(context.Background(), mg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mc.pruned) != 1 || mc.pruned[0] != "w1" {
		t.Fatalf("expected PruneWorker(w1), got pruned=%v", mc.pruned)
	}
}
