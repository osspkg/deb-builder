/*
 *  Copyright (c) 2021-2025 Mikhail Knyazhev <markus621@gmail.com>. All rights reserved.
 *  Use of this source code is governed by a BSD-3-Clause license that can be found in the LICENSE file.
 */

package commands

import (
	"bytes"
	"fmt"

	"github.com/osspkg/pkg-build/pkg/pgp"
)

func signDetachedRelease(releaseData []byte, signer pgp.Signer) ([]byte, error) {
	var output bytes.Buffer
	if err := signer.SignDetached(bytes.NewReader(releaseData), &output); err != nil {
		return nil, fmt.Errorf("create detached signature: %w", err)
	}
	if output.Len() == 0 {
		return nil, fmt.Errorf("detached signature is empty")
	}
	return output.Bytes(), nil
}
