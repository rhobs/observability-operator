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
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/promql"
)

var testData = [][]byte{
	[]byte(`
cheese{sharpness="vermont",adj="extra"} 0.33
cheese{sharpness="vermont",adj="seriously"} 0.42
cheese{sharpness="sunnyvale"} 0.22
cheese{sharpness="secret cheese enclave"} 0.01
crackers{crunchy="yes",name="triscuit"} 37.5`),

	[]byte(`
cheese{sharpness="vermont",adj="extra"} 0.43
cheese{sharpness="vermont",adj="seriously"} 0.32
cheese{sharpness="secret cheese enclave"} 0.79
crackers{crunchy="yes",name="triscuit"} 9000.1
tea{variety="black",caffeine="yes"} 0.22
tea{variety="green",caffeine="yes"} 87.9
`),
}

func TestRangeQuery(t *testing.T) {
	start := time.Unix(0, 0)
	now := start

	storage := NewRangeStorage()

	for _, data := range testData {
		now = now.Add(1 * time.Second)
		points, err := ParseTextData(data, now)
		if err != nil {
			t.Errorf("unable to parse data: %v", err)
		}
		if err := storage.LoadData(points); err != nil {
			t.Errorf("unable to load load data: %v", err)
		}
	}

	end := now.Add(10 * time.Millisecond)

	// TODO: logger in opts
	// TODO: query logger?
	engine := promql.NewEngine(promql.EngineOpts{
		Timeout:    100000000 * time.Second, // this is what context is supposed to be for :-/
		MaxSamples: 1000,                    // TODO: find an ok value
	})

	testcases := []struct {
		qs                      string
		expectedValuesForLabels map[string][]float64
	}{
		{
			qs: `cheese{sharpness="vermont"}`,
			expectedValuesForLabels: map[string][]float64{
				`{__name__="cheese", adj="extra", sharpness="vermont"}`:     {0.33, 0.43},
				`{__name__="cheese", adj="seriously", sharpness="vermont"}`: {0.42, 0.32},
			},
		},
		{
			qs: `cheese{sharpness=~"s.*"}`,
			expectedValuesForLabels: map[string][]float64{
				`{__name__="cheese", sharpness="sunnyvale"}`:             {0.22, 0.22},
				`{__name__="cheese", sharpness="secret cheese enclave"}`: {0.01, 0.79},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.qs, func(tt *testing.T) {
			query, err := engine.NewRangeQuery(storage, tc.qs, start, end, 1*time.Second)
			if err != nil {
				t.Errorf("error creating query: %v", err)
			}

			ctx := context.TODO()
			res := query.Exec(ctx)
			defer query.Close()

			if res.Err != nil {
				t.Errorf("error running query: %v", res.Err)
			}

			if len(res.Warnings) > 0 {
				for _, warning := range res.Warnings {
					t.Logf("warning running query: %v", warning)
				}
			}
			// todo(han) test other types this doesn't really scale
			if res.Value.Type() != "matrix" {
				t.Errorf("expected matrix of data (vector over time), but got %s of %s", res.Value.Type(), res)
			}
			matrix, err := res.Matrix()
			if res.Err != nil {
				t.Errorf("error converting to matrix: %v", err)
			}
			for _, series := range matrix {
				s, ok := tc.expectedValuesForLabels[series.Metric.String()]
				if !ok {
					t.Errorf("Didn't expect this label combo: %v\n", series.Metric.String())
				}
				if len(s) != len(series.Points) {
					t.Errorf("mismatched points -- expected %v, got %v", s, series.Points)
					continue
				}
				for i, pt := range series.Points {
					if pt.V != s[i] {
						t.Errorf("Got: %v, want: %v at %s[%v]", pt.V, s[i], series.Metric, i)
					}
					expectedTime := PromTimestamp(start.Add(time.Duration(i+1) * time.Second))
					if pt.T != expectedTime {
						t.Errorf("got time %v, expected time %v", pt.T, expectedTime)
					}
				}
			}
		})
	}
}

func TestInstantQuery(t *testing.T) {
	now := time.Now()
	points, err := ParseTextData(testData[0], now)
	if err != nil {
		t.Errorf("invalid raw data: %v", err)
	}

	// TODO: logger in opts
	// TODO: query logger?
	engine := promql.NewEngine(promql.EngineOpts{
		Timeout:    100000000 * time.Second, // this is what context is supposed to be for :-/
		MaxSamples: 1000,                    // TODO: find an ok value
	})

	storage := NewRangeStorage()
	if err := storage.LoadData(points); err != nil {
		t.Errorf("unable to load data: %v", err)
	}

	testcases := []struct {
		qs                      string
		expectedValuesForLabels map[string]float64
	}{
		{
			qs: `cheese{sharpness="vermont"}`,
			expectedValuesForLabels: map[string]float64{
				`{__name__="cheese", adj="extra", sharpness="vermont"}`:     0.33,
				`{__name__="cheese", adj="seriously", sharpness="vermont"}`: 0.42,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.qs, func(tt *testing.T) {
			query, err := engine.NewInstantQuery(storage, `cheese{sharpness="vermont"}`, now)
			if err != nil {
				t.Errorf("error creating query: %v", err)
			}

			ctx := context.TODO()
			res := query.Exec(ctx)
			defer query.Close()

			if res.Err != nil {
				t.Errorf("error running query: %v", res.Err)
			}

			if len(res.Warnings) > 0 {
				for _, warning := range res.Warnings {
					t.Logf("warning running query: %v", warning)
				}
			}
			// todo(han) test other types this doesn't really scale
			vec, err := res.Vector()
			if res.Err != nil {
				t.Errorf("error converting to scalar: %v", err)
			}
			for _, v := range vec {
				s, ok := tc.expectedValuesForLabels[v.Metric.String()]
				if !ok {
					t.Errorf("Didn't expect this label combo: %v\n", v.Metric.String())
				}
				if v.V != s {
					t.Errorf("Got: %v, want: %v", v.V, s)
				}
			}
		})
	}
}
