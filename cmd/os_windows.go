//go:build windows

// Package cmd 提供Windows系统下的文件系统功能
package cmd

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// rootDirFS 返回以Windows系统目录所在卷为根的文件系统[fs.FS]
// 通常情况下，Windows系统目录位于C:卷，但为了更精确，会先获取系统目录
// 返回值:
//   - fs.FS: 以系统目录所在卷为起点的文件系统接口
func rootDirFS() (fsys fs.FS) {
	// TODO(a.garipov): 如果golang/go#44279问题被解决，应该使用更好的方法
	// 获取Windows系统目录路径
	sysDir, err := windows.GetSystemDirectory()
	if err != nil {
		// 获取系统目录失败，记录错误并使用C:作为默认值
		log.Printf("Error: Getting root filesystem: %s; using C:\n", err)
		// 假设C:是安全的默认值
		return os.DirFS("C:")
	}

	// 返回系统目录所在卷的文件系统
	return os.DirFS(filepath.VolumeName(sysDir))
}

// isOpenWrt 检查当前系统是否为OpenWrt
// 在Windows平台上，OpenWrt系统不存在
// 返回值:
//   - bool: 总是返回false，表示不是OpenWrt系统
func isOpenWrt() (ok bool) {
	// 在Windows平台上，OpenWrt系统不存在，直接返回false
	return false
}
