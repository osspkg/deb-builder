
# pkg-build

[![Release](https://img.shields.io/github/release/osspkg/pkg-build.svg?style=flat-square)](https://github.com/osspkg/pkg-build/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/osspkg/pkg-build)](https://goreportcard.com/report/github.com/osspkg/pkg-build)
[![CI](https://github.com/osspkg/pkg-build/actions/workflows/ci.yml/badge.svg)](https://github.com/osspkg/pkg-build/actions/workflows/ci.yml)

# install

```go
 go install github.com/osspkg/pkg-build/cmd/pkg-build@latest
```

# create config file `.deb.yaml`

```shell
pkg-build config
```

example:

```yaml
package: demo-app # The name of the binary package.
source: demo # This field identifies the source package name.
version: 1:0.0.1 # The version number of a package. The format is: [epoch:]upstream_version. Or use `git` for build version by git commit.
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
  build: scripts/build.sh %arch% # This field defines the script for building the application from the source code. During the build, the name of the architecture is passed to the script. Example: sh scripts/build.sh amd64
  conffiles: # The list of package files that are configuration files, when updating, files from this list are not overwritten with new ones, unless this is specified separately;
    - /etc/demo/config.yaml
  preinst: scripts/preinst.sh # The script executed before installation.
  postinst: scripts/postinst.sh # The script executed after installation.
  prerm: scripts/prerm.sh # The script executed before removal.
  postrm: scripts/postrm.sh # The script executed after removal.
data: # A list of files that will be packaged during the build, where the file in the destination package is preceded by a colon, and the source file is indicated after it. A placeholder %arch% is available indicating the architecture.
  bin/demo: build/bin/demo_%arch% 
  etc/demo/config.yaml: configs/config.yaml 
  demo/file: 'c:write file content' 
  demo/dir: 'd:/build' 
  demo/dir1: 'e:/build/.*.(go|js)' 
```

data prefix:
- `c:`, `+`: create a file with the specified content
- `d:`, `~`: copy the entire contents of a directory into a package
- `e:`: copy the files with the specified regexp to the package

# build deb package

```shell
pkg-build build \
  --base-dir=/path_to_deb_release_directory/pool/main \
  --tmp-dir=/path/to/build/directory
```

or use env

```shell
DEB_STORAGE_BASE_DIR=/path_to_deb_release_directory/pool/main \
DEB_BUILD_DIR=/path/to/build/directory \
pkg-build build
```

# build release repos

```shell
pkg-build release \
  --release-dir=/path_to_deb_release_directory \
  --private-key=/path_to_pgp_key/private.pgp \
  --passwd='' \
  --origin='Company Name' \
  --label='Company Info'
```

or use env

```shell
DEB_STORAGE_BASE_DIR=/path_to_deb_release_directory/pool/main \
DEB_BUILD_DIR=/path/to/build/directory \
DEB_PGP_KEY=/path_to_pgp_key/private.pgp \
DEB_PGP_KEY_PASSWD='' \
DEB_RELEASE_ORIGIN='Company Name' \
DEB_RELEASE_LABEL='Company Info' \
pkg-build release
```

Verify the detached repository signature after importing the repository public key:

```bash
gpg --verify /path_to_deb_release_directory/dists/stable/Release.gpg \
  /path_to_deb_release_directory/dists/stable/Release
```

Add to apt [amd64]

```bash
curl -fsSL https://[yourdomain]/key.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/[yourdomain].gpg
sudo chmod a+r /etc/apt/keyrings/[yourdomain].gpg
sudo tee /etc/apt/sources.list.d/[yourdomain].list <<'EOF'
deb [arch=amd64 signed-by=/etc/apt/keyrings/[yourdomain].gpg] https://[yourdomain]/ stable main
EOF
sudo apt update
```

Add to apt [arm64]

```bash
curl -fsSL https://[yourdomain]/key.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/[yourdomain].gpg
sudo chmod a+r /etc/apt/keyrings/[yourdomain].gpg
sudo tee /etc/apt/sources.list.d/[yourdomain].list <<'EOF'
deb [arch=arm64 signed-by=/etc/apt/keyrings/[yourdomain].gpg] https://[yourdomain]/ stable main
EOF
sudo apt update
```

# build pgp key

```bash
pkg-build pgp new --name='Company Name' --email='email@company' --comment='Comment about key' --path=/path_to_pgp_key
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
