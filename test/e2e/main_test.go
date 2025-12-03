package e2e

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/rhobs/observability-operator/pkg/operator"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

var (
	f *framework.Framework
)

const e2eTestNamespace = "e2e-tests"

var (
	retain            = flag.Bool("retain", false, "When set, the namespace in which tests are run will not be cleaned up")
	operatorInstallNS = flag.String("operatorInstallNS", "openshift-operator", "The namespace where the operator is installed")
)

func TestMain(m *testing.M) {
	flag.Parse()

	// Setup controller-runtime logger to avoid warning messages
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Deferred calls are not executed on os.Exit from TestMain.
	// As a workaround, we call another function in which we can add deferred calls.
	// http://blog.englund.nu/golang,/testing/2017/03/12/using-defer-in-testmain.html
	code := main(m)
	os.Exit(code)
}

func main(m *testing.M) int {
	if err := setupFramework(); err != nil {
		log.Println(err)
		return 1
	}

	cleanup, err := createNamespace(e2eTestNamespace)
	if err != nil {
		log.Println(err)
		return 1
	}
	if !*retain {
		defer cleanup()
	}

	exitCode := m.Run()

	tests := []testing.InternalTest{
		{
			Name: "NoReconcilationErrors",
			F: func(t *testing.T) {
				// see: https://github.com/rhobs/observability-operator/issues/200
				t.Skip("skipping reconciliation error test until #200 is fixed")
				f.AssertNoReconcileErrors(t)
			},
		},
		{
			// Kubernetes will emit events with reason=OwnerRefInvalidNamespace
			// if the operator defines invalid owner references.
			// See:
			// - https://kubernetes.io/docs/concepts/architecture/garbage-collection/#owners-dependents
			// - https://issues.redhat.com/browse/COO-117
			Name: "NoOwnerRefInvalidNamespaceReasonEvent",
			F: func(t *testing.T) {
				f.AssertNoEventWithReason(t, "OwnerRefInvalidNamespace")
			},
		},
	}

	log.Println("=== Running post e2e test validations ===")
	if !testing.RunTests(func(_, _ string) (bool, error) { return true, nil }, tests) {
		return 1
	}

	return exitCode
}

func setupFramework() error {
	cfg := config.GetConfigOrDie()
	scheme := operator.NewScheme(&operator.OperatorConfiguration{})
	err := olmv1alpha1.AddToScheme(scheme)
	if err != nil {
		return fmt.Errorf("failed to register olmv1alpha1 to scheme %w", err)
	}
	err = configv1.Install(scheme)
	if err != nil {
		return fmt.Errorf("failed to register configv1 to scheme %w", err)
	}
	k8sClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return err
	}

	f = &framework.Framework{
		K8sClient:         k8sClient,
		Config:            cfg,
		Retain:            *retain,
		OperatorNamespace: *operatorInstallNS,
	}

	return f.Setup()
}

func createNamespace(name string) (func(), error) {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := f.K8sClient.Create(context.Background(), ns); err != nil {
		return nil, err
	}

	cleanup := func() {
		f.K8sClient.Delete(context.Background(), ns)
	}

	return cleanup, nil
}
