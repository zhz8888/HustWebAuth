//go:build linux

// Package cmd 提供Linux平台下OpenWrt系统检测功能
package cmd

import (
	"io"
	"sync"

	"github.com/AdguardTeam/golibs/stringutil"
)

var (
	// isOpenWrtOnce 确保OpenWrt检测只执行一次的同步对象
	isOpenWrtOnce sync.Once
	// isOpenWrtValue 存储OpenWrt检测结果
	isOpenWrtValue bool
	// isOpenWrtChecked 标记是否已完成OpenWrt检测
	isOpenWrtChecked bool
)

// isOpenWrt 检查当前系统是否为OpenWrt
// 使用sync.Once确保检测只执行一次，提高性能
// 返回值:
//   - bool: 如果是OpenWrt系统返回true，否则返回false
func isOpenWrt() (ok bool) {
	// 使用sync.Once确保检测逻辑只执行一次
	isOpenWrtOnce.Do(func() {
		// 定义要搜索的发行版信息文件模式
		const etcReleasePattern = "etc/*release*"

		var err error
		// 使用FileWalker遍历系统中的发行版信息文件
		ok, err = FileWalker(func(r io.Reader) (_ []string, cont bool, err error) {
			// OpenWrt系统的标识字符串
			const osNameData = "openwrt"

			// 读取文件内容，这里使用ReadAll是安全的，因为FileWalker的Walk()方法限制了r的大小
			var data []byte
			data, err = io.ReadAll(r)
			if err != nil {
				return nil, false, err
			}

			// 检查文件内容是否包含"openwrt"字符串（不区分大小写）
			// 如果包含，则停止遍历（cont=false），否则继续遍历（cont=true）
			return nil, !stringutil.ContainsFold(string(data), osNameData), nil
		}).Walk(RootDirFS(), etcReleasePattern)

		// 设置检测结果：没有错误且找到OpenWrt标识
		isOpenWrtValue = (err == nil && ok)
		// 标记检测已完成
		isOpenWrtChecked = true
	})
	
	// 返回检测结果
	return isOpenWrtValue
}
