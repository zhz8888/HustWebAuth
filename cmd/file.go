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
	tmpDirOnce sync.Once
	tmpDirValue string
	
	execPathOnce sync.Once
	execPathValue string
)

// 获取当前执行文件绝对路径
func getCurrentAbPath() string {
	execPath := getCurrentAbPathByExecutable()
	if strings.Contains(execPath, getTmpDir()) {
		return getCurrentAbPathByCaller()
	}
	return execPath
}

// 获取当前执行文件绝对目录
func getCurrentAbDir() string {
	return filepath.Dir(getCurrentAbPath())
}

// 获取系统临时目录，兼容go run
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

// 获取当前执行文件绝对路径
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

// 获取当前执行文件绝对路径（go run）
func getCurrentAbPathByCaller() string {
	var abPath string
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		abPath, _ = filepath.Abs(filename)
	}
	return abPath
}
