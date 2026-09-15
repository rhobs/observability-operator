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
	"context"
	"flag"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/rhobs/observability-operator/pkg/operator"
)

func TestFlagWasSet(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	value := flags.Bool("openshift.enabled", false, "")
	if err := flags.Parse([]string{"--openshift.enabled=false"}); err != nil {
		t.Fatal(err)
	}
	if *value {
		t.Fatal("expected flag value to remain false")
	}
	if !flagWasSet(flags, "openshift.enabled") {
		t.Fatal("expected explicitly supplied false flag to be reported as set")
	}
	if flagWasSet(flags, "missing") {
		t.Fatal("unexpected match for missing flag")
	}
}

func TestSetupOpenShiftDisabled(t *testing.T) {
	data, enabled, err := setupOpenShift(context.Background(), false, true, logr.Discard())
	if err != nil {
		t.Fatalf("setupOpenShift() error = %v", err)
	}
	if enabled {
		t.Error("setupOpenShift() enabled = true, want false")
	}
	if !reflect.DeepEqual(data, openShiftData{}) {
		t.Errorf("setupOpenShift() data = %#v, want empty data", data)
	}
}

func TestDetectOpenShift(t *testing.T) {
	apiServer := &configv1.APIServer{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}}
	clusterVersion := &configv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "version"},
		Status: configv1.ClusterVersionStatus{
			Desired: configv1.Release{Version: "4.20.0"},
		},
	}
	invalidAPIServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.APIServerSpec{
			TLSSecurityProfile: &configv1.TLSSecurityProfile{Type: configv1.TLSProfileCustomType},
		},
	}
	wantData := openShiftData{
		TLSProfile: *configv1.TLSProfiles[configv1.TLSProfileIntermediateType],
		Version:    "4.20.0",
	}

	tests := []struct {
		name        string
		required    bool
		objects     []client.Object
		wantEnabled bool
		wantErr     bool
		wantData    openShiftData
	}{
		{
			name:        "detected",
			objects:     []client.Object{apiServer, clusterVersion},
			wantEnabled: true,
			wantData:    wantData,
		},
		{name: "API absent in auto mode"},
		{name: "API absent when required", required: true, wantErr: true},
		{
			name:    "invalid profile in auto mode",
			objects: []client.Object{invalidAPIServer},
			wantErr: true,
		},
		{
			name:     "invalid profile when required",
			required: true,
			objects:  []client.Object{invalidAPIServer},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().
				WithScheme(operator.NewOpenShiftScheme()).
				WithObjects(tt.objects...).
				Build()

			data, enabled, err := detectOpenShift(context.Background(), tt.required, logr.Discard(), c)
			if (err != nil) != tt.wantErr {
				t.Fatalf("detectOpenShift() error = %v, wantErr %v", err, tt.wantErr)
			}
			if enabled != tt.wantEnabled {
				t.Errorf("detectOpenShift() enabled = %v, want %v", enabled, tt.wantEnabled)
			}
			if !reflect.DeepEqual(data, tt.wantData) {
				t.Errorf("detectOpenShift() data = %#v, want %#v", data, tt.wantData)
			}
		})
	}
}
