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
	"io"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
)

func PromTimestamp(normalTime time.Time) int64 {
	return normalTime.UnixNano() / int64(time.Millisecond/time.Nanosecond)
}

type ParsedSeries struct {
	Labels    labels.Labels
	Value     float64
	Timestamp int64
}

func ParseTextData(data []byte, nowish time.Time) ([]ParsedSeries, error) {
	return ParseTextDataWithAdditionalLabels(data, nowish, map[string]string{})
}

func ParseTextDataWithAdditionalLabels(data []byte, nowish time.Time, ls map[string]string) ([]ParsedSeries, error) {
	// prometheus time is milliseconds, cause
	nowAbouts := PromTimestamp(nowish)
	p := textparse.NewPromParser(data)
	metrics := make([]ParsedSeries, 0)
	for {
		et, err := p.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		switch et {
		case textparse.EntrySeries:
			_, optTimestamp, v := p.Series()
			var res labels.Labels

			p.Metric(&res)
			var timestamp int64
			if optTimestamp != nil {
				timestamp = *optTimestamp
			} else {
				timestamp = nowAbouts
			}

			// TODO(directxman12): seems like this is a rather big allocation, would
			// be nice if we could cut down on this
			lb := labels.NewBuilder(res)

			for k, v := range ls {
				lb.Set(k, v)
			}

			metrics = append(metrics, ParsedSeries{
				Labels:    lb.Labels(),
				Value:     v,
				Timestamp: timestamp,
			})
		}
	}
	return metrics, nil
}
