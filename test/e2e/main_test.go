package e2e

import (
	"context"
	"flag"
	"log"
	"os"
	"testing"

	"github.com/rhobs/observability-operator/pkg/operator"
	"github.com/rhobs/observability-operator/test/e2e/framework"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	tests := []testing.InternalTest{{
		Name: "NoReconcilationErrors",
		F:    f.AssertNoReconcileErrors,
	}}

	log.Println("=== Running post e2e test validations ===")
	if !testing.RunTests(func(_, _ string) (bool, error) { return true, nil }, tests) {
		return 1
	}

	return exitCode
}

func setupFramework() error {
	cfg := config.GetConfigOrDie()
	k8sClient, err := client.New(cfg, client.Options{
		Scheme: operator.NewScheme(),
	})
	if err != nil {
		return err
	}

	f = &framework.Framework{
		K8sClient: k8sClient,
		Config:    cfg,
		Retain:    *retain,
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
