package archive_test

import (
	"testing"

	"github.com/dewep-online/deb-builder/pkg/archive"
	"github.com/stretchr/testify/require"
)

func TestDeb(t *testing.T) {
	deb, err := archive.NewDeb("/tmp/test.deb")
	require.NoError(t, err)

	err = deb.WriteData("hello.txt", []byte("bbbbb"))
	require.NoError(t, err)
	err = deb.WriteData("hello2.txt", []byte("bbbbb"))
	require.NoError(t, err)

	// err = os.WriteFile("/tmp/test.txt", []byte("aaaaa"), 0755)
	// require.NoError(t, err)
	// err = deb.WriteFile("/tmp/test.txt")
	// require.NoError(t, err)

	err = deb.Close()
	require.NoError(t, err)
}
