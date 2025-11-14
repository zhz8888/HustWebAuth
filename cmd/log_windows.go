//go:build windows || plan9

// Package cmd 提供日志初始化功能，适用于Windows和Plan9系统
package cmd

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

// initLog 初始化日志系统
// 根据配置参数设置日志输出到文件或标准错误输出
func initLog() {
	// 默认使用标准错误输出作为日志输出
	logWriter := os.Stderr
	
	// 如果指定了日志文件，则配置文件日志
	if logFile != "" {
		var err error
		// 检查日志目录是否存在，不存在则创建
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			os.Mkdir(logDir, fs.ModeDir)
		}
		
		// 根据配置选择不同的文件打开方式
		if logRandom {
			// 创建临时随机名称的日志文件
			logWriter, err = os.CreateTemp(logDir, logFile)
		} else if logAppend {
			// 以追加模式打开日志文件
			logWriter, err = os.OpenFile(filepath.Join(logDir, logFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			// 以写入模式打开日志文件（覆盖原有内容）
			logWriter, err = os.OpenFile(filepath.Join(logDir, logFile), os.O_CREATE|os.O_WRONLY, 0644)
		}
		
		// 如果打开文件失败，记录错误并退出
		if err != nil {
			log.Fatal("Open log file failed, Err:", err)
		}
		// 输出日志文件路径
		log.Println("Log file:", logWriter.Name())
	}
	
	// 设置日志输出（Windows和Plan9系统不支持系统日志）
	log.SetOutput(logWriter)
}
