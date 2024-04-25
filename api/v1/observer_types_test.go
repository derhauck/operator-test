package v1

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestObserverStatus_GetLatestIterationResult(t *testing.T) {
	entry := ObserverSpecEntry{
		Endpoint: "http://testendpoint",
		Name:     "test",
	}
	observer := Observer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Observer",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: ObserverSpec{
			Entries: []ObserverSpecEntry{
				entry,
			},
			RetryAfterSeconds: 3,
		},
		Status: ObserverStatus{
			CurrentItem: 0,
		},
	}

	t.Run("can get last iteration result", func(t *testing.T) {
		status := observer.Status
		if status.GetLatestIterationResult() == nil {
			t.Errorf("latest iterationresult is nil, needs to be set")
		}

		result := status.GetLatestIterationResult()
		if result.PodStatusList == nil {
			t.Errorf("current podStatus list is nil, needs to be empty")
		}
		result.Add(&PodStatus{
			Name:   "current",
			Status: v1.PodFailed,
		})

		if len(*result.PodStatusList) != 1 {
			t.Errorf("current podStatusList should contain one item %v", result.PodStatusList)
		}

		if (*result.PodStatusList)[0].Name == "" {
			t.Errorf("current podStatusList item should contain 'Name'")
		}

		newResult := status.GetLatestIterationResult()
		if len(*newResult.PodStatusList) != 1 {
			t.Errorf("current podStatusList should still contain one item %v", newResult.PodStatusList)
		}
		if result != newResult {
			t.Errorf("Getting latest iteration result should return the same result when called")
		}
	})
}
