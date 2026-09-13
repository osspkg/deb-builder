/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package packages

import (
	"sync"
	"testing"
)

func TestUnit_SplitVersion(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "Case1", args: args{v: "1:1.1.1"}, want: "1.1.1"},
		{name: "Case2", args: args{v: "1.1.1.1"}, want: "1.1.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitVersion(tt.args.v); got != tt.want {
				t.Errorf("SplitVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReserveBuildNameAllocatesUniqueRevisions(t *testing.T) {
	const workers = 16
	outputDir := t.TempDir()
	start := make(chan struct{})
	reservations := make([]*BuildNameReservation, workers)
	errors := make([]error, workers)
	var waitGroup sync.WaitGroup

	for index := range workers {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			<-start
			reservations[index], _, _, errors[index] = ReserveBuildName(outputDir, "demo", "1.2.3", "amd64", false)
		}(index)
	}
	close(start)
	waitGroup.Wait()

	paths := make(map[string]struct{}, workers)
	for index, reservation := range reservations {
		if errors[index] != nil {
			t.Fatalf("reserve %d: %v", index, errors[index])
		}
		if _, exists := paths[reservation.Path()]; exists {
			t.Fatalf("duplicate reservation path %q", reservation.Path())
		}
		paths[reservation.Path()] = struct{}{}
	}
	for _, reservation := range reservations {
		if err := reservation.Abort(); err != nil {
			t.Fatalf("abort reservation: %v", err)
		}
	}
}
