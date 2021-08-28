package control

import "github.com/dewep-online/deb-builder/pkg/config"

type ControlPkg struct {
	conf  *config.Config
	files []string
}

func NewControlPkg(conf *config.Config) *ControlPkg {
	return &ControlPkg{
		conf:  conf,
		files: make([]string, 0),
	}
}

func (v *ControlPkg) AddFile(filepath ...string) {
	v.files = append(v.files, filepath...)
}

func (v *ControlPkg) List() []string {
	return v.files
}
