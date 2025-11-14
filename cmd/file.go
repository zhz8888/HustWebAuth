// 文件路径处理相关功能
package cmd

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// 缓存计算结果，避免重复的文件系统操作
var (
	tmpDirOnce  sync.Once
	tmpDirValue string

	execPathOnce  sync.Once
	execPathValue string
)

// getCurrentAbPath 获取当前执行文件的绝对路径
// 优先使用可执行文件路径，如果路径包含临时目录则使用调用者路径
func getCurrentAbPath() string {
	execPath := getCurrentAbPathByExecutable()
	if strings.Contains(execPath, getTmpDir()) {
		return getCurrentAbPathByCaller()
	}
	return execPath
}

// getCurrentAbDir 获取当前执行文件的绝对目录
func getCurrentAbDir() string {
	return filepath.Dir(getCurrentAbPath())
}

// getTmpDir 获取系统临时目录，兼容 go run 模式
// 使用 sync.Once 确保只计算一次，提高性能
func getTmpDir() string {
	tmpDirOnce.Do(func() {
		dir := os.Getenv("TEMP")
		if dir == "" {
			dir = os.Getenv("TMP")
			if dir == "" {
				dir = os.TempDir()
			}
		}
		// 使用sync.Once确保只计算一次
		res, _ := filepath.EvalSymlinks(dir)
		tmpDirValue = res
	})
	return tmpDirValue
}

// getCurrentAbPathByExecutable 通过可执行文件获取当前执行文件的绝对路径
// 使用 sync.Once 确保只计算一次，提高性能
func getCurrentAbPathByExecutable() string {
	execPathOnce.Do(func() {
		exePath, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		res, _ := filepath.EvalSymlinks(exePath)
		execPathValue = res
	})
	return execPathValue
}

// getCurrentAbPathByCaller 通过调用者获取当前执行文件的绝对路径（适用于 go run 模式）
func getCurrentAbPathByCaller() string {
	var abPath string
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		abPath, _ = filepath.Abs(filename)
	}
	return abPath
}
