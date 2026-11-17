package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
)

// statusCapability is a capability whose only behavior is its status result.
type statusCapability struct {
	capability.Capability
	result capability.StatusResult
}

func (statusCapability) Name() string { return "status" }

func (s *statusCapability) UpdateStatus(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) capability.StatusResult {
	return s.result
}

func TestUpdateStatusClearsReconciledCondition(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(instance).WithStatusSubresource(instance).Build()
	status := &statusCapability{result: capability.StatusResult{Err: errors.New("transient")}}
	controller := observabilityInstallerController{client: c, capabilities: []capability.Capability{status}}

	controller.updateStatus(context.Background(), instance, nil)
	condition := apimeta.FindStatusCondition(instance.Status.Conditions, conditionTypeReconciled)
	require.NotNil(t, condition)
	require.Equal(t, metav1.ConditionFalse, condition.Status)

	status.result = capability.StatusResult{}
	controller.updateStatus(context.Background(), instance, nil)
	require.Nil(t, apimeta.FindStatusCondition(instance.Status.Conditions, conditionTypeReconciled))
}

type failingDesiredCapability struct {
	fakeCapability
	err error
}

func (f failingDesiredCapability) Desired(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) ([]client.Object, error) {
	return nil, f.err
}

func TestReconcileReportsPlanningError(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	planErr := errors.New("credential Secret missing")
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(instance).WithStatusSubresource(instance).Build()
	controller := observabilityInstallerController{
		client: c, apiReader: c, scheme: getScheme(), logger: logr.Discard(),
		capabilities: []capability.Capability{failingDesiredCapability{fakeCapability: fakeCapability{name: "tracing"}, err: planErr}},
	}

	_, err := controller.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(instance)})
	require.ErrorIs(t, err, planErr)

	got := &obsv1alpha1.ObservabilityInstaller{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKeyFromObject(instance), got))
	condition := apimeta.FindStatusCondition(got.Status.Conditions, conditionTypeReconciled)
	require.NotNil(t, condition)
	require.Equal(t, metav1.ConditionFalse, condition.Status)
	require.Contains(t, condition.Message, planErr.Error())
}

// resourceCapability plans one ConfigMap and inventories it plus a stale one.
// A planning error must preserve both without blocking another capability.
type resourceCapability struct {
	fakeCapability
	resource *corev1.ConfigMap
	err      error
}

func (r resourceCapability) Inventory(*obsv1alpha1.ObservabilityInstaller) []client.Object {
	stale := r.resource.DeepCopy()
	stale.Name += "-stale"
	return []client.Object{r.resource, stale}
}

func (r resourceCapability) Desired(context.Context, *obsv1alpha1.ObservabilityInstaller, client.Reader) ([]client.Object, error) {
	if r.err != nil {
		return nil, r.err
	}
	return []client.Object{r.resource}, nil
}

func TestReconcileIsolatesPlanningErrors(t *testing.T) {
	ctx := context.Background()
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	broken := configMap("test")
	broken.Name = "broken"
	healthy := configMap("test")
	healthy.Name = "healthy"
	brokenStale := broken.DeepCopy()
	brokenStale.Name += "-stale"
	healthyStale := healthy.DeepCopy()
	healthyStale.Name += "-stale"
	firstErr := errors.New("missing credential")
	secondErr := errors.New("invalid configuration")
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(instance, broken, brokenStale, healthyStale).
		WithStatusSubresource(instance).Build()
	controller := observabilityInstallerController{
		client: c, apiReader: c, scheme: getScheme(), logger: logr.Discard(),
		capabilities: []capability.Capability{
			resourceCapability{fakeCapability: fakeCapability{name: "broken"}, resource: broken, err: firstErr},
			resourceCapability{fakeCapability: fakeCapability{name: "healthy"}, resource: healthy},
			failingDesiredCapability{fakeCapability: fakeCapability{name: "also-broken"}, err: secondErr},
		},
	}

	_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(instance)})
	require.ErrorIs(t, err, firstErr)
	require.ErrorIs(t, err, secondErr)
	for _, obj := range []*corev1.ConfigMap{broken, brokenStale, healthy} {
		require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(obj), &corev1.ConfigMap{}), obj.Name)
	}
	require.Error(t, c.Get(ctx, client.ObjectKeyFromObject(healthyStale), &corev1.ConfigMap{}))
	got := &obsv1alpha1.ObservabilityInstaller{}
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(instance), got))
	condition := apimeta.FindStatusCondition(got.Status.Conditions, conditionTypeReconciled)
	require.NotNil(t, condition)
	require.Contains(t, condition.Message, "broken")
	require.Contains(t, condition.Message, "also-broken")
}

