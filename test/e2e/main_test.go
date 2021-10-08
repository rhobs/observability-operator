package e2e

import (
	"context"
	"log"
	"os"
	prometheus_operator "rhobs/monitoring-stack-operator/pkg/controllers/prometheus-operator"
	"rhobs/monitoring-stack-operator/pkg/operator"
	"rhobs/monitoring-stack-operator/test/e2e/framework"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	f *framework.Framework

	// TODO(fpetkovski): change once we are able to deploy the operator to a different namespace
	e2eTestNamespace = "default"
)

func TestMain(m *testing.M) {
	setLogger()
	op := createOperator()
	setupFramework(op)

	go runOperator(op, ctrl.SetupSignalHandler())
	m.Run()
	os.Exit(0)
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
	opts := zap.Options{}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

func createOperator() *operator.Operator {
	op, err := operator.New("", prometheus_operator.Options{
		Namespace:  e2eTestNamespace,
		AssetsPath: "../../assets/prometheus-operator/",
		DeployCRDs: true,
	})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	return op
}
