package observability

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	lokiv1 "github.com/grafana/loki/operator/api/loki/v1"
	tempov1alpha1 "github.com/grafana/tempo-operator/api/tempo/v1alpha1"
	clfv1 "github.com/openshift/cluster-logging-operator/api/observability/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/capability"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/logging"
	"github.com/rhobs/observability-operator/pkg/controllers/observability/tracing"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
)

// These tests cover ownership and deletion regressions.

func p1Scheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := getScheme()
	require.NoError(t, lokiv1.AddToScheme(scheme))
	require.NoError(t, clfv1.AddToScheme(scheme))
	return scheme
}

func p1OperatorConfig(name string) capability.OperatorConfig {
	return capability.OperatorConfig{Namespace: "operators", PackageName: name, Channel: "stable"}
}

func p1TempoSubscription() *olmv1alpha1.Subscription {
	return &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{Name: "tempo-product", Namespace: "operators", Labels: map[string]string{util.ResourceLabel: util.OpName}},
		Spec:       &olmv1alpha1.SubscriptionSpec{Package: "tempo-product"},
		Status:     olmv1alpha1.SubscriptionStatus{CurrentCSV: "tempo.v1"},
	}
}

func TestReviewP1ForeignLokiStackPreserved(t *testing.T) {
	ctx := context.Background()
	scheme := p1Scheme(t)
	// Logging is absent. The independently managed LokiStack happens to have
	// the name the logging capability would choose for this installer.
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "observability"}}
	foreign := &lokiv1.LokiStack{ObjectMeta: metav1.ObjectMeta{Name: "example-loki", Namespace: instance.Namespace}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(instance, foreign).Build()
	definitions := []capability.Definition{logging.New(logging.Options{})}
	recs, err := getReconcilers(ctx, c, instance, definitions, operatorsStatus{}, []obsv1alpha1.ObservabilityInstaller{*instance})
	require.NoError(t, err)
	for _, rec := range recs {
		require.NoError(t, rec.Reconcile(ctx, c, scheme))
	}
	err = c.Get(ctx, client.ObjectKeyFromObject(foreign), &lokiv1.LokiStack{})
	if apierrors.IsNotFound(err) {
		t.Fatal("unowned LokiStack was deleted even though logging was never enabled")
	}
	require.NoError(t, err)
}

func TestReviewP1ConcurrentDeletionKeepsSharedOperator(t *testing.T) {
	ctx := context.Background()
	scheme := p1Scheme(t)
	now := metav1.Now()
	first := obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "first", Namespace: "observability", UID: "first-uid", DeletionTimestamp: &now, Finalizers: []string{finalizerName}}}
	second := obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "second", Namespace: "observability", UID: "second-uid", DeletionTimestamp: &now, Finalizers: []string{finalizerName}}}
	first.Spec.Capabilities = &obsv1alpha1.CapabilitiesSpec{Tracing: &obsv1alpha1.TracingSpec{CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true}}}
	second.Spec.Capabilities = first.Spec.Capabilities.DeepCopy()
	// The first installer has no remaining operands; the second has not yet
	// reconciled and still owns this TempoStack.
	operand := &tempov1alpha1.TempoStack{ObjectMeta: metav1.ObjectMeta{
		Name: "second", Namespace: "observability", Finalizers: []string{tempov1alpha1.TempoFinalizer},
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&second, obsv1alpha1.GroupVersion.WithKind("ObservabilityInstaller"))},
	}}
	sub := p1TempoSubscription()
	csv := &olmv1alpha1.ClusterServiceVersion{ObjectMeta: metav1.ObjectMeta{Name: sub.Status.CurrentCSV, Namespace: sub.Namespace}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(operand, sub, csv).Build()
	definitions := []capability.Definition{tracing.New(tracing.Options{TempoOperator: p1OperatorConfig("tempo-product"), OpenTelemetryOperator: p1OperatorConfig("opentelemetry-product")})}
	recs, err := getReconcilers(ctx, c, &first, definitions, operatorsStatus{subs: []olmv1alpha1.Subscription{*sub}}, []obsv1alpha1.ObservabilityInstaller{first, second})
	require.NoError(t, err)
	for _, rec := range recs {
		err = rec.Reconcile(ctx, c, scheme)
		if errors.Is(err, errDeletionPending) {
			break
		}
		require.NoError(t, err)
	}
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(operand), &tempov1alpha1.TempoStack{}))
	subErr := c.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{})
	csvErr := c.Get(ctx, client.ObjectKeyFromObject(csv), &olmv1alpha1.ClusterServiceVersion{})
	if apierrors.IsNotFound(subErr) || apierrors.IsNotFound(csvErr) {
		t.Fatalf("Tempo operator was uninstalled while the other deleting installer still had a TempoStack: subscription=%v CSV=%v", subErr, csvErr)
	}
	require.NoError(t, subErr)
	require.NoError(t, csvErr)
}

