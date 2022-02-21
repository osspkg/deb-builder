package control

import "github.com/dewep-online/deb-builder/pkg/config"

type Pkg struct {
	conf  *config.Config
	files []string
}

func NewControlPkg(conf *config.Config) *Pkg {
	return &Pkg{
		conf:  conf,
		files: make([]string, 0),
	}
}

func (v *Pkg) AddFile(filepath ...string) {
	v.files = append(v.files, filepath...)
}

func (v *Pkg) List() []string {
	return v.files
}
