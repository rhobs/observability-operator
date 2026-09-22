package monitoringstack

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	securityv1 "github.com/openshift/api/security/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	stack "github.com/rhobs/observability-operator/pkg/apis/monitoring/v1alpha1"
)

const (
	unsupportedPlatformReason            = "UnsupportedPlatform"
	sccUnavailableReason                 = "SecurityContextConstraintUnavailable"
	invalidNamespaceIDRangeReason        = "InvalidNamespaceIDRange"
	maxUserNamespaceID            uint64 = 65535
)

type podSecurityValidationError struct {
	reason  string
	message string
}

func (e *podSecurityValidationError) Error() string {
	return e.message
}

func validatePodSecurityProfile(
	ctx context.Context,
	reader client.Reader,
	ms *stack.MonitoringStack,
) error {
	profile := podSecurityProfile(ms)
	if profile == stack.LegacyStaticPodSecurityProfile {
		return nil
	}

	sccName := sccForProfile(profile)
	if err := reader.Get(ctx, client.ObjectKey{Name: sccName}, &securityv1.SecurityContextConstraints{}); err != nil {
		if meta.IsNoMatchError(err) {
			return &podSecurityValidationError{
				reason:  unsupportedPlatformReason,
				message: fmt.Sprintf("pod security profile %q requires OpenShift", profile),
			}
		}
		if apierrors.IsNotFound(err) {
			return &podSecurityValidationError{
				reason:  sccUnavailableReason,
				message: fmt.Sprintf("required security context constraint %q is not available", sccName),
			}
		}
		return fmt.Errorf("get security context constraint %q: %w", sccName, err)
	}

	if profile != stack.RestrictedV3PodSecurityProfile {
		return nil
	}

	namespace := &corev1.Namespace{}
	if err := reader.Get(ctx, client.ObjectKey{Name: ms.Namespace}, namespace); err != nil {
		return fmt.Errorf("get namespace %q: %w", ms.Namespace, err)
	}

	uidRange := namespace.Annotations["openshift.io/sa.scc.uid-range"]
	if err := validateUserNamespaceIDRange(uidRange); err != nil {
		return &podSecurityValidationError{
			reason: invalidNamespaceIDRangeReason,
			message: fmt.Sprintf(
				"namespace annotation openshift.io/sa.scc.uid-range must define IDs no greater than %d: %v",
				maxUserNamespaceID,
				err,
			),
		}
	}

	groupRange := namespace.Annotations["openshift.io/sa.scc.supplemental-groups"]
	if groupRange == "" {
		groupRange = uidRange
	}
	if err := validateUserNamespaceIDRange(groupRange); err != nil {
		return &podSecurityValidationError{
			reason: invalidNamespaceIDRangeReason,
			message: fmt.Sprintf(
				"namespace annotation openshift.io/sa.scc.supplemental-groups must define IDs no greater than %d: %v",
				maxUserNamespaceID,
				err,
			),
		}
	}

	return nil
}

func validateUserNamespaceIDRange(value string) error {
	if value == "" {
		return errors.New("annotation is empty")
	}

	for _, block := range strings.Split(value, ",") {
		start, end, err := parseIDRange(strings.TrimSpace(block))
		if err != nil {
			return err
		}
		if start == 0 {
			return fmt.Errorf("range %q includes root as its default ID", block)
		}
		if start > maxUserNamespaceID || end > maxUserNamespaceID {
			return fmt.Errorf("range %q exceeds the supported maximum", block)
		}
	}

	return nil
}

func parseIDRange(value string) (uint64, uint64, error) {
	if startValue, lengthValue, found := strings.Cut(value, "/"); found {
		start, err := strconv.ParseUint(startValue, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse range %q: %w", value, err)
		}
		length, err := strconv.ParseUint(lengthValue, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse range %q: %w", value, err)
		}
		if length == 0 || start > math.MaxUint64-(length-1) {
			return 0, 0, fmt.Errorf("range %q has an invalid length", value)
		}
		return start, start + length - 1, nil
	}

	if startValue, endValue, found := strings.Cut(value, "-"); found {
		start, err := strconv.ParseUint(startValue, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse range %q: %w", value, err)
		}
		end, err := strconv.ParseUint(endValue, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse range %q: %w", value, err)
		}
		if end < start {
			return 0, 0, fmt.Errorf("range %q ends before it starts", value)
		}
		return start, end, nil
	}

	return 0, 0, fmt.Errorf("range %q must use start/length or start-end format", value)
}
