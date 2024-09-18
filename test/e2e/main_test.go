package e2e

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/rhobs/observability-operator/pkg/operator"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

var (
	f *framework.Framework
)

const e2eTestNamespace = "e2e-tests"

var retain = flag.Bool("retain", false, "When set, the namespace in which tests are run will not be cleaned up")

func TestMain(m *testing.M) {
	flag.Parse()

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
	err := configv1.Install(scheme)
	if err != nil {
		return fmt.Errorf("failed to register configv1 to scheme %w", err)
	}
	k8sClient, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return err
	}

	isOpenshiftCluster, err := isOpenshiftCluster(k8sClient)
	if err != nil {
		return err
	}

	f = &framework.Framework{
		K8sClient:          k8sClient,
		Config:             cfg,
		Retain:             *retain,
		IsOpenshiftCluster: isOpenshiftCluster,
	}

	return nil
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

func isOpenshiftCluster(k8sClient client.Client) (bool, error) {
	clusterVersion := &configv1.ClusterVersion{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "version"}, clusterVersion)
	if err == nil {
		return true, nil
	} else if meta.IsNoMatchError(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("failed to get clusterversion %w", err)
	}
}
