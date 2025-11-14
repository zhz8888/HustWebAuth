// Package cmd 提供操作系统相关的命令和功能
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
)

// MaxCmdOutputSize 是执行的shell命令输出的最大长度（字节）
const MaxCmdOutputSize = 64 * 1024

// RunCommand 运行shell命令
// 参数:
//   - command: 要执行的命令
//   - arguments: 命令的参数列表
// 返回值:
//   - int: 命令的退出码
//   - []byte: 命令的输出（限制在MaxCmdOutputSize内）
//   - error: 如果发生错误则返回错误信息
func RunCommand(command string, arguments ...string) (code int, output []byte, err error) {
	// 创建命令执行器
	cmd := exec.Command(command, arguments...)
	// 执行命令并获取输出
	out, err := cmd.Output()

	// 限制输出大小，防止过大
	if len(out) > MaxCmdOutputSize {
		out = out[:MaxCmdOutputSize]
	}

	// 处理命令执行错误
	if err != nil {
		// 如果是退出错误，返回退出码和标准错误输出
		if eerr := new(exec.ExitError); errors.As(err, &eerr) {
			return eerr.ExitCode(), eerr.Stderr, nil
		}

		// 其他类型的错误
		return 1, nil, fmt.Errorf("command %q failed: %w: %s", command, err, out)
	}

	// 成功执行，返回退出码和输出
	return cmd.ProcessState.ExitCode(), out, nil
}

// IsOpenWrt 检查当前主机操作系统是否为OpenWrt
// 返回值:
//   - bool: 如果是OpenWrt系统返回true，否则返回false
func IsOpenWrt() (ok bool) {
	return isOpenWrt()
}

// RootDirFS 返回以操作系统根目录为根的文件系统[fs.FS]
// 在Windows系统上，它返回以系统目录所在卷（通常是C:）为根的文件系统
// 返回值:
//   - fs.FS: 文件系统接口
func RootDirFS() (fsys fs.FS) {
	return rootDirFS()
}
