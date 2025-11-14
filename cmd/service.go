// Package cmd 提供系统服务相关功能实现
package cmd

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
)

// program 系统服务程序结构体
type program struct {
	// cmd  *cobra.Command  // 保留字段，用于存储命令对象
	// args []string       // 保留字段，用于存储命令参数
}

// newSVCConfig 创建新的系统服务配置
func newSVCConfig() *service.Config {
	// 确定是否启用日志输出
	var logOutput = false
	if logFile != "" {
		logOutput = true
	}

	// 创建服务配置
	c := &service.Config{
		Name:        "HustWebAuth",
		DisplayName: "HustWebAuth",
		Description: "A service used to implement Ruijie web authentication.",
		Arguments:   []string{"service"},
		EnvVars:     map[string]string{"HOME": homeDir},
		Option:      service.KeyValue{"LogOutput": logOutput, "LogDirectory": logDir},
	}

	// Linux/systemd系统上，仅在网络就绪后启动服务
	if sysType == "linux" {
		c.Dependencies = []string{
			"After=syslog.target network.target",
		}
	}

	// 在OpenWrt和FreeBSD上使用不同的脚本
	if IsOpenWrt() {
		c.Option["SysvScript"] = openWrtScript
	}

	return c
}

// newSVC 创建新的系统服务实例
func newSVC(prg *program, conf *service.Config) (service.Service, error) {
	s, err := service.New(prg, conf)
	if err != nil {
		// log.Fatal(err)
		return nil, err
	}
	return s, nil
}

