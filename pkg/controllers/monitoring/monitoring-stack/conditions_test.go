package monitoringstack

import (
	"testing"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateConditions(t *testing.T) {
	transitionTime := metav1.Now()

	tt := []struct {
		name             string
		msWithConditions v1alpha1.MonitoringStack
		prometheus       monv1.Prometheus
		recError         error
		expectedResults  []v1alpha1.Condition
		// flag to compare LastTransitionTime of each condition
		sameTransitionTimes bool
	}{
		{
			name: "empty conditions",
			msWithConditions: v1alpha1.MonitoringStack{
				Status: v1alpha1.MonitoringStackStatus{
					Conditions: []v1alpha1.Condition{},
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
			},
			recError:   nil,
			prometheus: monv1.Prometheus{},
			expectedResults: []v1alpha1.Condition{
				{
					Type:   v1alpha1.AvailableCondition,
					Status: v1alpha1.ConditionUnknown,
					Reason: NoReason,
				},
				{
					Type:   v1alpha1.ReconciledCondition,
					Status: v1alpha1.ConditionUnknown,
					Reason: NoReason,
				},
				{
					Type:               v1alpha1.ResourceDiscoveryCondition,
					Status:             v1alpha1.ConditionFalse,
					Reason:             ResourceSelectorIsNil,
					Message:            ResourceSelectorIsNilMessage,
					ObservedGeneration: 1,
					LastTransitionTime: transitionTime,
				},
			},
		},
		{
			name: "lastTransitionTime is updated",
			msWithConditions: v1alpha1.MonitoringStack{
				Status: v1alpha1.MonitoringStackStatus{
					Conditions: []v1alpha1.Condition{
						{
							Type:               v1alpha1.AvailableCondition,
							Status:             v1alpha1.ConditionTrue,
							ObservedGeneration: 1,
							Reason:             AvailableReason,
							Message:            AvailableMessage,
							LastTransitionTime: transitionTime,
						},
						{
							Type:               v1alpha1.ReconciledCondition,
							Status:             v1alpha1.ConditionTrue,
							ObservedGeneration: 1,
							Reason:             ReconciledReason,
							Message:            SuccessfullyReconciledMessage,
							LastTransitionTime: transitionTime,
						},
						{
							Type:               v1alpha1.ResourceDiscoveryCondition,
							Status:             v1alpha1.ConditionFalse,
							Reason:             ResourceSelectorIsNil,
							Message:            ResourceSelectorIsNilMessage,
							ObservedGeneration: 1,
							LastTransitionTime: transitionTime,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
			},
			recError: nil,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.PrometheusCondition{
						{
							Type:               monv1.PrometheusAvailable,
							Status:             monv1.PrometheusConditionTrue,
							ObservedGeneration: 2,
						},
						{
							Type:               monv1.PrometheusReconciled,
							Status:             monv1.PrometheusConditionTrue,
							ObservedGeneration: 2,
						},
					}}},
			expectedResults: []v1alpha1.Condition{
				{
					Type:               v1alpha1.AvailableCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 2,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
					LastTransitionTime: transitionTime,
				},
				{
					Type:               v1alpha1.ReconciledCondition,
					Status:             v1alpha1.ConditionTrue,
					ObservedGeneration: 2,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
					LastTransitionTime: transitionTime,
				},
				{
					Type:               v1alpha1.ResourceDiscoveryCondition,
					Status:             v1alpha1.ConditionFalse,
					Reason:             ResourceSelectorIsNil,
					Message:            ResourceSelectorIsNilMessage,
					ObservedGeneration: 2,
					LastTransitionTime: transitionTime,
				},
			},
			sameTransitionTimes: false,
		},
	}

	for _, test := range tt {
		res := updateConditions(&test.msWithConditions, test.prometheus, test.recError)
		for _, c := range res {
			expectedC := getConditionByType(test.expectedResults, c.Type)
			assert.Check(t, expectedC.Equal(c), "%s - expected:\n %v\n and got:\n %v\n", test.name, expectedC, c)
			if test.sameTransitionTimes {
				assert.Check(t, expectedC.LastTransitionTime.Equal(&c.LastTransitionTime))
			} else {
				assert.Check(t, c.LastTransitionTime.After(transitionTime.Time))
			}
		}
	}
}

func getConditionByType(conditions []v1alpha1.Condition, ctype v1alpha1.ConditionType) *v1alpha1.Condition {
	for _, c := range conditions {
		if c.Type == ctype {
			return &c
		}
	}
	return nil
}

func TestUpdateAvailable(t *testing.T) {
	tt := []struct {
		name              string
		prometheus        monv1.Prometheus
		previousCondition v1alpha1.Condition
		generation        int64
		expectedResult    v1alpha1.Condition
	}{
		{
			name: "conditions not changed when Prometheus Available",
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
			},
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.PrometheusCondition{
						{
							Type:               monv1.PrometheusAvailable,
							Status:             monv1.PrometheusConditionTrue,
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
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
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
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
			},
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.PrometheusCondition{
						{
							Type:               monv1.PrometheusAvailable,
							Status:             monv1.PrometheusConditionDegraded,
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
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 2,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
			},
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.PrometheusCondition{
						{
							Type:               monv1.PrometheusAvailable,
							Status:             monv1.PrometheusConditionFalse,
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
		res := updateAvailable(test.previousCondition, test.prometheus, test.generation)
		assert.Check(t, test.expectedResult.Equal(res), "%s - expected:\n %v\n and got:\n %v\n", test.name, test.expectedResult, res)
	}
}

func TestUpdateReconciled(t *testing.T) {
	tt := []struct {
		name              string
		prometheus        monv1.Prometheus
		previousCondition v1alpha1.Condition
		generation        int64
		recError          error
		expectedResult    v1alpha1.Condition
	}{
		{
			name: "conditions not changed when Prometheus Available",
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
			},
			recError:   nil,
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.PrometheusCondition{
						{
							Type:               monv1.PrometheusReconciled,
							Status:             monv1.PrometheusConditionTrue,
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
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
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
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
			},
			recError:   nil,
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.PrometheusCondition{
						{
							Type:               monv1.PrometheusReconciled,
							Status:             monv1.PrometheusConditionDegraded,
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
			previousCondition: v1alpha1.Condition{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionTrue,
				ObservedGeneration: 2,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
			},
			recError:   nil,
			generation: 1,
			prometheus: monv1.Prometheus{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 3,
				},
				Status: monv1.PrometheusStatus{
					Conditions: []monv1.PrometheusCondition{
						{
							Type:               monv1.PrometheusReconciled,
							Status:             monv1.PrometheusConditionFalse,
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
		res := updateReconciled(test.previousCondition, test.prometheus, test.generation, test.recError)
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
