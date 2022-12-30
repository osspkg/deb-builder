package packages

import (
	"fmt"
	"os"
	"strings"

	"github.com/dewep-online/deb-builder/pkg/utils"
)

var pkgArchAlias = map[string]string{
	"386": "i386",
}

func SplitVersion(v string) string {
	if strings.Contains(v, ":") {
		vv := strings.SplitN(v, ":", 2)
		if len(vv) == 2 {
			return vv[1]
		}
	}
	return v
}

func BuildName(dir, name, version, arch string) (string, string, string) {
	if v, ok := pkgArchAlias[arch]; ok {
		arch = v
	}
	version = strings.ReplaceAll(version, ":", ".")
	subver := ""
	callFunc := func() string {
		return fmt.Sprintf("%s/%s_%s%s_%s.deb", dir, name, version, subver, arch)
	}
	path := callFunc()
	revision := 1
	for {
		utils.FileStat(path, func(fi os.FileInfo) {
			subver = fmt.Sprintf("-%d", revision)
			path = callFunc()
			revision++
		})

		if !utils.FileExist(path) {
			return path, subver, arch
		}
	}
}
