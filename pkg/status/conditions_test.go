package status

import (
	"testing"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	shared "github.com/rhobs/observability-operator/pkg/apis/shared"
)

func TestUpdateAvailable(t *testing.T) {
	tt := []struct {
		name               string
		operand            Operand
		previousConditions []shared.Condition
		generation         int64
		expectedResult     shared.Condition
	}{
		{
			name: "conditions not changed when Prometheus Available",
			previousConditions: []shared.Condition{
				{
					Type:               shared.AvailableCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object: &monv1.Prometheus{
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
			},
			generation: 1,
			expectedResult: shared.Condition{
				Type:               shared.AvailableCondition,
				Status:             shared.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
			},
		},
		{
			name: "cannot read Prometheus conditions",
			previousConditions: []shared.Condition{
				{
					Type:               shared.AvailableCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			generation: 1,
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object:              &monv1.Prometheus{},
			},
			expectedResult: shared.Condition{
				Type:               shared.AvailableCondition,
				Status:             shared.ConditionUnknown,
				ObservedGeneration: 1,
				Reason:             "PrometheusNotAvailable",
				Message:            "Cannot read Prometheus status conditions",
			},
		},
		{
			name: "degraded Prometheus conditions",
			previousConditions: []shared.Condition{
				{
					Type:               shared.AvailableCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			generation: 1,
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object: &monv1.Prometheus{
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
			},
			expectedResult: shared.Condition{
				Type:               shared.AvailableCondition,
				Status:             shared.ConditionFalse,
				ObservedGeneration: 1,
				Reason:             "PrometheusDegraded",
			},
		},
		{
			name: "Prometheus observed generation is different from the Prometheus generation",
			previousConditions: []shared.Condition{
				{
					Type:               shared.AvailableCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 2,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
				},
			},
			generation: 1,
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object: &monv1.Prometheus{
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
			},
			expectedResult: shared.Condition{
				Type:               shared.AvailableCondition,
				Status:             shared.ConditionTrue,
				ObservedGeneration: 2,
				Reason:             AvailableReason,
				Message:            AvailableMessage,
			},
		},
	}

	for _, test := range tt {
		res := updateAvailable(test.previousConditions, test.operand, test.generation)
		assert.Check(t, test.expectedResult.Equal(res), "%s - expected:\n %v\n and got:\n %v\n", test.name, test.expectedResult, res)
	}
}

func TestUpdateReconciled(t *testing.T) {
	tt := []struct {
		name               string
		operand            Operand
		previousConditions []shared.Condition
		generation         int64
		recError           error
		expectedResult     shared.Condition
	}{
		{
			name: "conditions not changed when Prometheus Available",
			previousConditions: []shared.Condition{
				{
					Type:               shared.ReconciledCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object: &monv1.Prometheus{
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
			},
			expectedResult: shared.Condition{
				Type:               shared.ReconciledCondition,
				Status:             shared.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
			},
		},
		{
			name: "cannot read Prometheus status conditions",
			previousConditions: []shared.Condition{
				{
					Type:               shared.ReconciledCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object:              &monv1.Prometheus{},
			},
			expectedResult: shared.Condition{
				Type:               shared.ReconciledCondition,
				Status:             shared.ConditionUnknown,
				ObservedGeneration: 1,
				Reason:             "PrometheusNotReconciled",
				Message:            "Cannot read Prometheus status conditions",
			},
		},
		{
			name: "degraded Prometheus conditions",
			previousConditions: []shared.Condition{
				{
					Type:               shared.ReconciledCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 1,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object: &monv1.Prometheus{
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
			},
			expectedResult: shared.Condition{
				Type:               shared.ReconciledCondition,
				Status:             shared.ConditionFalse,
				ObservedGeneration: 1,
				Reason:             "PrometheusNotReconciled",
			},
		},
		{
			name: "Prometheus observed generation is different from the Prometheus generation",
			previousConditions: []shared.Condition{
				{
					Type:               shared.ReconciledCondition,
					Status:             shared.ConditionTrue,
					ObservedGeneration: 2,
					Reason:             ReconciledReason,
					Message:            SuccessfullyReconciledMessage,
				},
			},
			recError:   nil,
			generation: 1,
			operand: Operand{
				name:                "Prometheus",
				affectsAvailability: true,
				affectsReconciled:   true,
				Object: &monv1.Prometheus{
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
			},
			expectedResult: shared.Condition{
				Type:               shared.ReconciledCondition,
				Status:             shared.ConditionTrue,
				ObservedGeneration: 2,
				Reason:             ReconciledReason,
				Message:            SuccessfullyReconciledMessage,
			},
		},
	}

	for _, test := range tt {
		res := updateReconciled(test.previousConditions, test.operand, test.generation, test.recError)
		assert.Check(t, test.expectedResult.Equal(res), "%s - expected:\n %v\n and got:\n %v\n", test.name, test.expectedResult, res)
	}
}

func TestUpdateResourceDiscovery(t *testing.T) {
	transitionTime := metav1.Now()
	tt := []struct {
		name             string
		msWithConditions *v1alpha1.MonitoringStack
		expectedResults  shared.Condition
	}{
		{
			name: "set resource discovery true when ResourceSelector not nil",
			msWithConditions: &v1alpha1.MonitoringStack{
				Spec: v1alpha1.MonitoringStackSpec{
					ResourceSelector: &metav1.LabelSelector{},
				},
			},
			expectedResults: shared.Condition{
				Type:    shared.ResourceDiscoveryCondition,
				Status:  shared.ConditionTrue,
				Reason:  NoReason,
				Message: ResourceDiscoveryOnMessage,
			},
		},
		{
			name: "set resource discovery false when ResourceSelector is nil",
			msWithConditions: &v1alpha1.MonitoringStack{
				Spec: v1alpha1.MonitoringStackSpec{
					ResourceSelector: nil,
				},
			},
			expectedResults: shared.Condition{
				Type:               shared.ResourceDiscoveryCondition,
				Status:             shared.ConditionFalse,
				Reason:             ResourceSelectorIsNil,
				Message:            ResourceSelectorIsNilMessage,
				LastTransitionTime: transitionTime,
			},
		},
	}

	for _, test := range tt {
		res, err := updateResourceDiscovery(test.msWithConditions)
		assert.NilError(t, err)
		assert.Check(t, test.expectedResults.Equal(*res), "%s - expected:\n %v\n and got:\n %v\n", test.name, test.expectedResults, res)
	}
}

/* func TestGetConditionsFromObject(t *testing.T) {
	tests := []struct {
		name               string
		testObject         client.Object
		expectedConditions []shared.Condition
	}{
		{
			name:               "empty monitoring stack",
			testObject:         &v1alpha1.MonitoringStack{},
			expectedConditions: nil,
		},
		{
			name: "monitoring stack with some valid conditions",
			testObject: &v1alpha1.MonitoringStack{
				Status: v1alpha1.MonitoringStackStatus{
					Conditions: []shared.Condition{
						{
							Type:               available,
							Status:             shared.ConditionTrue,
							Reason:             AvailableReason,
							Message:            AvailableMessage,
							ObservedGeneration: 1,
						},
					},
				},
			},
			expectedConditions: []shared.Condition{
				{
					Type:               available,
					Status:             shared.ConditionTrue,
					Reason:             AvailableReason,
					Message:            AvailableMessage,
					ObservedGeneration: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, err := getConditionsFromObject(tt.testObject)
			assert.NilError(t, err)
			assert.DeepEqual(t, conditions, tt.expectedConditions)
		})
	}
}
*/
