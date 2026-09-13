/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package control

import (
	"bytes"
	"math"
	"os"
	"strings"
	"text/template"

	"github.com/osspkg/pkg-build/pkg/config"
)

const descriptionMaxLen = 70

type (
	Control struct {
		conf config.Config
		size int64
		arch string
	}
	modelControl struct {
		Package      string
		Source       string
		Version      string
		Architecture string
		Maintainer   string
		Size         int64
		Depends      string
		Section      string
		Priority     string
		Homepage     string
		Description  string
	}
)

func NewControl(conf config.Config) *Control {
	return &Control{
		conf: conf,
	}
}

func (v *Control) DataSize(s int64) {
	v.size = int64(math.Round(float64(s) / 1024))
	if v.size == 0 {
		v.size = 1
	}
}
func (v *Control) Arch(arch string) {
	v.arch = arch
}

func (v *Control) Save(dir string, subver string) (string, error) {
	buf := &bytes.Buffer{}
	controlFile := dir + "/control"
	model := modelControl{
		Package:      v.conf.Package,
		Source:       v.conf.Source,
		Version:      v.conf.Version + subver,
		Architecture: v.arch,
		Maintainer:   v.conf.Maintainer,
		Size:         v.size,
		Section:      v.conf.Section,
		Priority:     v.conf.Priority,
		Homepage:     v.conf.Homepage,
		Depends: func() string {
			return strings.Join(v.conf.Control.Depends, ", ")
		}(),
		Description: formatDescription(v.conf.Description),
	}

	buf.Reset()

	if err := template.Must(template.New("").Parse(controlTmpl)).Execute(buf, model); err != nil {
		return controlFile, err
	}

	return controlFile, os.WriteFile(controlFile, buf.Bytes(), 0644)
}

func formatDescription(paragraphs []string) string {
	lines := make([]string, 0, len(paragraphs))
	for index, paragraph := range paragraphs {
		if index > 0 {
			lines = append(lines, ".")
		}

		words := strings.Fields(paragraph)
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
				continue
			}
			if len(line)+1+len(word) > descriptionMaxLen {
				lines = append(lines, line)
				line = word
				continue
			}
			line += " " + word
		}
		if line != "" {
			lines = append(lines, line)
		}
	}

	var description strings.Builder
	for index, line := range lines {
		if index > 0 {
			description.WriteString("\n ")
		}
		description.WriteString(line)
	}
	return description.String()
}

var controlTmpl = `Package: {{.Package}}
Source: {{.Source}}
Version: {{.Version}}
Architecture: {{.Architecture}}
Maintainer: {{.Maintainer}}
Installed-Size: {{.Size}}
Depends: {{.Depends}}
Section: {{.Section}}
Priority: {{.Priority}}
Homepage: {{.Homepage}}
Description: {{.Description}}
`
