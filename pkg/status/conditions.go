package status

import (
	"fmt"

	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AvailableReason               = "MonitoringStackAvailable"
	ReconciledReason              = "MonitoringStackReconciled"
	FailedToReconcileReason       = "FailedToReconcile"
	ResourceSelectorIsNil         = "ResourceSelectorNil"
	AvailableMessage              = "Monitoring Stack is available"
	SuccessfullyReconciledMessage = "Monitoring Stack is successfully reconciled"
	ResourceSelectorIsNilMessage  = "No resources will be discovered, ResourceSelector is nil"
	ResourceDiscoveryOnMessage    = "Resource discovery is operational"
	NoReason                      = "None"
	available                     = "Available"
	reconciled                    = "Reconciled"
)

func UpdateConditions(stackObj client.Object, operands []Operand, recError error) ([]v1alpha1.Condition, error) {
	var availableCon v1alpha1.Condition
	var reconciledCon v1alpha1.Condition
	conditions, err := getConditionsFromObject(stackObj)
	if err != nil {
		return nil, err
	}
	for _, opr := range operands {
		if opr.affectsAvailability {
			availableCon = updateAvailable(conditions, opr, stackObj.GetGeneration())
		}
		if opr.affectsReconciled {
			reconciledCon = updateReconciled(conditions, opr, stackObj.GetGeneration(), recError)
		}
	}

	resourceDiscoveryCon, err := updateResourceDiscovery(stackObj)
	if err != nil {
		return nil, err
	}

	return []v1alpha1.Condition{
		availableCon,
		reconciledCon,
		*resourceDiscoveryCon,
	}, nil
}

// updateAvailable gets existing "Available" condition and updates its parameters
// based on the operand "Available" condition
func updateAvailable(conditions []v1alpha1.Condition, opr Operand, generation int64) v1alpha1.Condition {
	ac, err := getConditionByType(conditions, v1alpha1.AvailableCondition)
	if err != nil {
		ac = v1alpha1.Condition{
			Type:               v1alpha1.AvailableCondition,
			Status:             v1alpha1.ConditionUnknown,
			Reason:             NoReason,
			LastTransitionTime: metav1.Now(),
		}
	}

	operandAvailable, err := opr.getConditionByType(available)

	if err != nil {
		ac.Status = v1alpha1.ConditionUnknown
		ac.Reason = fmt.Sprintf("%sNotAvailable", opr.name)
		ac.Message = fmt.Sprintf("Cannot read %s status conditions", opr.name)
		ac.LastTransitionTime = metav1.Now()
		return ac
	}
	// MonitoringStack status will not be updated if there is a difference between the operand generation
	// and the operand ObservedGeneration. This can occur, for example, in the case of an invalid operand configuration.
	if operandAvailable.ObservedGeneration != opr.Object.GetGeneration() {
		return ac
	}

	if operandAvailable.Status != "True" {
		ac.Status = prometheusStatusToMSStatus(operandAvailable.Status)
		if operandAvailable.Status == "Degraded" {
			ac.Reason = fmt.Sprintf("%sDegraded", opr.name)
		} else {
			ac.Reason = fmt.Sprintf("%sNotAvailable", opr.name)
		}
		ac.Message = operandAvailable.Message
		ac.LastTransitionTime = metav1.Now()
		return ac
	}
	ac.Status = v1alpha1.ConditionTrue
	ac.Reason = AvailableReason
	ac.Message = AvailableMessage
	ac.ObservedGeneration = generation
	ac.LastTransitionTime = metav1.Now()
	return ac
}

// updateReconciled updates "Reconciled" conditions based on the provided error value and
// the operand "Reconciled" condition
func updateReconciled(conditions []v1alpha1.Condition, opr Operand, generation int64, reconcileErr error) v1alpha1.Condition {
	rc, cErr := getConditionByType(conditions, v1alpha1.ReconciledCondition)
	if cErr != nil {
		rc = v1alpha1.Condition{
			Type:               v1alpha1.ReconciledCondition,
			Status:             v1alpha1.ConditionUnknown,
			Reason:             NoReason,
			LastTransitionTime: metav1.Now(),
		}
	}
	if reconcileErr != nil {
		rc.Status = v1alpha1.ConditionFalse
		rc.Message = reconcileErr.Error()
		rc.Reason = FailedToReconcileReason
		rc.LastTransitionTime = metav1.Now()
		return rc
	}
	operandReconciled, reconcileErr := opr.getConditionByType(reconciled)

	if reconcileErr != nil {
		rc.Status = v1alpha1.ConditionUnknown
		rc.Reason = fmt.Sprintf("%sNotReconciled", opr.name)
		rc.Message = fmt.Sprintf("Cannot read %s status conditions", opr.name)
		rc.LastTransitionTime = metav1.Now()
		return rc
	}

	if operandReconciled.ObservedGeneration != opr.Object.GetGeneration() {
		return rc
	}

	if operandReconciled.Status != "True" {
		rc.Status = prometheusStatusToMSStatus(operandReconciled.Status)
		rc.Reason = fmt.Sprintf("%sNotReconciled", opr.name)
		rc.Message = operandReconciled.Message
		rc.LastTransitionTime = metav1.Now()
		return rc
	}
	rc.Status = v1alpha1.ConditionTrue
	rc.Reason = ReconciledReason
	rc.Message = SuccessfullyReconciledMessage
	rc.ObservedGeneration = generation
	rc.LastTransitionTime = metav1.Now()
	return rc
}

func getConditionByType(conditions []v1alpha1.Condition, t v1alpha1.ConditionType) (v1alpha1.Condition, error) {
	for _, c := range conditions {
		if c.Type == t {
			return c, nil
		}
	}
	return v1alpha1.Condition{}, fmt.Errorf("ERROR: condition type %v not found", t)
}

func getConditionsFromObject(o client.Object) ([]v1alpha1.Condition, error) {
	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	if err != nil {
		return nil, err
	}
	var conditions []v1alpha1.Condition
	untypedCon, ok, err := unstructured.NestedSlice(unstrObj, "status", "conditions")
	// if no conditions found, return empty conditions
	if !ok {
		return conditions, nil
	}
	if err != nil {
		return nil, err
	}

	for _, untypedC := range untypedCon {
		cMap, ok := untypedC.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("converting to map[string]interface{}: %v", untypedC)
		}
		conditions = append(conditions, v1alpha1.Condition{
			Type:               v1alpha1.ConditionType(convert[string](cMap["type"])),
			Reason:             convert[string](cMap["reason"]),
			Status:             v1alpha1.ConditionStatus(convert[string](cMap["status"])),
			Message:            convert[string](cMap["message"]),
			ObservedGeneration: convert[int64](cMap["observedGeneration"]),
			LastTransitionTime: convert[metav1.Time](cMap["lastTransitionTime"]),
		})
	}
	return conditions, nil

}

func prometheusStatusToMSStatus(ps string) v1alpha1.ConditionStatus {
	switch ps {
	// Prometheus "Available" condition with status "Degraded" is reported as "Available" condition
	// with status false
	case "Degraded":
		return v1alpha1.ConditionFalse
	case "True":
		return v1alpha1.ConditionTrue
	case "False":
		return v1alpha1.ConditionFalse
	case "Unknown":
		return v1alpha1.ConditionUnknown
	default:
		return v1alpha1.ConditionUnknown
	}
}
