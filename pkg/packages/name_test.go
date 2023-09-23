/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package packages

import "testing"

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
