/*
Copyright 2021 Flant JSC

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

package check

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StatusSeries_Add(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		before  int
		wantErr bool
	}{
		{name: "1 -> 0/0", size: 0, before: 0, wantErr: true},
		{name: "1 -> 0/1", size: 1, before: 0, wantErr: false},
		{name: "1 -> 0/2", size: 2, before: 0, wantErr: false},
		{name: "1 -> 1/2", size: 2, before: 1, wantErr: false},
		{name: "1 -> 2/2", size: 2, before: 2, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := NewStatusSeries(tt.size)
			for i := 1; i <= tt.before; i++ {
				ss.Add(Unknown)
			}

			if err := ss.Add(Unknown); (err != nil) != tt.wantErr {
				t.Errorf("StatusSeries.Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_StatusSeries_Stats(t *testing.T) {
	type data struct {
		size int
		add  []Status
	}
	tests := []struct {
		name string
		data data
		want Stats
	}{
		{
			name: "zeroes",
		}, {
			name: "0/1",
			data: data{
				size: 1,
			},
			want: Stats{Expected: 1},
		}, {
			name: "1/1 up",
			data: data{
				size: 1,
				add:  []Status{Up},
			},
			want: Stats{Expected: 1, Up: 1},
		}, {
			name: "1/1 down",
			data: data{
				size: 1,
				add:  []Status{Down},
			},
			want: Stats{Expected: 1, Down: 1},
		}, {
			name: "1/1 unknown",
			data: data{
				size: 1,
				add:  []Status{Unknown},
			},
			want: Stats{Expected: 1, Unknown: 1},
		}, {
			name: "1/5 up",
			data: data{
				size: 5,
				add:  []Status{Up},
			},
			want: Stats{Expected: 5, Up: 1},
		}, {
			name: "1/5 down",
			data: data{
				size: 5,
				add:  []Status{Down},
			},
			want: Stats{Expected: 5, Down: 1},
		}, {
			name: "1/5 unknown",
			data: data{
				size: 5,
				add:  []Status{Unknown},
			},
			want: Stats{Expected: 5, Unknown: 1},
		}, {
			name: "3/5 up",
			data: data{
				size: 5,
				add:  []Status{Up, Up, Up},
			},
			want: Stats{Expected: 5, Up: 3},
		}, {
			name: "3/5 down",
			data: data{
				size: 5,
				add:  []Status{Down, Down, Down},
			},
			want: Stats{Expected: 5, Down: 3},
		}, {
			name: "3/5 unknown",
			data: data{
				size: 5,
				add:  []Status{Unknown, Unknown, Unknown},
			},
			want: Stats{Expected: 5, Unknown: 3},
		}, {
			name: "3/5 mixed",
			data: data{
				size: 5,
				add:  []Status{Up, Down, Unknown},
			},
			want: Stats{Expected: 5, Up: 1, Down: 1, Unknown: 1},
		}, {
			name: "5/5 mixed",
			data: data{
				size: 5,
				add:  []Status{Down, Unknown, Up, Down, Unknown},
			},
			want: Stats{Expected: 5, Up: 1, Down: 2, Unknown: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// prepare the data
			ss := NewStatusSeries(tt.data.size)
			for _, s := range tt.data.add {
				ss.Add(s)
			}

			if got := ss.Stats(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StatusSeries.Stats() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_StatusSeries_Merge(t *testing.T) {
	type args struct {
		dstSize int
		dstAdd  []Status
		srcSize int
		srcAdd  []Status
	}
	tests := []struct {
		name      string
		wantErr   bool
		args      args
		wantStats Stats
	}{

		{
			name:    "nodata 0/0 + 0/0",
			wantErr: false,
		},
		{
			name:    "nodata 0/1 + 0/1",
			wantErr: false,
			args: args{
				dstSize: 1,
				srcSize: 1,
			},
			wantStats: Stats{Expected: 1},
		},
		{
			name:    "nodata 0/5 + 0/5",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
			},
			wantStats: Stats{Expected: 5},
		},
		{
			name:    "nodata 0/5 + filled one",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				srcAdd:  []Status{Up, Up, Down, Down, Up},
			},
			wantStats: Stats{
				Expected: 5,
				Up:       3,
				Down:     2,
			},
		},
		{
			name:    "filled one + nodata 0/5",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Up, Down, Down, Up},
			},
			wantStats: Stats{
				Expected: 5,
				Up:       3,
				Down:     2,
			},
		},
		{
			name:    "One up 1/1 + 1/1 up",
			wantErr: false,
			args: args{
				dstSize: 1,
				srcSize: 1,
				dstAdd:  []Status{Up},
				srcAdd:  []Status{Up},
			},
			wantStats: Stats{
				Expected: 1,
				Up:       1,
			},
		},
		{
			name:    "All up 3/5 + 3/5 up",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Up, Up},
				srcAdd:  []Status{Up, Up, Up},
			},
			wantStats: Stats{
				Expected: 5,
				Up:       3,
			},
		},
		{
			name:    "Up wins unknown from source 3/5 up + 3/5 unknown",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Unknown, Unknown, Unknown},
				srcAdd:  []Status{Up, Up, Up},
			},
			wantStats: Stats{
				Expected: 5,
				Up:       3,
			},
		},
		{
			name:    "Up wins unknown from dest 3/5 unknown + 3/5 up",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Up, Up},
				srcAdd:  []Status{Unknown, Unknown, Unknown},
			},
			wantStats: Stats{
				Expected: 5,
				Up:       3,
			},
		},
		{
			name:    "Up wins unknown when mixed 3/5 ↑? + 3/5 ↑?",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Unknown, Up},
				srcAdd:  []Status{Unknown, Up, Unknown},
			},
			wantStats: Stats{
				Expected: 5,
				Up:       3,
			},
		},
		{
			name:    "Down everywhere 3/5 ↓ + 3/5 ↓",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Down, Down, Down},
				srcAdd:  []Status{Down, Down, Down},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
			},
		},
		{
			name:    "Down wins when mixed 3/5 ↑↓? + 3/5 ↓?↓",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Down, Unknown},
				srcAdd:  []Status{Down, Unknown, Down},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
			},
		},

		{
			name:    "Source with extra 3/5 ... + 4/5 with down",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Down, Unknown},
				srcAdd:  []Status{Down, Unknown, Down, Down},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     4,
			},
		},
		{
			name:    "Source with extra 3/5 ... + 4/5 with unknown",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Down, Unknown},
				srcAdd:  []Status{Down, Unknown, Down, Unknown},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
				Unknown:  1,
			},
		},
		{
			name:    "Source with extra 3/5 ... + 4/5 with up",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Down, Unknown},
				srcAdd:  []Status{Down, Unknown, Down, Up},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
				Up:       1,
			},
		},

		{
			name:    "Destination with extra 4/5 with down + 3/5 ...",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Down, Unknown, Down},
				srcAdd:  []Status{Down, Unknown, Down},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     4,
			},
		},
		{
			name:    "Destination with extra 4/5 with unknown + 3/5 ...",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Down, Unknown, Unknown},
				srcAdd:  []Status{Down, Unknown, Down},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
				Unknown:  1,
			},
		},
		{
			name:    "Destination with extra 4/5 with up + 3/5 ...",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Down, Unknown, Up},
				srcAdd:  []Status{Down, Unknown, Down},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
				Up:       1,
			},
		},
		{
			name:    "All the same 3/5 down + 4/5 up",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Down, Down, Down},
				srcAdd:  []Status{Up, Up, Up, Up},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
				Up:       1,
			},
		},
		{
			name:    "All the same 4/5 up + 3/5 down",
			wantErr: false,
			args: args{
				dstSize: 5,
				srcSize: 5,
				dstAdd:  []Status{Up, Up, Up, Up},
				srcAdd:  []Status{Down, Down, Down},
			},
			wantStats: Stats{
				Expected: 5,
				Down:     3,
				Up:       1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup series data
			dst := NewStatusSeries(tt.args.dstSize)
			src := NewStatusSeries(tt.args.srcSize)
			for _, s := range tt.args.dstAdd {
				dst.Add(s)
			}
			for _, s := range tt.args.srcAdd {
				src.Add(s)
			}

			// assert merging error
			if err := dst.Merge(src); (err != nil) != tt.wantErr {
				t.Errorf("dst.Merge() error = %v, wantErr %v", err, tt.wantErr)
			}

			// assert resulting counters
			got := dst.Stats()
			if !reflect.DeepEqual(got, tt.wantStats) {
				t.Errorf("dst.Merge() and Stats(): got=%v, want=%v", got, tt.wantStats)
			}
		})
	}
}

func Test_StatusSeries_Clean(t *testing.T) {
	// setup
	size := 5
	ss := NewStatusSeries(size)
	ss.Add(Unknown)
	ss.Add(Up)
	ss.Add(Down)
	ss.Add(Up)

	// action
	ss.Clean()

	// assert
	if ss.size() != size {
		t.Errorf("unexpected size after Clean(): got=%v, want=%v", ss.size(), size)
	}
	gotEmpty := ss.Stats()
	wantEmpty := Stats{Expected: size}
	if !reflect.DeepEqual(gotEmpty, wantEmpty) {
		t.Errorf("unexpected stats after Clean(): got=%v, want=%v", gotEmpty, wantEmpty)
	}
}

func Test_MergeStatusSeries(t *testing.T) {
	size := 10

	type args struct {
		a   *StatusSeries
		b   *StatusSeries
		ids []string
	}

	someData := NewStatusSeries(size)
	misSized := NewStatusSeries(size + 1)
	for i := 0; i < size; i++ {
		someData.Add(Up)
		misSized.Add(Up)
	}
	misSized.Add(Up) // one extra status

	tests := []struct {
		name    string
		args    args
		want    *StatusSeries
		wantErr bool
	}{
		{
			name: "all nodata returns nodata",
			args: args{
				a:   NewStatusSeries(size),
				b:   NewStatusSeries(size),
				ids: []string{"a", "b"},
			},
			want:    NewStatusSeries(size),
			wantErr: false,
		},
		{
			name: "data with nodata returns the data",
			args: args{
				a:   NewStatusSeries(size),
				b:   someData,
				ids: []string{"a", "b"},
			},
			want:    someData,
			wantErr: false,
		},
		{
			name: "missing id returns nodata",
			args: args{
				// no "a"
				b:   someData,
				ids: []string{"a", "b"},
			},
			want:    NewStatusSeries(size),
			wantErr: false,
		},
		{
			name: "size mismatch results in error",
			args: args{
				a:   misSized,
				b:   someData,
				ids: []string{"a", "b"},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := map[string]*StatusSeries{}
			if tt.args.a != nil {
				ss["a"] = tt.args.a
			}
			if tt.args.b != nil {
				ss["b"] = tt.args.b
			}

			got, err := MergeStatusSeries(size, ss, tt.args.ids)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error %v", err)
				}
				if got != nil {
					t.Errorf("expected nil in place of series data, got %v", got)
				}
				return
			}

			// assert the content
			assert.Equal(t, got.series, tt.want.series)
		})
	}
}