// 服务相关命令定义
var (
	// serviceCmd 表示服务命令
	serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "System service related commands",
		Long:  `Use HustWebAuth as a system service: install, start, stop, uninstall, etc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSVC(&program{}, newSVCConfig())
			if err != nil {
				return err
			}
			return s.Run()
		},
	}

	// installCmd 安装服务命令
	installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install HustWebAuth service",
		Run: func(cmd *cobra.Command, args []string) {
			// 创建服务配置
			svcConfig := newSVCConfig()

			// 创建服务实例
			s, err := newSVC(&program{}, svcConfig)
			if err != nil {
				log.Fatal(err)
				return
			}

			// 安装服务
			err = svcAction(s, "install")
			if err != nil {
				log.Fatal(err)
				return
			}
			
			// 在OpenWrt上，安装后必须运行enable命令，否则服务不会在系统启动时自动启动
			if IsOpenWrt() {
				_, err = runInitdCommand(s.String(), "enable")
				if err != nil {
					log.Fatalf("service: running init enable: %s", err)
				}
			}
			log.Println("HustWebAuth service has been installed")

			// 启动服务
			err = svcAction(s, "start")
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Println("HustWebAuth service started.")
			saveCfg = true
		},
	}

	// startCmd 启动服务命令
	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start HustWebAuth service",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := newSVC(&program{}, newSVCConfig())
			if err != nil {
				log.Fatal(err)
				return
			}

			err = svcAction(s, "start")
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Println("HustWebAuth service started.")
		},
	}

	// statusCmd 查询服务状态命令
	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Get HustWebAuth service status",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := newSVC(&program{}, newSVCConfig())
			if err != nil {
				log.Fatal(err)
				return
			}

			// 获取服务状态
			status, err := svcStatus(s)
			if err != nil {
				log.Fatal(err)
				return
			}
			
			// 根据状态输出相应信息
			switch status {
			case service.StatusUnknown:
				log.Println("HustWebAuth service status is unable to be determined due to an error or it was not installed.")
			case service.StatusStopped:
				log.Println("HustWebAuth service is stopped.")
			case service.StatusRunning:
				log.Println("HustWebAuth service is running.")
			}
		},
	}

	// stopCmd 停止服务命令
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop HustWebAuth service",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := newSVC(&program{}, newSVCConfig())
			if err != nil {
				log.Fatal(err)
				return
			}
			err = svcAction(s, "stop")
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Println("HustWebAuth service stoped.")
		},
	}

	// restartCmd 重启服务命令
	restartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart HustWebAuth service",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := newSVC(&program{}, newSVCConfig())
			if err != nil {
				log.Fatal(err)
				return
			}
			err = svcAction(s, "restart")
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Println("HustWebAuth service has been restarted.")
		},
	}

	// uninstallCmd 卸载服务命令
	uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall HustWebAuth service from system",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := newSVC(&program{}, newSVCConfig())
			if err != nil {
				log.Fatal(err)
				return
			}

			// 在OpenWrt上，首先运行disable命令，因为它会删除符号链接
			if IsOpenWrt() {
				_, err := runInitdCommand(s.String(), "disable")
				if err != nil {
					log.Fatalf("service: running init disable: %s", err)
				}
			}

			// 获取服务状态
			status, err := svcStatus(s)
			if err != nil {
				log.Fatal(err)
				return
			}
			
			// 如果服务正在运行，先停止它
			if status == service.StatusRunning {
				err = svcAction(s, "stop")
				if err != nil {
					log.Println(err)
				}
			}

			// 卸载服务
			err = svcAction(s, "uninstall")
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Println("HustWebAuth service has been uninstalled")
		},
	}
)

// init 初始化函数，将服务相关命令添加到根命令
func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(installCmd, startCmd, statusCmd, stopCmd, restartCmd, uninstallCmd)
}

// runInitdCommand 运行init.d服务命令
// 返回命令代码或错误信息
func runInitdCommand(serviceName, action string) (int, error) {
	confPath := "/etc/init.d/" + serviceName
	// 将脚本和操作作为单个字符串参数传递
	code, _, err := RunCommand("sh", "-c", confPath+" "+action)

	return code, err
}

// svcAction 执行服务操作
//
// 在OpenWrt上，service工具可能不存在，我们直接使用服务脚本
func svcAction(s service.Service, action string) (err error) {
	// macOS系统上启动服务时进行特殊检查
	if sysType == "darwin" && action == "start" {
		var exe string
		if exe, err = os.Executable(); err != nil {
			log.Println("Starting service error: getting executable path: ", err)
		} else if exe, err = filepath.EvalSymlinks(exe); err != nil {
			log.Println("Starting service error: evaluating executable symlinks: ", err)
		} else if !strings.HasPrefix(exe, "/Applications/") {
			log.Println("warning: service must be started from within the /Applications directory")
		}
	}

	// 尝试执行服务操作
	err = service.Control(s, action)
	// 如果是unix-systemv平台且操作失败，尝试直接使用init.d脚本
	if err != nil && service.Platform() == "unix-systemv" &&
		(action == "start" || action == "stop" || action == "restart") {
		_, err = runInitdCommand(s.String(), action)

		return err
	}

	return err
}

// svcStatus 返回服务状态
//
// 在OpenWrt上，service工具可能不存在，我们直接使用服务脚本
func svcStatus(s service.Service) (status service.Status, err error) {
	// 尝试获取服务状态
	status, err = s.Status()
	// 如果是unix-systemv平台且获取失败，尝试直接使用init.d脚本
	if err != nil && service.Platform() == "unix-systemv" {
		var code int
		code, err = runInitdCommand(s.String(), "status")
		if err != nil || code != 0 {
			return service.StatusStopped, nil
		}

		return service.StatusRunning, nil
	}

	return status, err
}

// openWrtScript OpenWrt procd初始化脚本
// 参考: https://github.com/AdguardTeam/AdGuardHome/issues/1386
const openWrtScript = `#!/bin/sh /etc/rc.common

START=90
STOP=01

cmd="{{.Path}}{{range .Arguments}} {{.|cmd}}{{end}}"
name="{{.Name}}"
pid_file="/var/run/${name}.pid"
stdout_log="{{.LogDirectory}}/$name.log"
stderr_log="{{.LogDirectory}}/$name.err"

{{range $k, $v := .EnvVars -}}
export {{$k}}={{$v}}
{{end -}}

EXTRA_COMMANDS="status"
EXTRA_HELP="$(printf "\t%-16s%s\n" "status" "Print the service status")"

get_pid() {
    cat "${pid_file}"
}

is_running() {
    [ -f "${pid_file}" ] && ps | grep -v grep | grep $(get_pid) >/dev/null 2>&1
}

start() {
    if is_running; then
        echo "Already started"
    else
        echo "Starting $name"
        {{if .WorkingDirectory}}cd '{{.WorkingDirectory}}'{{end}}
        mkdir -p {{.LogDirectory}}
        $cmd >> "$stdout_log" 2>> "$stderr_log" &
        echo $! > "$pid_file"
        if ! is_running; then
            echo "Unable to start, see $stdout_log and $stderr_log"
            exit 1
        fi
    fi
}

stop() {
    if is_running; then
        echo -n "Stopping $name.."
        kill $(get_pid)
        for i in $(seq 1 10)
        do
            if ! is_running; then
                break
            fi
            echo -n "."
            sleep 1
        done
        echo
        if is_running; then
            echo "Not stopped; may still be shutting down or shutdown may have failed"
            exit 1
        else
            echo "Stopped"
            if [ -f "$pid_file" ]; then
                rm "$pid_file"
            fi
        fi
    else
        echo "Not running"
    fi
}

restart() {
    stop
    if is_running; then
        echo "Unable to stop, will not attempt to start"
        exit 1
    fi
    start
}

status() {
    if is_running; then
        echo "Running"
    else
        echo "Stopped"
        exit 1
    fi
}

`