func TestReconcileContinuesAfterExecutionErrors(t *testing.T) {
	ctx := context.Background()
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	failed := configMap("test")
	failed.Name = "failed"
	healthy := configMap("test")
	healthy.Name = "healthy"
	failedStale := failed.DeepCopy()
	failedStale.Name += "-stale"
	healthyStale := healthy.DeepCopy()
	healthyStale.Name += "-stale"
	deleteErr := errors.New("delete failed")
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(instance, failedStale, healthyStale).
		WithStatusSubresource(instance).WithInterceptorFuncs(interceptor.Funcs{
		Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			if obj.GetName() == failedStale.Name {
				return deleteErr
			}
			return c.Delete(ctx, obj, opts...)
		},
	}).Build()
	controller := observabilityInstallerController{
		client: c, apiReader: c, scheme: getScheme(), logger: logr.Discard(),
		capabilities: []capability.Capability{
			resourceCapability{fakeCapability: fakeCapability{name: "failed"}, resource: failed},
			resourceCapability{fakeCapability: fakeCapability{name: "healthy"}, resource: healthy},
		},
	}
	_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(instance)})
	require.ErrorIs(t, err, deleteErr)
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(healthy), &corev1.ConfigMap{}))
	require.Error(t, c.Get(ctx, client.ObjectKeyFromObject(healthyStale), &corev1.ConfigMap{}))
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(failedStale), &corev1.ConfigMap{}))
}

func TestDeletingInstallerDoesNotApplyAnotherInstallersSharedResource(t *testing.T) {
	ctx := context.Background()
	now := metav1.Now()
	deleting := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{
		Name: "deleting", Namespace: "test", DeletionTimestamp: &now, Finalizers: []string{finalizerName},
	}}
	other := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "test"}}
	shared := configMap("test")
	applyErr := errors.New("shared apply failed")
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(deleting, other, shared).
		WithStatusSubresource(deleting, other).WithInterceptorFuncs(interceptor.Funcs{
		Apply: func(context.Context, client.WithWatch, runtime.ApplyConfiguration, ...client.ApplyOption) error {
			return applyErr
		},
	}).Build()
	controller := observabilityInstallerController{
		client: c, apiReader: c, scheme: getScheme(), logger: logr.Discard(),
		capabilities: []capability.Capability{fakeCapability{name: "shared", shared: []client.Object{shared}, enabledFor: "other"}},
	}
	_, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(deleting)})
	require.NoError(t, err)
	got := &obsv1alpha1.ObservabilityInstaller{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(deleting), got); err == nil {
		require.NotContains(t, got.Finalizers, finalizerName)
	} else {
		require.True(t, apierrors.IsNotFound(err), "unexpected get error: %v", err)
	}
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(shared), &corev1.ConfigMap{}))
}

func TestReconcileReportsReconcilerError(t *testing.T) {
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"}}
	deleteErr := errors.New("delete failed")
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(instance).WithStatusSubresource(instance).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				if _, ok := obj.(*otelv1beta1.OpenTelemetryCollector); ok {
					return deleteErr
				}
				return c.Delete(ctx, obj, opts...)
			},
		}).Build()
	controller := observabilityInstallerController{
		client: c, apiReader: c, scheme: getScheme(), logger: logr.Discard(),
		capabilities: capabilities(Options{}),
	}

	_, err := controller.Reconcile(context.Background(), reconcile.Request{NamespacedName: client.ObjectKeyFromObject(instance)})
	require.ErrorIs(t, err, deleteErr)

	got := &obsv1alpha1.ObservabilityInstaller{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKeyFromObject(instance), got))
	condition := apimeta.FindStatusCondition(got.Status.Conditions, conditionTypeReconciled)
	require.NotNil(t, condition)
	require.Equal(t, metav1.ConditionFalse, condition.Status)
	require.Contains(t, condition.Message, deleteErr.Error())
}
