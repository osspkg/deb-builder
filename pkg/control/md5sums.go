/*
 *  Copyright (c) 2021-2023 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package control

import (
	"bytes"
	"os"
	"sort"
)

type (
	Md5Sums struct {
		data []md5Item
	}
	md5Item struct {
		File string
		Hash string
	}
)

func NewMd5Sums() *Md5Sums {
	return &Md5Sums{
		data: make([]md5Item, 0),
	}
}

func (v *Md5Sums) Add(filename, hash string) {
	v.data = append(v.data, md5Item{File: filename, Hash: hash})
}

func (v *Md5Sums) Save(dir string) (string, error) {
	buf := &bytes.Buffer{}
	sort.Slice(v.data, func(i, j int) bool {
		return v.data[i].File < v.data[j].File
	})
	for _, item := range v.data {
		if _, err := buf.WriteString(item.Hash + "  " + item.File + "\n"); err != nil {
			return "", err
		}
	}

	md5sumsFile := dir + "/md5sums"

	return md5sumsFile, os.WriteFile(md5sumsFile, buf.Bytes(), 0644)
}
