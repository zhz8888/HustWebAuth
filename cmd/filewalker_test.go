// 文件遍历功能测试，基于 AdGuardHome 项目
// https://github.com/AdguardTeam/AdGuardHome/blob/master/internal/aghos/filewalker_test.go

package cmd

import (
	"bufio"
	"io"
	"path"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileWalker_Walk 测试 FileWalker 的 Walk 方法
func TestFileWalker_Walk(t *testing.T) {
	const attribute = `000`

	// 创建文件遍历器函数
	makeFileWalker := func(_ string) (fw FileWalker) {
		return func(r io.Reader) (patterns []string, cont bool, err error) {
			s := bufio.NewScanner(r)
			for s.Scan() {
				line := s.Text()
				if line == attribute {
					// 如果遇到特定属性，停止遍历
					return nil, false, nil
				}

				if len(line) != 0 {
					// 将非空行添加到模式列表中
					patterns = append(patterns, path.Join(".", line))
				}
			}

			return patterns, true, s.Err()
		}
	}

	const nl = "\n"

	// 定义测试用例
	testCases := []struct {
		testFS      fstest.MapFS
		want        assert.BoolAssertionFunc
		initPattern string
		name        string
	}{{
		name: "simple",
		testFS: fstest.MapFS{
			"simple_0001.txt": &fstest.MapFile{Data: []byte(attribute + nl)},
		},
		initPattern: "simple_0001.txt",
		want:        assert.True,
	}, {
		name: "chain",
		testFS: fstest.MapFS{
			"chain_0001.txt": &fstest.MapFile{Data: []byte(`chain_0002.txt` + nl)},
			"chain_0002.txt": &fstest.MapFile{Data: []byte(`chain_0003.txt` + nl)},
			"chain_0003.txt": &fstest.MapFile{Data: []byte(attribute + nl)},
		},
		initPattern: "chain_0001.txt",
		want:        assert.True,
	}, {
		name: "several",
		testFS: fstest.MapFS{
			"several_0001.txt": &fstest.MapFile{Data: []byte(`several_*` + nl)},
			"several_0002.txt": &fstest.MapFile{Data: []byte(`several_0001.txt` + nl)},
			"several_0003.txt": &fstest.MapFile{Data: []byte(attribute + nl)},
		},
		initPattern: "several_0001.txt",
		want:        assert.True,
	}, {
		name: "no",
		testFS: fstest.MapFS{
			"no_0001.txt": &fstest.MapFile{Data: []byte(nl)},
			"no_0002.txt": &fstest.MapFile{Data: []byte(nl)},
			"no_0003.txt": &fstest.MapFile{Data: []byte(nl)},
		},
		initPattern: "no_*",
		want:        assert.False,
	}, {
		name: "subdirectory",
		testFS: fstest.MapFS{
			path.Join("dir", "subdir_0002.txt"): &fstest.MapFile{
				Data: []byte(attribute + nl),
			},
			"subdir_0001.txt": &fstest.MapFile{Data: []byte(`dir/*`)},
		},
		initPattern: "subdir_0001.txt",
		want:        assert.True,
	}}

	// 运行所有测试用例
	for _, tc := range testCases {
		fw := makeFileWalker("")

		t.Run(tc.name, func(t *testing.T) {
			ok, err := fw.Walk(tc.testFS, tc.initPattern)
			require.NoError(t, err)

			tc.want(t, ok)
		})
	}

	t.Run("pattern_malformed", func(t *testing.T) {
		// 测试格式错误的模式
		f := fstest.MapFS{}
		ok, err := makeFileWalker("").Walk(f, "[]")
		require.Error(t, err)

		assert.False(t, ok)
		assert.ErrorIs(t, err, path.ErrBadPattern)
	})

	t.Run("bad_filename", func(t *testing.T) {
		// 测试错误的文件名
		const filename = "bad_filename.txt"

		f := fstest.MapFS{
			filename: &fstest.MapFile{Data: []byte("[]")},
		}
		ok, err := FileWalker(func(r io.Reader) (patterns []string, cont bool, err error) {
			s := bufio.NewScanner(r)
			for s.Scan() {
				patterns = append(patterns, s.Text())
			}

			return patterns, true, s.Err()
		}).Walk(f, filename)
		require.Error(t, err)

		assert.False(t, ok)
		assert.ErrorIs(t, err, path.ErrBadPattern)
	})

	t.Run("itself_error", func(t *testing.T) {
		// 测试函数本身返回错误的情况
		const rerr errors.Error = "returned error"

		f := fstest.MapFS{
			"mockfile.txt": &fstest.MapFile{Data: []byte(`mockdata`)},
		}

		ok, err := FileWalker(func(r io.Reader) (patterns []string, ok bool, err error) {
			return nil, true, rerr
		}).Walk(f, "*")
		require.ErrorIs(t, err, rerr)

		assert.False(t, ok)
	})
}
