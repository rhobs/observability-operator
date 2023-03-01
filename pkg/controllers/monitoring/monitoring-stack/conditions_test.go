package monitoringstack

import (
	"testing"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateAvailable(t *testing.T) {
	tt := []struct {
		name               string
		prometheus         monv1.Prometheus
		previousConditions []v1alpha1.Condition
		generation         int64
		expectedResult     v1alpha1.Condition
	}{
		{
			name: "conditions not changed when Prometheus Available",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.AvailableCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.Condition{
						{
							Type:               monv1.Available,
							Status:             monv1.ConditionTrue,
							ObservedGeneration: 1,
						},
					}}},
			generation: 1,
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
			},
		},
		{
			name: "cannot read Prometheus conditions",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.AvailableCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			generation: 1,
			prometheus: monv1.Prometheus{},
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionUnknown,
				ObservedGeneration: 1,
				Reason:             PrometheusNotAvailable,
				Message:            CannotReadPrometheusConditions,
			},
		},
		{
			name: "degraded Prometheus conditions",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.AvailableCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.Condition{
						{
							Type:               monv1.Available,
							Status:             monv1.ConditionDegraded,
							ObservedGeneration: 1,
						},
					}}},
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionFalse,
				ObservedGeneration: 1,
				Reason:             PrometheusDegraded,
			},
		},
		{
			name: "Prometheus observed generation is different from the Prometheus generation",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.AvailableCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 2,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.Condition{
						{
							Type:               monv1.Available,
							Status:             monv1.ConditionFalse,
							ObservedGeneration: 2,
						},
					}}},
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 2,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
			},
		},
	}

	for _, test := range tt {
		res := updateAvailable(test.previousConditions, test.prometheus, test.generation)
		assert.Check(t, test.expectedResult.Equal(res), "%s - expected:\n %v\n and got:\n %v\n", test.name, test.expectedResult, res)
	}
}

func TestUpdateReconciled(t *testing.T) {
	tt := []struct {
		name               string
		prometheus         monv1.Prometheus
		previousConditions []v1alpha1.Condition
		generation         int64
		recError           error
		expectedResult     v1alpha1.Condition
	}{
		{
			name: "conditions not changed when Prometheus Available",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.ReconciledCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.Condition{
						{
							Type:               monv1.Reconciled,
							Status:             monv1.ConditionTrue,
							ObservedGeneration: 1,
						},
					}}},
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
			},
		},
		{
			name: "cannot read Prometheus conditions",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.ReconciledCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			prometheus: monv1.Prometheus{},
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionUnknown,
				ObservedGeneration: 1,
				Reason:             PrometheusNotReconciled,
				Message:            CannotReadPrometheusConditions,
			},
		},
		{
			name: "degraded Prometheus conditions",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.ReconciledCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.Condition{
						{
							Type:               monv1.Reconciled,
							Status:             monv1.ConditionDegraded,
							ObservedGeneration: 1,
						},
					}}},
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionFalse,
				ObservedGeneration: 1,
				Reason:             PrometheusNotReconciled,
			},
		},
		{
			name: "Prometheus observed generation is different from the Prometheus generation",
			previousConditions: []v1alpha1.Condition{
				{
					Type:               v1alpha1.ReconciledCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 2,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.Condition{
						{
							Type:               monv1.Reconciled,
							Status:             monv1.ConditionFalse,
							ObservedGeneration: 2,
						},
					}}},
			expectedResult: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 2,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
			},
		},
	}

	for _, test := range tt {
		res := updateReconciled(test.previousConditions, test.prometheus, test.generation, test.recError)
		assert.Check(t, test.expectedResult.Equal(res), "%s - expected:\n %v\n and got:\n %v\n", test.name, test.expectedResult, res)
	}
}

func TestUpdateResourceDiscovery(t *testing.T) {
	transitionTime := metav1.Now()
	tt := []struct {
		name             string
		msWithConditions v1alpha1.MonitoringStack
		expectedResults  v1alpha1.Condition
	}{
		{
			name: "set resource discovery true when ResourceSelector not nil",
			msWithConditions: v1alpha1.MonitoringStack{
				Spec: v1alpha1.MonitoringStackSpec{
					ResourceSelector: &metav1.LabelSelector{},
				},
			},
			expectedResults: v1alpha1.Condition{
				Type:    v1alpha1.ResourceDiscoveryCondition,
				Status:  v1alpha1.ConditionTrue,
				Reason:  NoReason,
				Message: ResourceDiscoveryOnMessage,
			},
		},
		{
			name: "set resource discovery false when ResourceSelector is nil",
			msWithConditions: v1alpha1.MonitoringStack{
				Spec: v1alpha1.MonitoringStackSpec{
					ResourceSelector: nil,
				},
			},
			expectedResults: v1alpha1.Condition{
				Type:               v1alpha1.ResourceDiscoveryCondition,
				Status:             v1alpha1.ConditionFalse,
				Reason:             ResourceSelectorIsNil,
				Message:            ResourceSelectorIsNilMessage,
				LastTransitionTime: transitionTime,
			},
		},
	}

	for _, test := range tt {
		res := updateResourceDiscovery(&test.msWithConditions)
		assert.Check(t, test.expectedResults.Equal(res), "%s - expected:\n %v\n and got:\n %v\n", test.name, test.expectedResults, res)
	}

}
