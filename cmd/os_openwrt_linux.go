//go:build linux

package cmd

import (
	"io"
	"sync"

	"github.com/AdguardTeam/golibs/stringutil"
)

var (
	isOpenWrtOnce sync.Once
	isOpenWrtValue bool
	isOpenWrtChecked bool
)

func isOpenWrt() (ok bool) {
	isOpenWrtOnce.Do(func() {
		const etcReleasePattern = "etc/*release*"

		var err error
		ok, err = FileWalker(func(r io.Reader) (_ []string, cont bool, err error) {
			const osNameData = "openwrt"

			// This use of ReadAll is now safe, because FileWalker's Walk()
			// have limited r.
			var data []byte
			data, err = io.ReadAll(r)
			if err != nil {
				return nil, false, err
			}

			return nil, !stringutil.ContainsFold(string(data), osNameData), nil
		}).Walk(RootDirFS(), etcReleasePattern)

		isOpenWrtValue = (err == nil && ok)
		isOpenWrtChecked = true
	})
	return isOpenWrtValue
}
