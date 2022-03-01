
# deb-builder

[![Coverage Status](https://coveralls.io/repos/github/dewep-online/deb-builder/badge.svg?branch=master)](https://coveralls.io/github/dewep-online/deb-builder?branch=master)
[![Release](https://img.shields.io/github/release/dewep-online/deb-builder.svg?style=flat-square)](https://github.com/dewep-online/deb-builder/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/dewep-online/deb-builder)](https://goreportcard.com/report/github.com/dewep-online/deb-builder)
[![CI](https://github.com/dewep-online/deb-builder/actions/workflows/ci.yml/badge.svg)](https://github.com/dewep-online/deb-builder/actions/workflows/ci.yml)

# install

```go
 go install github.com/dewep-online/deb-builder/cmd/deb-builder@latest
```

# create config file `.deb.yaml`

```bash
deb-builder config
```

example:

```yaml
package: demo-app # The name of the binary package.
source: demo # This field identifies the source package name.
version: 1:0.0.1 # The version number of a package. The format is: [epoch:]upstream_version.
architecture: # OS Architecture: all, 386, amd64, arm, arm64
  - 386
  - amd64
  - arm
  - arm64
maintainer: User Name <user.name@example.com> # The package maintainer’s name and email address. The name must come first, then the email address inside angle brackets <> (in RFC822 format).
homepage: http://example.com/ #The URL of the web site for this package, preferably (when applicable) the site from which the original source can be obtained and any additional upstream documentation or information may be found. 
description: # This field contains a description of the binary package, consisting of two parts, the synopsis or the short description, and the long description.
  - This is a demo utility
  - It performs some actions. Don't forget to update this text.
section: utils # This field specifies an application area into which the package has been classified: admin, cli-mono, comm, database, debug, devel, doc, editors, education, electronics, embedded, fonts, games, gnome, gnu-r, gnustep, graphics, hamradio, haskell, httpd, interpreters, introspection, java, javascript, kde, kernel, libdevel, libs, lisp, localization, mail, math, metapackages, misc, net, news, ocaml, oldlibs, otherosfs, perl, php, python, ruby, rust, science, shells, sound, tasks, tex, text, utils, vcs, video, web, x11, xfce, zope.
priority: optional # This field represents how important it is that the user have the package installed: required, important, standard, optional, extra.
control:
  depends: # This declares an absolute dependency. A package will not be configured unless all of the packages listed in its Depends field have been correctly configured (unless there is a circular dependency as described above).
    - systemd | supervisor
    - ca-certificates
  build: scripts/build.sh # This field defines the script for building the application from the source code. During the build, the name of the architecture is passed to the script. Example: sh scripts/build.sh amd64
  conffiles: # The list of package files that are configuration files, when updating, files from this list are not overwritten with new ones, unless this is specified separately;
    - /etc/demo/config.yaml
  preinst: scripts/preinst.sh # The script executed before installation.
  postinst: scripts/postinst.sh # The script executed after installation.
  prerm: scripts/prerm.sh # The script executed before removal.
  postrm: scripts/postrm.sh # The script executed after removal.
data: # A list of files that will be packaged during the build, where the file in the destination package is preceded by a colon, and the source file is indicated after it. A placeholder %arch% is available indicating the architecture.
  bin/demo: build/bin/demo_%arch% 
  etc/demo/config.yaml: configs/config.yaml 
```

# build deb package

```bash
deb-builder build --base-dir=/path_to_deb_release_directory/pool/main --tmp-dir=/path/to/build/directory
```

# build release repos

```bash
deb-builder build --release-dir=/path_to_deb_release_directory --private-key=/path_to_pgp_key/private.pgp --origin='Company Name' --label='Company Info'
```

Add to apt [amd64]

```bash
$ wget -qO - https://yourdomain/key.gpg | sudo apt-key add -
$ sudo tee /etc/apt/sources.list.d/yourdomain.list <<'EOF'
deb [arch=amd64] https://yourdomain/ stable main
EOF
$ sudo apt-get update
```

Add to apt [arm64]

```bash
$ wget -qO - https://yourdomain/key.gpg | sudo apt-key add -
$ sudo tee /etc/apt/sources.list.d/yourdomain.list <<'EOF'
deb [arch=arm64] https://yourdomain/ stable main
EOF
$ sudo apt-get update
```

# build pgp key

```bash
deb-builder pgp new --name='Company Name' --email='email@company' --comment='Comment about key' --path=/path_to_pgp_key
```

# Note

## compilation for ARM64 in Golang with support for CGO libraries

Installing the compiler

```bash
sudo apt install gcc-aarch64-linux-gnu
```

Building a project

```bash
GO111MODULE=on CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -a
```

