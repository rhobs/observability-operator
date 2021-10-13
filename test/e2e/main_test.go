package e2e

import (
	"context"
	"log"
	"os"
	prometheus_operator "rhobs/monitoring-stack-operator/pkg/controllers/prometheus-operator"
	"rhobs/monitoring-stack-operator/pkg/operator"
	"rhobs/monitoring-stack-operator/test/e2e/framework"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
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
	setLogger()
	op, err := createOperator()
	if err != nil {
		log.Println(err)
		return 1
	}
	setupFramework(op)

	cleanup, err := createTestNamespace()
	if err != nil {
		log.Println(err)
		return 1
	}
	defer cleanup()

	go runOperator(op, ctrl.SetupSignalHandler())
	return m.Run()
}

func runOperator(op *operator.Operator, ctx context.Context) {
	if err := op.Start(ctx); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func setupFramework(op *operator.Operator) {
	f = &framework.Framework{
		K8sClient: op.GetClient(),
	}
}

func setLogger() {
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

func createOperator() (*operator.Operator, error) {
	op, err := operator.New("", prometheus_operator.Options{
		Namespace:  e2eTestNamespace,
		AssetsPath: "../../assets/prometheus-operator/",
		DeployCRDs: true,
	})
	if err != nil {
		return nil, err
	}

	return op, nil
}

func createTestNamespace() (func(), error) {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: e2eTestNamespace,
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
