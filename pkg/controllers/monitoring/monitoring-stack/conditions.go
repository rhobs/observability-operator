package monitoringstack

import (
	"fmt"

	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AvailableReason                = "MonitoringStackAvailable"
	ReconciledReason               = "MonitoringStackReconciled"
	FailedToReconcileReason        = "FailedToReconcile"
	PrometheusNotAvailable         = "PrometheusNotAvailable"
	PrometheusNotReconciled        = "PrometheusNotReconciled"
	PrometheusDegraded             = "PrometheusDegraded"
	CannotReadPrometheusConditions = "Cannot read Prometheus status conditions"
	AvailableMessage               = "Monitoring Stack is available"
	SuccessfullyReconciledMessage  = "Monitoring Stack is successfully reconciled"
	NoReason                       = "None"
)

func updateConditions(msConditions []v1alpha1.Condition, promConditions []monv1.PrometheusCondition, generation int64, recError error) []v1alpha1.Condition {
	if len(msConditions) == 0 {
		return []v1alpha1.Condition{
			{
				Type:               v1alpha1.AvailableCondition,
				Status:             v1alpha1.ConditionUnknown,
				Reason:             NoReason,
				LastTransitionTime: metav1.Now(),
			},
			{
				Type:               v1alpha1.ReconciledCondition,
				Status:             v1alpha1.ConditionUnknown,
				Reason:             NoReason,
				LastTransitionTime: metav1.Now(),
			},
		}
	}
	var updatedConditions []v1alpha1.Condition

	for _, mc := range msConditions {
		switch mc.Type {
		case v1alpha1.AvailableCondition:
			available := updateAvailable(mc, promConditions, generation)
			if !available.Equal(mc) {
				available.LastTransitionTime = metav1.Now()
			}
			updatedConditions = append(updatedConditions, available)
		case v1alpha1.ReconciledCondition:
			reconciled := updateReconciled(mc, promConditions, generation, recError)
			if !reconciled.Equal(mc) {
				reconciled.LastTransitionTime = metav1.Now()
			}
			updatedConditions = append(updatedConditions, reconciled)
		}
	}

	return updatedConditions
}

// updateAvailable gets existing "Available" condition and updates its parameters
// based on the Prometheus "Available" condition
func updateAvailable(ac v1alpha1.Condition, prometheusConditions []monv1.PrometheusCondition, generation int64) v1alpha1.Condition {
	ac.ObservedGeneration = generation

	prometheusAvailable, err := getPrometheusCondition(prometheusConditions, monv1.PrometheusAvailable)

	if err != nil {
		ac.Status = v1alpha1.ConditionUnknown
		ac.Reason = PrometheusNotAvailable
		ac.Message = CannotReadPrometheusConditions
		return ac
	}

	if prometheusAvailable.Status != monv1.PrometheusConditionTrue {
		ac.Status = prometheusStatusToMSStatus(prometheusAvailable.Status)
		if prometheusAvailable.Status == monv1.PrometheusConditionDegraded {
			ac.Reason = PrometheusDegraded
		} else {
			ac.Reason = PrometheusNotAvailable
		}
		ac.Message = prometheusAvailable.Message
		return ac
	}
	ac.Status = v1alpha1.ConditionTrue
	ac.Reason = AvailableReason
	ac.Message = AvailableMessage
	return ac
}

// updateReconciled updates "Reconciled" conditions based on the provided error value and
// Prometheus "Reconciled" condition
func updateReconciled(rc v1alpha1.Condition, prometheusConditions []monv1.PrometheusCondition, generation int64, err error) v1alpha1.Condition {
	rc.ObservedGeneration = generation

	if err != nil {
		rc.Status = v1alpha1.ConditionFalse
		rc.Message = err.Error()
		rc.Reason = FailedToReconcileReason
		return rc
	}
	prometheusReconciled, err := getPrometheusCondition(prometheusConditions, monv1.PrometheusReconciled)

	if err != nil {
		rc.Status = v1alpha1.ConditionUnknown
		rc.Reason = PrometheusNotReconciled
		rc.Message = CannotReadPrometheusConditions
		return rc
	}

	if prometheusReconciled.Status != monv1.PrometheusConditionTrue {
		rc.Status = prometheusStatusToMSStatus(prometheusReconciled.Status)
		rc.Reason = PrometheusNotReconciled
		rc.Message = prometheusReconciled.Message
		return rc
	}
	rc.Status = v1alpha1.ConditionTrue
	rc.Reason = ReconciledReason
	rc.Message = SuccessfullyReconciledMessage
	return rc
}

func getPrometheusCondition(prometheusConditions []monv1.PrometheusCondition, t monv1.PrometheusConditionType) (*monv1.PrometheusCondition, error) {
	for _, c := range prometheusConditions {
		if c.Type == t {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("cannot find condition %v", t)
}

func prometheusStatusToMSStatus(ps monv1.PrometheusConditionStatus) v1alpha1.ConditionStatus {
	switch ps {
	// Prometheus "Available" condition with status "Degraded" is reported as "Available" condition
	// with status false
	case monv1.PrometheusConditionDegraded:
		return v1alpha1.ConditionFalse
	case monv1.PrometheusConditionTrue:
		return v1alpha1.ConditionTrue
	case monv1.PrometheusConditionFalse:
		return v1alpha1.ConditionFalse
	case monv1.PrometheusConditionUnknown:
		return v1alpha1.ConditionUnknown
	default:
		return v1alpha1.ConditionUnknown
	}
}