func TestReviewP1KeepsOperatorForFinalizingOperand(t *testing.T) {
	ctx := context.Background()
	scheme := p1Scheme(t)
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{
		Name: "example", Namespace: "observability", UID: "example-uid", Finalizers: []string{finalizerName},
		DeletionTimestamp: &metav1.Time{Time: time.Now().Add(-time.Hour)},
	}}
	operand := &tempov1alpha1.TempoStack{ObjectMeta: metav1.ObjectMeta{
		Name: instance.Name, Namespace: instance.Namespace, Finalizers: []string{tempov1alpha1.TempoFinalizer},
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(instance, obsv1alpha1.GroupVersion.WithKind("ObservabilityInstaller"))},
	}}
	sub := p1TempoSubscription()
	csv := &olmv1alpha1.ClusterServiceVersion{ObjectMeta: metav1.ObjectMeta{Name: sub.Status.CurrentCSV, Namespace: sub.Namespace}}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(instance).WithObjects(instance, operand, sub, csv).Build()
	tracingDefinition := tracing.New(tracing.Options{TempoOperator: p1OperatorConfig("tempo-product"), OpenTelemetryOperator: p1OperatorConfig("opentelemetry-product")})
	// No watch discovery is needed for this deletion test.
	tracingDefinition.WatchTypes = nil
	controller := observabilityInstallerController{client: c, apiReader: c, scheme: scheme, logger: logr.Discard(), watches: newWatchRegistry(), capabilities: []capability.Definition{tracingDefinition}}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(instance)}
	result, err := controller.Reconcile(ctx, request)
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, result.RequeueAfter)
	var live obsv1alpha1.ObservabilityInstaller
	require.NoError(t, c.Get(ctx, request.NamespacedName, &live))
	require.Contains(t, live.Finalizers, finalizerName)
	var pending tempov1alpha1.TempoStack
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(operand), &pending))
	require.NotNil(t, pending.DeletionTimestamp)
	subErr := c.Get(ctx, client.ObjectKeyFromObject(sub), &olmv1alpha1.Subscription{})
	csvErr := c.Get(ctx, client.ObjectKeyFromObject(csv), &olmv1alpha1.ClusterServiceVersion{})
	if apierrors.IsNotFound(subErr) || apierrors.IsNotFound(csvErr) {
		t.Fatalf("Tempo operator was uninstalled while the TempoStack still had its finalizer: subscription=%v CSV=%v", subErr, csvErr)
	}
	require.NoError(t, subErr)
	require.NoError(t, csvErr)
}

