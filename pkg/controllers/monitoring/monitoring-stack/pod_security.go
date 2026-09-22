package monitoringstack

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

const requiredSCCAnnotation = "openshift.io/required-scc"

func podSecurityProfile(ms *stack.MonitoringStack) stack.PodSecurityProfile {
	if ms.Spec.PodSecurity.Profile == "" {
		return stack.LegacyStaticPodSecurityProfile
	}

	return ms.Spec.PodSecurity.Profile
}

func sccForProfile(profile stack.PodSecurityProfile) string {
	switch profile {
	case stack.RestrictedV2PodSecurityProfile:
		return "restricted-v2"
	case stack.RestrictedV3PodSecurityProfile:
		return "restricted-v3"
	default:
		return "nonroot-v2"
	}
}

func podAnnotations(profile stack.PodSecurityProfile) map[string]string {
	return map[string]string{
		requiredSCCAnnotation: sccForProfile(profile),
	}
}

func podSecurityContext(profile stack.PodSecurityProfile, staticID int64) *corev1.PodSecurityContext {
	securityContext := &corev1.PodSecurityContext{
		RunAsNonRoot: ptr.To(true),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}

	if profile == stack.LegacyStaticPodSecurityProfile {
		securityContext.FSGroup = ptr.To(staticID)
		securityContext.RunAsUser = ptr.To(staticID)
		return securityContext
	}

	securityContext.FSGroupChangePolicy = ptr.To(corev1.FSGroupChangeOnRootMismatch)
	return securityContext
}

func hostUsers(profile stack.PodSecurityProfile) *bool {
	if profile == stack.RestrictedV3PodSecurityProfile {
		return ptr.To(false)
	}

	return nil
}
