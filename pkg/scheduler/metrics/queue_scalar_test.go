package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	v1 "k8s.io/api/core/v1"
)

func TestUpdateScalarResourceMetrics_ZeroAndCleanup(t *testing.T) {
	// Reset global state for this test to ensure isolation.
	queueAllocatedScalarResource.Reset()
	knownScalarResourcesLock.Lock()
	knownScalarResources = make(map[string]map[string]struct{})
	knownScalarResourcesLock.Unlock()

	queueName := "testqueue"
	resourceA := v1.ResourceName("nvidia.com/gpu")
	resourceB := v1.ResourceName("amd.com/gpu")

	// 1. Set resourceA to 5, resourceB to 10
	UpdateQueueAllocated(queueName, 0, 0, map[v1.ResourceName]float64{resourceA: 5, resourceB: 10})
	if got := testutil.ToFloat64(queueAllocatedScalarResource.WithLabelValues(queueName, string(resourceA))); got != 5 {
		t.Errorf("expected %s to be 5, got %v", resourceA, got)
	}
	if got := testutil.ToFloat64(queueAllocatedScalarResource.WithLabelValues(queueName, string(resourceB))); got != 10 {
		t.Errorf("expected %s to be 10, got %v", resourceB, got)
	}

	// 2. Update with only resourceA, resourceB should be set to zero
	UpdateQueueAllocated(queueName, 0, 0, map[v1.ResourceName]float64{resourceA: 3})
	if got := testutil.ToFloat64(queueAllocatedScalarResource.WithLabelValues(queueName, string(resourceA))); got != 3 {
		t.Errorf("expected %s to be 3, got %v", resourceA, got)
	}
	if got := testutil.ToFloat64(queueAllocatedScalarResource.WithLabelValues(queueName, string(resourceB))); got != 0 {
		t.Errorf("expected %s to be 0 after missing in update, got %v", resourceB, got)
	}

	// 3. Update with nil scalarResources, all known resources should be set to zero
	UpdateQueueAllocated(queueName, 0, 0, nil)
	if got := testutil.ToFloat64(queueAllocatedScalarResource.WithLabelValues(queueName, string(resourceA))); got != 0 {
		t.Errorf("expected %s to be 0 after nil update, got %v", resourceA, got)
	}
	if got := testutil.ToFloat64(queueAllocatedScalarResource.WithLabelValues(queueName, string(resourceB))); got != 0 {
		t.Errorf("expected %s to be 0 after nil update, got %v", resourceB, got)
	}

	// 4. Delete metrics and ensure they're gone
	DeleteQueueMetrics(queueName)
	if count := testutil.CollectAndCount(queueAllocatedScalarResource); count != 0 {
		t.Errorf("expected no metrics for queueAllocatedScalarResource after delete, got %d", count)
	}
}

func TestUpdateQueueInqueue(t *testing.T) {
	queueInqueueMilliCPU.Reset()
	queueInqueueMemory.Reset()
	queueInqueueScalarResource.Reset()
	knownScalarResourcesLock.Lock()
	knownScalarResources = make(map[string]map[string]struct{})
	knownScalarResourcesLock.Unlock()

	queueName := "inqueue-test-queue"
	gpu := v1.ResourceName("nvidia.com/gpu")

	// basic write
	UpdateQueueInqueue(queueName, 8000, 68719476736, map[v1.ResourceName]float64{gpu: 8})
	if got := testutil.ToFloat64(queueInqueueMilliCPU.WithLabelValues(queueName)); got != 8000 {
		t.Errorf("inqueue milli_cpu: expected 8000, got %v", got)
	}
	if got := testutil.ToFloat64(queueInqueueMemory.WithLabelValues(queueName)); got != 68719476736 {
		t.Errorf("inqueue memory: expected 68719476736, got %v", got)
	}
	if got := testutil.ToFloat64(queueInqueueScalarResource.WithLabelValues(queueName, string(gpu))); got != 8 {
		t.Errorf("inqueue gpu: expected 8, got %v", got)
	}

	// scalar resource removed in next update must zero out
	UpdateQueueInqueue(queueName, 4000, 34359738368, map[v1.ResourceName]float64{})
	if got := testutil.ToFloat64(queueInqueueScalarResource.WithLabelValues(queueName, string(gpu))); got != 0 {
		t.Errorf("inqueue gpu after removal: expected 0, got %v", got)
	}

	// DeleteQueueMetrics cleans up inqueue series
	DeleteQueueMetrics(queueName)
	if count := testutil.CollectAndCount(queueInqueueMilliCPU); count != 0 {
		t.Errorf("expected queueInqueueMilliCPU to be empty after delete, got %d", count)
	}
	if count := testutil.CollectAndCount(queueInqueueScalarResource); count != 0 {
		t.Errorf("expected queueInqueueScalarResource to be empty after delete, got %d", count)
	}
}