func TestReviewP1UnmanagedLoggingPluginPreserved(t *testing.T) {
	ctx := context.Background()
	scheme := p1Scheme(t)
	instance := &obsv1alpha1.ObservabilityInstaller{ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "observability"}, Spec: obsv1alpha1.ObservabilityInstallerSpec{Capabilities: &obsv1alpha1.CapabilitiesSpec{
		Logging: &obsv1alpha1.LoggingSpec{
			CommonCapabilitiesSpec: obsv1alpha1.CommonCapabilitiesSpec{Enabled: true},
			LokiStack: &obsv1alpha1.LokiStackSpec{Size: obsv1alpha1.SizeOneXPico, StorageClassName: "standard", ObjectStorage: &obsv1alpha1.ObjectStorageSpec{
				S3: &obsv1alpha1.S3Spec{Bucket: "logs", Endpoint: "https://s3.example", AccessKeyID: "key", AccessKeySecret: obsv1alpha1.SecretKeySelector{Name: "credentials", Key: "secret"}},
			}},
		},
	}}}
	foreign := &uiv1alpha1.UIPlugin{ObjectMeta: metav1.ObjectMeta{Name: uiv1alpha1.LoggingPluginName}, Spec: uiv1alpha1.UIPluginSpec{Type: uiv1alpha1.TypeLogging, Logging: &uiv1alpha1.LoggingConfig{LokiStack: &uiv1alpha1.LokiStackReference{Name: "user-loki", Namespace: "user-logging"}}}}
	credentials := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: instance.Namespace}, Data: map[string][]byte{"secret": []byte("secret")}}
	definitions := []capability.Definition{logging.New(logging.Options{LokiOperator: p1OperatorConfig("loki-operator"), ClusterLoggingOperator: p1OperatorConfig("cluster-logging")})}
	// Externally managed subscriptions avoid unrelated OLM installation in this test.
	status := operatorsStatus{subs: []olmv1alpha1.Subscription{
		{Spec: &olmv1alpha1.SubscriptionSpec{Package: "loki-operator"}},
		{Spec: &olmv1alpha1.SubscriptionSpec{Package: "cluster-logging"}},
	}}
	// The fake client's Apply method cannot unmarshal the response into the
	// project's private apply wrapper. Route it through fake Patch instead;
	// this still uses the fake client's server-side apply implementation for
	// the plugin. Other operand applies are no-ops in this focused test.
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(foreign, credentials).WithInterceptorFuncs(interceptor.Funcs{
		Apply: func(ctx context.Context, backend client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
			data, err := json.Marshal(obj)
			if err != nil {
				return err
			}
			var patch unstructured.Unstructured
			if err := json.Unmarshal(data, &patch); err != nil {
				return err
			}
			if patch.GetKind() != "UIPlugin" {
				return nil
			}
			applyOpts := &client.ApplyOptions{}
			applyOpts.ApplyOptions(opts)
			return backend.Patch(ctx, &patch, client.Apply, &client.PatchOptions{FieldManager: applyOpts.FieldManager, Force: applyOpts.Force, DryRun: applyOpts.DryRun})
		},
	}).Build()
	reconcile := func() error {
		recs, err := getReconcilers(ctx, c, instance, definitions, status, []obsv1alpha1.ObservabilityInstaller{*instance})
		if err != nil {
			return err
		}
		for _, rec := range recs {
			if err := rec.Reconcile(ctx, c, scheme); err != nil {
				return err
			}
		}
		return nil
	}
	err := reconcile()
	// A conflict that refuses to adopt a foreign plugin is a safe outcome.
	if err != nil && !apierrors.IsConflict(err) {
		t.Fatalf("unexpected reconciliation error: %v", err)
	}
	var live uiv1alpha1.UIPlugin
	require.NoError(t, c.Get(ctx, client.ObjectKeyFromObject(foreign), &live))
	if live.Labels[util.PartOfLabel] == util.OpName || live.Spec.Logging == nil || live.Spec.Logging.LokiStack == nil || live.Spec.Logging.LokiStack.Name != "user-loki" || live.Spec.Logging.LokiStack.Namespace != "user-logging" {
		t.Errorf("unowned UIPlugin was adopted or changed: labels=%v, spec=%s", live.Labels, mustJSON(t, live.Spec))
	}
	instance.Spec.Capabilities.Logging.Enabled = false
	require.NoError(t, reconcile())
	err = c.Get(ctx, client.ObjectKeyFromObject(foreign), &uiv1alpha1.UIPlugin{})
	if apierrors.IsNotFound(err) {
		t.Error("disabling logging deleted the previously user-owned UIPlugin")
	} else {
		require.NoError(t, err)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return string(data)
}
