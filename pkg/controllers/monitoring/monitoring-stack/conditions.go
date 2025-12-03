package monitoringstack

import (
	"fmt"

	monv1 "github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

const (
	AvailableReason                = "MonitoringStackAvailable"
	ReconciledReason               = "MonitoringStackReconciled"
	FailedToReconcileReason        = "FailedToReconcile"
	PrometheusNotAvailable         = "PrometheusNotAvailable"
	PrometheusNotReconciled        = "PrometheusNotReconciled"
	PrometheusDegraded             = "PrometheusDegraded"
	ResourceSelectorIsNil          = "ResourceSelectorNil"
	CannotReadPrometheusConditions = "Cannot read Prometheus status conditions"
	AvailableMessage               = "Monitoring Stack is available"
	SuccessfullyReconciledMessage  = "Monitoring Stack is successfully reconciled"
	ResourceSelectorIsNilMessage   = "No resources will be discovered, ResourceSelector is nil"
	ResourceDiscoveryOnMessage     = "Resource discovery is operational"
	NoReason                       = "None"
)

func updateConditions(ms *v1alpha1.MonitoringStack, prom monv1.Prometheus, recError error) []v1alpha1.Condition {
	return []v1alpha1.Condition{
		updateResourceDiscovery(ms),
		updateAvailable(ms.Status.Conditions, prom, ms.Generation),
		updateReconciled(ms.Status.Conditions, prom, ms.Generation, recError),
	}
}

// getMSCondition returns the matching condition from the conditions slice.
// If no item matches, it returns an condition initialized to unknown status.
func getMSCondition(conditions []v1alpha1.Condition, t v1alpha1.ConditionType) v1alpha1.Condition {
	for _, c := range conditions {
		if c.Type == t {
			return c
		}
	}
	return v1alpha1.Condition{
		Type:               t,
		Status:             v1alpha1.ConditionUnknown,
		Reason:             NoReason,
		LastTransitionTime: metav1.Now(),
	}
}

// updateResourceDiscovery updates the ResourceDiscoveryCondition based on the
// ResourceSelector in the MonitorinStack spec. A ResourceSelector of nil causes
// the condition to be false, any other value sets the condition to true
func updateResourceDiscovery(ms *v1alpha1.MonitoringStack) v1alpha1.Condition {
	if ms.Spec.ResourceSelector == nil {
		return v1alpha1.Condition{
			Type:               v1alpha1.ResourceDiscoveryCondition,
			Status:             v1alpha1.ConditionFalse,
			Reason:             ResourceSelectorIsNil,
			Message:            ResourceSelectorIsNilMessage,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: ms.Generation,
		}
	} else {
		return v1alpha1.Condition{
			Type:               v1alpha1.ResourceDiscoveryCondition,
			Status:             v1alpha1.ConditionTrue,
			Reason:             NoReason,
			Message:            ResourceDiscoveryOnMessage,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: ms.Generation,
		}
	}

}

// updateAvailable gets existing "Available" condition and updates its parameters
// based on the Prometheus "Available" condition
func updateAvailable(conditions []v1alpha1.Condition, prom monv1.Prometheus, generation int64) v1alpha1.Condition {
	ac := getMSCondition(conditions, v1alpha1.AvailableCondition)

	prometheusAvailable, err := getPrometheusCondition(prom.Status.Conditions, monv1.Available)
	if err != nil {
		ac.Status = v1alpha1.ConditionUnknown
		ac.Reason = PrometheusNotAvailable
		ac.Message = CannotReadPrometheusConditions
		ac.ObservedGeneration = generation
		ac.LastTransitionTime = metav1.Now()
		return ac
	}

	// MonitoringStack status should not be updated if there is a difference
	// between the Prometheus generation and the Prometheus ObservedGeneration.
	// This can occur, for example, in the case of the Prometheus operator
	// being down and not reconciling the resource.
	if prometheusAvailable.ObservedGeneration != prom.Generation {
		return ac
	}

	ac.ObservedGeneration = generation
	ac.LastTransitionTime = metav1.Now()
	if prometheusAvailable.Status == monv1.ConditionTrue {
		ac.Status = v1alpha1.ConditionTrue
		ac.Reason = AvailableReason
		ac.Message = AvailableMessage
		return ac
	}

	ac.Status = prometheusStatusToMSStatus(prometheusAvailable.Status)
	ac.Reason = PrometheusNotAvailable
	if prometheusAvailable.Status == monv1.ConditionDegraded {
		ac.Reason = PrometheusDegraded
	}
	ac.Message = prometheusAvailable.Message

	return ac

}

// updateReconciled updates "Reconciled" conditions based on the provided error value and
// Prometheus "Reconciled" condition
func updateReconciled(conditions []v1alpha1.Condition, prom monv1.Prometheus, generation int64, reconcileErr error) v1alpha1.Condition {
	rc := getMSCondition(conditions, v1alpha1.ReconciledCondition)

	if reconcileErr != nil {
		rc.Status = v1alpha1.ConditionFalse
		rc.Message = reconcileErr.Error()
		rc.Reason = FailedToReconcileReason
		rc.LastTransitionTime = metav1.Now()
		rc.ObservedGeneration = generation
		return rc
	}

	prometheusReconciled, err := getPrometheusCondition(prom.Status.Conditions, monv1.Reconciled)
	if err != nil {
		rc.Status = v1alpha1.ConditionUnknown
		rc.Reason = PrometheusNotReconciled
		rc.Message = CannotReadPrometheusConditions
		rc.LastTransitionTime = metav1.Now()
		rc.ObservedGeneration = generation
		return rc
	}

	if prometheusReconciled.ObservedGeneration != prom.Generation {
		return rc
	}

	rc.ObservedGeneration = generation
	rc.LastTransitionTime = metav1.Now()
	if prometheusReconciled.Status != monv1.ConditionTrue {
		rc.Status = prometheusStatusToMSStatus(prometheusReconciled.Status)
		rc.Reason = PrometheusNotReconciled
		rc.Message = prometheusReconciled.Message
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

func getPrometheusCondition(prometheusConditions []monv1.Condition, t monv1.ConditionType) (*monv1.Condition, error) {
	for _, c := range prometheusConditions {
		if c.Type == t {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("cannot find condition %v", t)
}

func prometheusStatusToMSStatus(ps monv1.ConditionStatus) v1alpha1.ConditionStatus {
	switch ps {
	// Prometheus "Available" condition with status "Degraded" is reported as
	// "Available" condition with status false.
	case monv1.ConditionDegraded:
		return v1alpha1.ConditionFalse
	case monv1.ConditionTrue:
		return v1alpha1.ConditionTrue
	case monv1.ConditionFalse:
		return v1alpha1.ConditionFalse
	case monv1.ConditionUnknown:
		return v1alpha1.ConditionUnknown
	default:
		return v1alpha1.ConditionUnknown
	}
}
