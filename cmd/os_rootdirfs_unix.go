//go:build darwin || freebsd || linux || openbsd

// Package cmd 提供Unix-like系统下的根目录文件系统功能
package cmd

import (
	"io/fs"
	"os"
)

// rootDirFS 返回以Unix-like系统根目录为根的文件系统[fs.FS]
// 适用于darwin、freebsd、linux和openbsd系统
// 返回值:
//   - fs.FS: 以根目录"/"为起点的文件系统接口
func rootDirFS() (fsys fs.FS) {
	// 使用os.DirFS创建以根目录"/"为起点的文件系统
	return os.DirFS("/")
}
