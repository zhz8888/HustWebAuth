// 文件遍历功能内部测试，基于 AdGuardHome 项目
// https://github.com/AdguardTeam/AdGuardHome/blob/master/internal/aghos/filewalker_internal_test.go

package cmd

import (
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errFS 是一个 fs.FS 实现，其 Open 方法总是返回 errFSOpen
type errFS struct{}

// errFSOpen 是从 errFS.Open 返回的错误
const errFSOpen errors.Error = "test open error"

// Open 为 *errFS 实现 fs.FS 接口
// fsys 始终为 nil，err 始终为 errFSOpen
func (efs *errFS) Open(name string) (fsys fs.File, err error) {
	return nil, errFSOpen
}

// TestWalkerFunc_CheckFile 测试 checkFile 函数的各种情况
func TestWalkerFunc_CheckFile(t *testing.T) {
	emptyFS := fstest.MapFS{}

	t.Run("non-existing", func(t *testing.T) {
		// 测试不存在的文件
		_, ok, err := checkFile(emptyFS, nil, "lol")
		require.NoError(t, err)

		assert.True(t, ok)
	})

	t.Run("invalid_argument", func(t *testing.T) {
		// 测试无效参数
		_, ok, err := checkFile(&errFS{}, nil, "")
		require.ErrorIs(t, err, errFSOpen)

		assert.False(t, ok)
	})

	t.Run("ignore_dirs", func(t *testing.T) {
		// 测试忽略目录
		const dirName = "dir"

		testFS := fstest.MapFS{
			path.Join(dirName, "file"): &fstest.MapFile{Data: []byte{}},
		}

		patterns, ok, err := checkFile(testFS, nil, dirName)
		require.NoError(t, err)

		assert.Empty(t, patterns)
		assert.True(t, ok)
	})
}
