package e2e

import (
	"context"
	"log"
	"os"
	"rhobs/monitoring-stack-operator/pkg/operator"
	"rhobs/monitoring-stack-operator/test/e2e/framework"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	f *framework.Framework
)

const e2eTestNamespace = "e2e-tests"

func TestMain(m *testing.M) {

	// Deferred calls are not executed on os.Exit from TestMain.
	// As a workaround, we call another function in which we can add deferred calls.
	// http://blog.englund.nu/golang,/testing/2017/03/12/using-defer-in-testmain.html
	code := main(m)
	os.Exit(code)
}

func main(m *testing.M) int {
	// setLogger()
	if err := setupFramework(); err != nil {
		log.Println(err)
		return 1
	}

	cleanup, err := createNamespace(e2eTestNamespace)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer cleanup()
	return m.Run()
}

func setupFramework() error {

	k8sClient, err := client.New(config.GetConfigOrDie(), client.Options{
		Scheme: operator.NewScheme(),
	})
	if err != nil {
		return err
	}

	f = &framework.Framework{
		K8sClient: k8sClient,
	}

	return nil
}

func createNamespace(name string) (func(), error) {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
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
