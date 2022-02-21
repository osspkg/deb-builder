package packages

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModel_Decode(t *testing.T) {
	type fields struct {
		Package      string
		Source       string
		Version      string
		Architecture string
		Maintainer   string
		Filename     string
		Size         int64
		MD5sum       string
		SHA1         string
		SHA256       string
		Raw          string
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				Package:      "steam-launcher",
				Source:       "steam",
				Version:      "1:1.0.0.74",
				Architecture: "all",
				Maintainer:   "Valve Corporation <linux@steampowered.com>",
				Filename:     "pool/steam/s/steam/steam-launcher_1.0.0.74_all.deb",
				Size:         3585480,
				MD5sum:       "e3a4406e410d2ed472fcfce37833d58f",
				SHA1:         "a46c57ff1e882ecd96ae0a43a344b79c614c48aa",
				SHA256:       "317fb8048ba649fb00a1e33581e72fbade5c1eff83adaf8e4472481dbddffae1",
				Raw: `Installed-Size: 3685
Depends: apt (>= 1.6) | apt-transport-https, ca-certificates, coreutils (>= 8.23-1~) | realpath, curl, file, libc6 (>= 2.15), libnss3 (>= 2:3.26), policykit-1, python3, python3-apt, xterm | gnome-terminal | konsole | x-terminal-emulator, xz-utils, zenity
Recommends: steam-libs-amd64, steam-libs-i386, sudo, xdg-utils | steamos-base-files
Breaks: steam64
Replaces: steam, steam64
Multi-Arch: foreign
Homepage: http://www.steampowered.com/
Priority: optional
Section: games
Description: Launcher for the Steam software distribution service
 Steam is a software distribution service with an online store, automated
 installation, automatic updates, achievements, SteamCloud synchronized
 savegame and screenshot functionality, and many social features.`,
			},
			args: args{
				data: []byte(`Package: steam-launcher
Source: steam
Version: 1:1.0.0.74
Architecture: all
Maintainer: Valve Corporation <linux@steampowered.com>
Installed-Size: 3685
Depends: apt (>= 1.6) | apt-transport-https, ca-certificates, coreutils (>= 8.23-1~) | realpath, curl, file, libc6 (>= 2.15), libnss3 (>= 2:3.26), policykit-1, python3, python3-apt, xterm | gnome-terminal | konsole | x-terminal-emulator, xz-utils, zenity
Recommends: steam-libs-amd64, steam-libs-i386, sudo, xdg-utils | steamos-base-files
Breaks: steam64
Replaces: steam, steam64
Multi-Arch: foreign
Homepage: http://www.steampowered.com/
Priority: optional
Section: games
Filename: pool/steam/s/steam/steam-launcher_1.0.0.74_all.deb
Size: 3585480
SHA256: 317fb8048ba649fb00a1e33581e72fbade5c1eff83adaf8e4472481dbddffae1
SHA1: a46c57ff1e882ecd96ae0a43a344b79c614c48aa
MD5sum: e3a4406e410d2ed472fcfce37833d58f
Description: Launcher for the Steam software distribution service
 Steam is a software distribution service with an online store, automated
 installation, automatic updates, achievements, SteamCloud synchronized
 savegame and screenshot functionality, and many social features.`),
			},
			wantErr: false,
		},
		{
			name: "case 2",
			fields: fields{
				Package:      "uri-one",
				Source:       "uri-one",
				Version:      "0.0.1",
				Architecture: "amd64",
				Maintainer:   "DewepPro <support@dewep.pro>",
				Filename:     "",
				Size:         0,
				MD5sum:       "",
				SHA1:         "",
				SHA256:       "",
				Raw: `Installed-Size: 13287
Depends: systemd, ca-certificates
Section: web
Priority: optional
Homepage: https://www.dewep.pro/products/urione
Description: Link shortening service `,
			},
			args: args{
				data: []byte(`Package: uri-one
Source: uri-one
Version: 0.0.1
Architecture: amd64
Maintainer: DewepPro <support@dewep.pro>
Installed-Size: 13287
Depends: systemd, ca-certificates
Section: web
Priority: optional
Homepage: https://www.dewep.pro/products/urione
Description: Link shortening service `),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &PackegesModel{}
			require.NoError(t, v.Decode(tt.args.data))
			require.Equal(t, tt.fields.Package, v.Package)
			require.Equal(t, tt.fields.Source, v.Source)
			require.Equal(t, tt.fields.Version, v.Version)
			require.Equal(t, tt.fields.Architecture, v.Architecture)
			require.Equal(t, tt.fields.Maintainer, v.Maintainer)
			require.Equal(t, tt.fields.Filename, v.Filename)
			require.Equal(t, tt.fields.Size, v.Size)
			require.Equal(t, tt.fields.MD5sum, v.MD5sum)
			require.Equal(t, tt.fields.SHA1, v.SHA1)
			require.Equal(t, tt.fields.SHA256, v.SHA256)
			require.Equal(t, tt.fields.Raw, v.Raw)
		})
	}
}

