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

	"github.com/rhobs/observability-operator/pkg/operator"
	"go.uber.org/zap/zapcore"

	k8sflag "k8s.io/component-base/cli/flag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// validateImages merges the passed images with the defaults and checks if any
// unknown image names are passed. If an unknown image is found, this raises an
// error.
func validateImages(images *k8sflag.MapStringString) (map[string]string, error) {
	res := operator.DefaultImages
	if !images.Empty() {
		imgs := *images.Map
		for k, v := range imgs {
			if _, ok := res[k]; !ok {
				return nil, fmt.Errorf(fmt.Sprintf("image %v is unknows", k))
			}
			res[k] = v
		}
	}
	return res, nil
}

func main() {
	var (
		namespace       string
		metricsAddr     string
		healthProbeAddr string

		setupLog = ctrl.Log.WithName("setup")
	)
	m := make(map[string]string)
	images := k8sflag.NewMapStringString(&m)

	flag.StringVar(&namespace, "namespace", "default", "The namespace in which the operator runs")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "The address the health probe endpoint binds to.")
	flag.Var(images, "images", fmt.Sprintf("Full images refs to use for containers managed by the operator. E.g thanos=quay.io/thanos/thanos:v0.33.0. Images used are %v", operator.ImagesUsed()))
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

	imgMap, err := validateImages(images)
	if err != nil {
		setupLog.Error(err, "cannot create a new operator")
		os.Exit(1)
	}

	op, err := operator.New(
		operator.NewOperatorConfiguration(
			operator.WithMetricsAddr(metricsAddr),
			operator.WithHealthProbeAddr(healthProbeAddr),
			operator.WithPrometheusImage(imgMap["prometheus"]),
			operator.WithAlertmanagerImage(imgMap["alertmanager"]),
			operator.WithThanosSidecarImage(imgMap["thanos"]),
			operator.WithThanosQuerierImage(imgMap["thanos"]),
		))
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
