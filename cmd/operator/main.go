/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rhobs/observability-operator/pkg/operator"
	"go.uber.org/zap/zapcore"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type kvarg map[string]string

func (a *kvarg) String() string {
	m := *a
	slice := m.asSlice()
	return strings.Join(slice, ",")
}

func (a *kvarg) Set(value string) error {
	m := *a
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		splitPair := strings.Split(pair, "=")
		if len(splitPair) != 2 {
			return fmt.Errorf("pair %q is malformed; key-value pairs must be in the form of \"key=value\"; multiple pairs must be comma-separated", value)
		}
		m[splitPair[0]] = splitPair[1]
	}
	return nil
}

func (a kvarg) asSlice() []string {
	pairs := []string{}
	for name, tag := range a {
		pairs = append(pairs, name+"="+tag)
	}
	return pairs
}

func (a kvarg) asMap() map[string]string {
	res := make(map[string]string, len(a))
	for k, v := range a {
		res[k] = v
	}
	return res
}

func (a *kvarg) Type() string {
	return "map[string]string"
}

func main() {
	var (
		namespace       string
		metricsAddr     string
		healthProbeAddr string

		setupLog = ctrl.Log.WithName("setup")
	)

	flag.StringVar(&namespace, "namespace", "default", "The namespace in which the operator runs")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "The address the health probe endpoint binds to.")
	images := kvarg{}
	flag.Var(&images, "images", "Images to use for containers managed by the observability-operator.")
	versions := kvarg{}
	flag.Var(&versions, "versions", "Version of containers managed by the observability-operator.")
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("running with arguments",
		"namespace", namespace,
		"metrics-bind-address", metricsAddr)

	op, err := operator.New(metricsAddr, healthProbeAddr, images.asMap(), versions.asMap())
	if err != nil {
		setupLog.Error(err, "cannot create a new operator")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	setupLog.Info("starting manager")
	if err := op.Start(ctx); err != nil {
		setupLog.Error(err, "terminating")
		os.Exit(1)
	}
}
