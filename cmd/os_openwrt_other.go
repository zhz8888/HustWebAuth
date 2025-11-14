//go:build darwin || openbsd || freebsd

// Package cmd 提供非Linux平台下OpenWrt系统检测功能
package cmd

// isOpenWrt 检查当前系统是否为OpenWrt
// 在非Linux平台（darwin、openbsd、freebsd）上，OpenWrt系统不存在
// 返回值:
//   - bool: 总是返回false，表示不是OpenWrt系统
func isOpenWrt() (ok bool) {
	// 在非Linux平台上，OpenWrt系统不存在，直接返回false
	return false
}
