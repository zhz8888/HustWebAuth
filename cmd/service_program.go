// Package cmd 提供系统服务程序实现
package cmd

import (
	"log"

	"github.com/kardianos/service"
)

// Start 实现服务启动接口
// Start方法不应阻塞，实际工作应该在异步goroutine中执行
func (p *program) Start(service.Service) error {
	// Start should not block. Do the actual work async.
	log.Println("Starting HustWebAuth service...")
	go p.run()
	return nil
}

// run 执行服务的主要逻辑
// 在单独的goroutine中运行循环认证逻辑
func (p *program) run() {
	runCycle()
}

// Stop 实现服务停止接口
// 当服务停止时调用此方法
func (p *program) Stop(service.Service) error {
	log.Println("Stoping HustWebAuth service...")
	return nil
}
