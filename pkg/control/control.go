package control

import (
	"bytes"
	"deb-builder/pkg/config"
	"math"
	"os"
	"strings"
	"text/template"
)

const descriptionMaxLen = 70

type (
	Control struct {
		conf *config.Config
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

func NewControl(conf *config.Config) *Control {
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

func (v *Control) Save(dir string) (string, error) {
	buf := &bytes.Buffer{}
	controlFile := dir + "/control"
	model := modelControl{
		Package:      v.conf.Package,
		Source:       v.conf.Source,
		Version:      v.conf.Version,
		Architecture: v.arch,
		Maintainer:   v.conf.Maintainer,
		Size:         v.size,
		Section:      v.conf.Section,
		Priority:     v.conf.Priority,
		Homepage:     v.conf.Homepage,
		Depends: func() string {
			return strings.Join(v.conf.Control.Depends, ", ")
		}(),
		Description: func() string {
			for indx, s := range v.conf.Description {
				cur := 0
				words := strings.Split(s, " ")
				if indx > 0 {
					buf.WriteString("\n .\n ")
				}
				for _, word := range words {
					i, _ := buf.WriteString(word + " ")
					cur += i

					if cur >= descriptionMaxLen {
						buf.WriteString("\n ")
						cur = 0
					}
				}
			}

			return buf.String()
		}(),
	}

	buf.Reset()

	if err := template.Must(template.New("").Parse(controlTmpl)).Execute(buf, model); err != nil {
		return controlFile, err
	}

	return controlFile, os.WriteFile(controlFile, buf.Bytes(), 0644)
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
