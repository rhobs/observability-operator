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
	"fmt"
	"math"
	"sort"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/index"
)

type seriesData struct {
	series labels.Labels
	data   []datapoint
}

type datapoint struct {
	timestamp int64
	value     float64
}

func NewRangeStorage() *rangeStorage {
	return &rangeStorage{
		series: make(seriesHashmap),
		data:   make(map[uint64]*seriesData),

		// observe: postings are ordered by the fact that we only ever add
		// after incrementing our nextPt value
		postings: NewmemPostings(),
	}
}

type rangeStorage struct {
	data   map[uint64]*seriesData
	series seriesHashmap
	nextPt uint64

	postings *memPostings
}

func (s *rangeStorage) LoadData(points []ParsedSeries) error {
	for _, point := range points {
		lblsHash := point.Labels.Hash()
		blockRef := s.series.get(lblsHash, point.Labels)
		var block *seriesData
		if blockRef == nil {
			block = &seriesData{
				series: point.Labels,
			}
			s.data[s.nextPt] = block
			s.series.set(lblsHash, &seriesRef{lset: point.Labels, index: s.nextPt})
			s.postings.Add(s.nextPt, point.Labels)

			if s.nextPt == math.MaxUint64 {
				// pretty unlikely to happen, but check just in case
				return fmt.Errorf("so much data, so few bits...")
			}
			s.nextPt++
		} else {
			block = s.data[blockRef.index]
		}

		needSort := false
		if len(block.data) > 0 {
			// in the unlikely event that we go backwards in time or have two data points
			// in a single batch that are out of order, check if we need to sort
			lastTime := block.data[len(block.data)-1].timestamp
			if point.Timestamp < lastTime {
				needSort = true
			}
		}
		datapt := datapoint{timestamp: point.Timestamp, value: point.Value}
		block.data = append(block.data, datapt)

		if needSort {
			// find the insertion point, shift things forward, and slot in the data point
			insertionPt := sort.Search(len(block.data)-1, func(ind int) bool {
				return datapt.timestamp > block.data[ind].timestamp
			})
			copy(block.data[insertionPt+1:], block.data[insertionPt:len(block.data)-1])
			block.data[insertionPt] = datapt
		}
	}

	return nil
}

func (s *rangeStorage) Clean(olderThan int64) {
	postingsToClean := make(map[uint64]struct{})

	for ref, block := range s.data {
		if len(block.data) == 0 {
			// oops
			postingsToClean[ref] = struct{}{}
		}
		if block.data[0].timestamp >= olderThan {
			// skip new enough blocks
			continue
		}

		keepInd := sort.Search(len(block.data), func(ind int) bool {
			return block.data[ind].timestamp >= olderThan
		})
		if keepInd == len(block.data) {
			postingsToClean[ref] = struct{}{}
		} else {
			keep := block.data[keepInd:]
			copy(block.data[:len(keep)], keep)
			block.data = block.data[:len(keep)]
		}
	}

	s.postings.Delete(postingsToClean)
	/*
		// first, find the first timestamp newer than our cutoff
		startKeeps := sort.Search(len(s.knownTimes), func(ind int) bool {
			return s.knownTimes[ind].timestamp >= olderThan
		})

		var keepOffset int
		if startKeeps == len(s.knownTimes) { // no data points new enough found
			keepOffset = len(s.data)
		} else {
			keepOffset = s.knownTimes[startKeeps].offset
		}
		s.knownTimes = s.knownTimes[startKeeps:]

		keptData := s.data[keepOffset:]
		s.dataOffset += len(s.data) - len(keptData)
		copy(s.data[:len(keptData)], keptData)
		s.data = s.data[:len(keptData)]

		s.postings.Clean(uint64(keepOffset))
		// seriesCache is magic, and takes care of itself
	*/
}

func (s *rangeStorage) Querier(ctx context.Context, minTime, maxTime int64) (storage.Querier, error) {
	// TODO(sollyross): we can short-circut here if we know the range of timestamps
	// stored in our storage (which we can store from calls to New)

	return &memQuerier{
		storage: s,
	}, nil
}

type memQuerier struct {
	storage *rangeStorage
}

func (q *memQuerier) SelectSorted(_ *storage.SelectParams, _ ...*labels.Matcher) (storage.SeriesSet, storage.Warnings, error) {
	panic("fix me")
}

func (q *memQuerier) Select(_ *storage.SelectParams, matchers ...*labels.Matcher) (storage.SeriesSet, storage.Warnings, error) {
	// we ignore hints b/c they're mostly just an optimization over large datasets

	// TODO(sollyross): we can use tricks from tsdb/querier to optimize a bit
	// if we need to for performance reasons
	sets := make([]index.Postings, len(matchers))
	for i, matcher := range matchers {
		// fast-case certain matchers
		if matcher.Type == labels.MatchEqual && matcher.Value != "" {
			// fast path: l="foo"
			sets[i] = q.storage.postings.Get(matcher.Name, matcher.Value)
			continue
		}

		// otherwise, figure out matching label values
		var matchingVals []index.Postings
		q.storage.postings.IterValues(matcher.Name, func(val string) {
			if matcher.Matches(val) {
				matchingVals = append(matchingVals, q.storage.postings.UnsafeGet(matcher.Name, val))
			}
		})
		// NB(sollyross): if we match the empty string, we also must match
		// series where the label is not set -- see
		// https://github.com/prometheus/prometheus/issues/3575.
		if matcher.Matches("") {
			// find all the places where this isn't set
			var withValueSet []index.Postings
			q.storage.postings.IterValues(matcher.Name, func(val string) {
				withValueSet = append(withValueSet, q.storage.postings.Get(matcher.Name, val))
			})
			matchingVals = append(matchingVals, index.Without(q.storage.postings.All(), index.Merge(withValueSet...)))
		}
		sets[i] = index.Merge(matchingVals...)
	}

	finalPostings := index.Intersect(sets...)
	return q.newSeriesSet(finalPostings), nil, nil
}

