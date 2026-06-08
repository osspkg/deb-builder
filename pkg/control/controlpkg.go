/*
 *  Copyright (c) 2021-2026 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package control

type Pkg struct {
	files []string
}

func NewControlPkg() *Pkg {
	return &Pkg{
		files: make([]string, 0),
	}
}

func (v *Pkg) AddFile(filepath ...string) {
	v.files = append(v.files, filepath...)
}

func (v *Pkg) List() []string {
	return v.files
}