func TestModel_Encode(t *testing.T) {
	type fields struct {
		Package      string
		Source       string
		Version      string
		Architecture string
		Maintainer   string
		Filename     string
		Size         int64
		MD5sum       string
		SHA1         string
		SHA256       string
		Raw          string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				Package:      "steam-launcher",
				Source:       "steam",
				Version:      "1:1.0.0.74",
				Architecture: "all",
				Maintainer:   "Valve Corporation <linux@steampowered.com>",
				Filename:     "pool/steam/s/steam/steam-launcher_1.0.0.74_all.deb",
				Size:         3585480,
				MD5sum:       "e3a4406e410d2ed472fcfce37833d58f",
				SHA1:         "a46c57ff1e882ecd96ae0a43a344b79c614c48aa",
				SHA256:       "317fb8048ba649fb00a1e33581e72fbade5c1eff83adaf8e4472481dbddffae1",
				Raw: `Installed-Size: 3685
Depends: apt (>= 1.6) | apt-transport-https, ca-certificates, coreutils (>= 8.23-1~) | realpath, curl, file, libc6 (>= 2.15), libnss3 (>= 2:3.26), policykit-1, python3, python3-apt, xterm | gnome-terminal | konsole | x-terminal-emulator, xz-utils, zenity
Recommends: steam-libs-amd64, steam-libs-i386, sudo, xdg-utils | steamos-base-files
Breaks: steam64
Replaces: steam, steam64
Multi-Arch: foreign
Homepage: http://www.steampowered.com/
Priority: optional
Section: games
Description: Launcher for the Steam software distribution service
 Steam is a software distribution service with an online store, automated
 installation, automatic updates, achievements, SteamCloud synchronized
 savegame and screenshot functionality, and many social features.`,
			},
			want: []byte(`Package: steam-launcher
Source: steam
Version: 1:1.0.0.74
Architecture: all
Maintainer: Valve Corporation <linux@steampowered.com>
Filename: pool/steam/s/steam/steam-launcher_1.0.0.74_all.deb
Size: 3585480
MD5sum: e3a4406e410d2ed472fcfce37833d58f
SHA1: a46c57ff1e882ecd96ae0a43a344b79c614c48aa
SHA256: 317fb8048ba649fb00a1e33581e72fbade5c1eff83adaf8e4472481dbddffae1
Installed-Size: 3685
Depends: apt (>= 1.6) | apt-transport-https, ca-certificates, coreutils (>= 8.23-1~) | realpath, curl, file, libc6 (>= 2.15), libnss3 (>= 2:3.26), policykit-1, python3, python3-apt, xterm | gnome-terminal | konsole | x-terminal-emulator, xz-utils, zenity
Recommends: steam-libs-amd64, steam-libs-i386, sudo, xdg-utils | steamos-base-files
Breaks: steam64
Replaces: steam, steam64
Multi-Arch: foreign
Homepage: http://www.steampowered.com/
Priority: optional
Section: games
Description: Launcher for the Steam software distribution service
 Steam is a software distribution service with an online store, automated
 installation, automatic updates, achievements, SteamCloud synchronized
 savegame and screenshot functionality, and many social features.`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &PackegesModel{
				Package:      tt.fields.Package,
				Source:       tt.fields.Source,
				Version:      tt.fields.Version,
				Architecture: tt.fields.Architecture,
				Maintainer:   tt.fields.Maintainer,
				Filename:     tt.fields.Filename,
				Size:         tt.fields.Size,
				MD5sum:       tt.fields.MD5sum,
				SHA1:         tt.fields.SHA1,
				SHA256:       tt.fields.SHA256,
				Raw:          tt.fields.Raw,
			}
			got, err := v.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() got = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}
