/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package prom

import (
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
)

func TestParseTextDataWithAdditionalLabels(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		data    []byte
		ls      map[string]string
		want    []ParsedSeries
		wantErr bool
	}{
		{
			name: "blah",
			data: []byte(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total 1
`),
			ls: map[string]string{labels.InstanceName: "hostname1"},
			want: []ParsedSeries{
				{
					Labels:    labels.FromMap(map[string]string{labels.InstanceName: "hostname1", labels.MetricName: "han_metric_total"}),
					Value:     1,
					Timestamp: PromTimestamp(now),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTextDataWithAdditionalLabels(tt.data, now, tt.ls)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTextDataWithAdditionalLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTextDataWithAdditionalLabels() got = %v, want %v", got, tt.want)
			}
		})
	}
}
