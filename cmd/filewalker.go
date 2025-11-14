// 文件遍历相关功能，基于 AdGuardHome 项目
// https://github.com/AdguardTeam/AdGuardHome/blob/master/internal/aghos/filewalker.go

package cmd

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/AdguardTeam/golibs/errors"
)

// FileWalker 是用于处理文件树中文件的函数签名
// 与 filepath.Walk 不同，它只遍历匹配提供模式的文件（不包括目录）
// 以及函数返回的文件。所有模式都应对 fs.Glob 有效。
// 如果 FileWalker 返回的 cont 为 false，则停止遍历。
// 建议使用 bufio.Scanner 读取 r，因为输入不受限制。
//
// TODO(e.burkov, a.garipov):  移动到另一个包，如 aghfs。
//
// TODO(e.burkov):  考虑传递文件名或任何附加数据。
type FileWalker func(r io.Reader) (patterns []string, cont bool, err error)

// checkFile 尝试打开并处理位于指定 fsys 中 sourcePath 的单个文件
// 如果路径是目录，则跳过该路径
func checkFile(
	fsys fs.FS,
	c FileWalker,
	sourcePath string,
) (patterns []string, cont bool, err error) {
	var f fs.File
	f, err = fsys.Open(sourcePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// 忽略不存在的文件，因为这可能仅在 filepath.Glob 匹配后文件被删除时发生
			return nil, true, nil
		}

		return nil, false, err
	}
	defer func() { err = errors.WithDeferred(err, f.Close()) }()

	var fi fs.FileInfo
	if fi, err = f.Stat(); err != nil {
		return nil, true, err
	} else if fi.IsDir() {
		// 跳过目录
		return nil, true, nil
	}

	return c(f)
}

// handlePatterns 解析 fsys 中的模式，使用 srcSet 忽略重复项
// srcSet 必须非空
func handlePatterns(
	fsys fs.FS,
	srcSet map[string]bool,
	patterns ...string,
) (sub []string, err error) {
	sub = make([]string, 0, len(patterns))
	for _, p := range patterns {
		var matches []string
		matches, err = fs.Glob(fsys, p)
		if err != nil {
			// 使用模式丰富错误信息，因为 filepath.Glob 不会这样做
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}

		for _, m := range matches {
			if srcSet[m] {
				continue
			}

			srcSet[m] = true
			sub = append(sub, m)
		}
	}

	return sub, nil
}

// Walk 开始遍历 fsys 中由 initial 模式定义的文件
// 如果 fw 签名停止遍历，则只返回 true
func (fw FileWalker) Walk(fsys fs.FS, initial ...string) (ok bool, err error) {
	// sources 切片保持文件遍历的顺序，因为 srcSet.Values() 以未定义顺序返回字符串
	srcSet := make(map[string]bool)
	var src []string
	src, err = handlePatterns(fsys, srcSet, initial...)
	if err != nil {
		return false, err
	}

	var filename string
	defer func() { err = errors.Annotate(err, "checking %q: %w", filename) }()

	for i := 0; i < len(src); i++ {
		var patterns []string
		var cont bool
		filename = src[i]
		patterns, cont, err = checkFile(fsys, fw, src[i])
		if err != nil {
			return false, err
		}

		if !cont {
			return true, nil
		}

		var subsrc []string
		subsrc, err = handlePatterns(fsys, srcSet, patterns...)
		if err != nil {
			return false, err
		}

		src = append(src, subsrc...)
	}

	return false, nil
}