func (q *memQuerier) LabelValues(name string) ([]string, storage.Warnings, error) {
	// doesn't look like we need to error on missing label from the prom implementations
	var res []string
	q.storage.postings.IterValues(name, func(val string) {
		res = append(res, val)
	})
	return res, nil, nil
}

func (q *memQuerier) LabelNames() ([]string, storage.Warnings, error) {
	var res []string
	q.storage.postings.IterNames(func(name string) {
		res = append(res, name)
	})
	return res, nil, nil

}

func (q *memQuerier) Close() error {
	return nil
}

func (q *memQuerier) newSeriesSet(postings index.Postings) *seriesSet {
	return &seriesSet{
		postings: postings,
		storage:  q.storage,
	}
}

type seriesSet struct {
	postings index.Postings
	storage  *rangeStorage
}

func (s *seriesSet) Next() bool {
	return s.postings.Next()
}

func (s *seriesSet) At() storage.Series {
	block := s.storage.data[s.postings.At()]
	return &blockSeries{
		block: block,
		ind:   -1,
	}
}

func (s *seriesSet) Err() error {
	return s.postings.Err()
}

type blockSeries struct {
	block *seriesData
	ind   int
}

func (s *blockSeries) Iterator() storage.SeriesIterator {
	return s
}
func (s *blockSeries) Labels() labels.Labels {
	return s.block.series
}
func (s *blockSeries) Next() bool {
	s.ind++
	return s.ind < len(s.block.data)
}
func (s *blockSeries) Seek(targetTime int64) bool {
	s.ind = sort.Search(len(s.block.data), func(ind int) bool {
		return s.block.data[ind].timestamp >= targetTime
	})
	return s.ind < len(s.block.data)
}
func (s *blockSeries) At() (int64, float64) {
	pt := s.block.data[s.ind]
	return pt.timestamp, pt.value
}
func (s *blockSeries) Err() error {
	return nil
}

// flow is:
// - call next to populate data
// - read data with at
// - call next to see if there's another data point
//
// iterator starts at "before index 0", so the initial call to Next moves
// it to index zero, returning if there's data at index 0.

/*

func (q *memQuerier) newSeriesSet(postings index.Postings) *seriesSet {
	nextOffset := q.end
	if q.startInd+1 < len(q.storage.knownTimes) {
		nextOffset = q.storage.knownTimes[q.startInd+1]
	}
	return &seriesSet{
		postings: postings,
		storage: q.storage,

		end: q.end,
		timestampInd: q.startInd,
		lastTimestamp: q.start.timestamp,
		nextOffset: nextOffset,
	}
}

type seriesSet struct {
	postings index.Postings
	storage *rangeStorage
	end timestampOffset

	timestampInd int
	nextOffset timestampOffset
	lastTimestamp int64
}
func (s *seriesSet) Next() bool {
	if !s.postings.Next() {
		return false
	}
	ind := int(s.postings.At())
	if ind >= s.nextOffset.offset {
		s.timestampInd++
		if s.timestampInd >= len(s.storage.knownTimes) {
			// off the end, no data
			return false
		}
		lastOffset := s.storage.knownTimes[s.timestampInd]
		if lastOffset.timestamp > s.end.timestamp {
			// past end, no more data
			return false
		}
		s.lastTimestamp = lastOffset.timestamp

		nextInd := s.timestampInd+1
		if nextInd < len(s.storage.knownTimes) {
			s.nextOffset = s.storage.knownTimes[nextInd]
		} else {
			s.nextOffset = s.end
		}
	}
	return true
}
func (s *seriesSet) At() storage.Series {
	ind := s.postings.At()
	return &memSeries{
		datapoint: s.storage.data[int(ind)-s.storage.dataOffset],
		timestamp: s.lastTimestamp,
	}
}

func (s *seriesSet) Err() error {
	return s.postings.Err()
}

type memSeries struct {
	datapoint
	timestamp int64
}
func (s *memSeries) Labels() labels.Labels {
	return *s.series
}
func (s *memSeries) Iterator() chunkenc.Iterator {
	// TODO: do better than single point iterator?
	return &singlePtIterator{memSeries: s}
}

// flow is:
// - call next to populate data
// - read data with at
// - call next to see if there's another data point
//
// iterator starts at "before index 0", so the initial call to Next moves
// it to index zero, returning if there's data at index 0.
type singlePtIterator struct {
	*memSeries
	done bool // can't just consume memSeries b/c we don't have a consuming iterator interface
}
func (p *singlePtIterator) Next() bool {
	wasDone := p.done
	p.done = true
	return !wasDone
}
func (p singlePtIterator) Seek(targetTime int64) bool {
	found := false
	if targetTime <= p.timestamp {
		found = true // we're back at a valid time
	}
	p.done = true // either way, we've got only one data point
	return found
}
func (p singlePtIterator) At() (int64, float64) {
	return p.timestamp, p.val
}
func (p singlePtIterator) Err() error {
	return nil
}
*/
