/*
Copyright © 2022 a76yyyy q981331502@163.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package cmd 提供锐捷网络认证的核心功能实现
package cmd

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	daemon "github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// 配置相关变量
	cfgFile       string   // 配置文件路径
	account       string   // 认证账号
	password      string   // 认证密码
	serviceType   string   // 服务类型
	encrypt       bool     // 密码是否加密
	saveCfg       bool     // 是否保存配置
	
	// User-Agent相关变量
	userAgent     string   // 自定义User-Agent
	
	// Ping相关变量
	pingIP        string   // Ping的目标IP地址
	pingCount     int      // Ping次数
	pingTimeout   time.Duration // Ping超时时间
	pingPrivilege bool     // 是否使用特权Ping
	
	// 重定向和日志相关变量
	redirectURL   string   // 重定向URL
	logDir        string   // 日志目录
	logFile       string   // 日志文件名
	logRandom     bool     // 日志文件名是否包含随机字符串
	logAppend     bool     // 日志文件是否追加模式
	logConnected  bool     // 是否记录网络连接日志
	sysLog        bool     // 是否启用系统日志
	
	// 守护进程相关变量
	daemonEnable  bool     // 是否启用守护进程模式
	daemonPidFile string   // 守护进程PID文件路径
	
	// 循环模式相关变量
	cycleEnable   bool     // 是否启用循环模式
	cycleDuration time.Duration // 循环间隔时间
	cycleRetry    int      // 循环重试次数
)

// 全局变量
var execPath = getCurrentAbPath()               // 当前可执行文件绝对路径
var _, filenameWithSuffix = filepath.Split(execPath) // 可执行文件名（含后缀）
var sysType = runtime.GOOS                      // 操作系统类型
var homeDir string                              // 用户主目录
var homeError error                             // 获取用户主目录时的错误

// rootCmd 表示不带任何子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   filenameWithSuffix,
	Short: "A program used to implement Ruijie web authentication",
	Long:  `HustWebAuth is a program used to implement Ruijie web authentication.`,
	// 如果您的应用程序有相关操作，请取消下面这行的注释
	Run: func(cmd *cobra.Command, args []string) {
		runDaemon()
	},
}

// runDaemon 处理守护进程模式和循环模式的启动逻辑
func runDaemon() {
	// 如果只是保存配置，直接返回
	if saveCfg {
		return
	}
	
	// 非Windows系统且启用守护进程模式
	if sysType != "windows" && daemonEnable {
		// 如果未指定日志文件，使用默认路径
		if logFile == "" {
			tmpDir := filepath.Join(getTmpDir(), "HustWebAuth")
			if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
				os.Mkdir(tmpDir, fs.ModeDir)
			}
			logFile = filepath.Join(tmpDir, filenameWithSuffix+".log")
		}
		
		// 如果未指定PID文件，使用默认路径
		if daemonPidFile == "" {
			daemonPidFile = "/var/run/" + filenameWithSuffix + "_daemon.pid"
		}
		
		// 创建守护进程上下文
		cntxt := &daemon.Context{
			PidFileName: daemonPidFile,
			PidFilePerm: 0644,
			LogFileName: logFile,
			LogFilePerm: 0644,
		}

		// Reborn()返回 子进程为nil 父进程不为nil
		child, err := cntxt.Reborn()
		if err != nil {
			log.Fatal("Unable to run: ", err)
		}
		if child != nil {
			return
		}
		defer func() {
			cntxt.Release()
			log.Println("HustWebAuth Daemon stopped.")
		}()

		log.Println("- - - - - - - - - - - - - - - - - - -")
		log.Println("HustWebAuth Daemon started.")
	}

	// 运行循环模式或单次认证
	runCycle()
}

// runCycle 处理循环认证逻辑
func runCycle() {
	log.Println("- - - - - - - - - - - - - - - - - - -")
	log.Println("HustWebAuth started.")
	retryCount := 0
	
	// 执行首次登录
	res, err := Login()
	if err != nil {
		if cycleEnable {
			if cycleRetry < 0 {
				log.Println("Login failed, Err: ", err)
				log.Println("Login retrying...")
			} else if retryCount < cycleRetry {
				retryCount++
				log.Println("Login failed, Err: ", err)
				log.Println("Login retry ", strconv.Itoa(retryCount), "times after "+cycleDuration.String())
			} else {
				log.Fatal("Login failed, Err: ", err)
			}
		} else {
			log.Fatal("Login failed, Err: ", err)
		}
	}
	if res != "" {
		log.Println(res)
	}

	// 如果启用循环模式，设置定时器
	if cycleEnable {
		eventsTick := time.NewTicker(cycleDuration)
		defer eventsTick.Stop()
		
		// 使用通道来控制并发，避免资源竞争
		loginChan := make(chan struct{}, 1) // 缓冲通道，防止阻塞
		resultChan := make(chan loginResult, 1)
		
		// 初始触发一次登录
		loginChan <- struct{}{}
		
		// 启动一个goroutine处理登录请求
		go func() {
			for range loginChan {
				res, err := Login()
				resultChan <- loginResult{res: res, err: err}
			}
		}()
		
		// 主循环，处理定时器和登录结果
		for {
			select {
			case <-eventsTick.C:
				// 定时触发登录请求
				select {
				case loginChan <- struct{}{}:
					// 成功发送登录请求
				default:
					// 上一次登录还在处理中，跳过这次
					log.Println("Previous login still in progress, skipping this cycle")
				}
				
			case result := <-resultChan:
				// 处理登录结果
				if result.err != nil {
					if cycleRetry < 0 {
						log.Println("Login failed, Err: ", result.err)
						log.Println("Login retrying...")
					} else if retryCount < cycleRetry {
						retryCount++
						log.Println("Login failed, Err: ", result.err)
						log.Println("Login retry", strconv.Itoa(retryCount), "times after", cycleDuration.String())
					} else {
						log.Println("Login failed, Err: ", result.err)
						log.Fatal("Exceed the maximum number of retries, daemon stopped!")
					}
				} else {
					if result.res != "" {
						log.Println(result.res)
					}
					retryCount = 0
				}
			}
		}
	}
}

// loginResult 登录结果结构体，用于在goroutine之间传递结果
type loginResult struct {
	res string // 登录响应消息
	err error  // 错误信息
}

// Execute 将所有子命令添加到根命令并适当设置标志
// 由main.main()调用，只需对rootCmd执行一次
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// 初始化函数，按顺序执行
	cobra.OnInitialize(initHomeDir) // 初始化用户主目录
	cobra.OnInitialize(initConfig)  // 初始化配置
	cobra.OnInitialize(initLog)     // 初始化日志
	cobra.OnFinalize(saveConfig)    // 保存配置

	// 在这里定义标志和配置设置
	// Cobra支持持久标志，如果在此处定义，将对整个应用程序全局有效

	// 基本认证配置
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "f", "", "配置文件路径 (默认是 $HOME/HustWebAuth.yaml)")
	rootCmd.PersistentFlags().StringVarP(&account, "account", "a", "", "锐捷网络认证账号")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "锐捷网络认证密码")
	rootCmd.PersistentFlags().StringVarP(&serviceType, "serviceType", "s", "internet", "服务类型，选项: [internet, local]")
	rootCmd.PersistentFlags().BoolVarP(&encrypt, "encrypt", "e", false, "密码是否加密 (默认 false)")
	
	// User-Agent配置
	rootCmd.PersistentFlags().StringVar(&userAgent, "userAgent", "", "自定义User-Agent字符串 (默认使用内置值)")
	
	// Ping配置
	rootCmd.PersistentFlags().StringVar(&pingIP, "pingIP", "202.114.0.131", "Ping的目标IP地址")
	rootCmd.PersistentFlags().IntVar(&pingCount, "pingCount", 3, "Ping次数")
	rootCmd.PersistentFlags().DurationVar(&pingTimeout, "pingTimeout", 3*time.Second, "Ping超时时间")
	rootCmd.PersistentFlags().BoolVar(&pingPrivilege, "pingPrivilege", true, `设置ping发送的类型。
false表示发送"非特权"UDP ping。
true表示发送"特权"原始ICMP ping。
注意：设置为true需要超级用户权限。
`)
	
	// 重定向和日志配置
	rootCmd.PersistentFlags().StringVar(&redirectURL, "redirectURL", "http://123.123.123.123", "重定向URL")
	rootCmd.PersistentFlags().StringVar(&logDir, "logDir", filepath.Join(os.TempDir(), "HustWebAuth"), "日志目录")
	rootCmd.PersistentFlags().StringVarP(&logFile, "logFile", "l", "", "日志文件名 (默认表示输出到os.stdout)")
	rootCmd.PersistentFlags().BoolVar(&logRandom, "logRandom", true, "日志文件名是否包含随机字符串。\n注意: 如果logFile包含\"*\"，随机字符串将替换最后一个\"*\"。\n")
	rootCmd.PersistentFlags().BoolVar(&logAppend, "logAppend", true, "日志文件是否追加模式。\n注意: 如果logRandom为true，此设置将被忽略")
	rootCmd.PersistentFlags().BoolVar(&logConnected, "logConnected", true, "是否记录\"网络已连接\"的日志")
	rootCmd.PersistentFlags().BoolVar(&sysLog, "syslog", false, "启用系统日志，不支持Windows")
	
	// 其他配置
	rootCmd.PersistentFlags().BoolVarP(&saveCfg, "save", "o", false, "保存配置文件")
	
	// 守护进程配置
	rootCmd.Flags().BoolVarP(&daemonEnable, "daemon", "d", false, "启用守护进程模式，不支持Windows")
	rootCmd.Flags().StringVar(&daemonPidFile, "daemonPidFile", "", "守护进程PID文件")
	
	// 循环模式配置
	rootCmd.Flags().BoolVarP(&cycleEnable, "cycle", "c", false, "启用循环模式")
	rootCmd.Flags().DurationVar(&cycleDuration, "cycleDuration", 5*time.Minute, "循环间隔时间")
	rootCmd.Flags().IntVar(&cycleRetry, "cycleRetry", 3, "循环重试次数，-1表示无限重试")

	// 标记必需的标志
	rootCmd.MarkFlagRequired("account")
	rootCmd.MarkFlagRequired("password")

	// 将命令行标志绑定到Viper配置
	viper.BindPFlag("auth.account", rootCmd.PersistentFlags().Lookup("account"))
	viper.BindPFlag("auth.password", rootCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("auth.serviceType", rootCmd.PersistentFlags().Lookup("serviceType"))
	viper.BindPFlag("auth.encrypt", rootCmd.PersistentFlags().Lookup("encrypt"))
	viper.BindPFlag("auth.userAgent", rootCmd.PersistentFlags().Lookup("userAgent"))
	viper.BindPFlag("ping.ip", rootCmd.PersistentFlags().Lookup("pingIP"))
	viper.BindPFlag("ping.count", rootCmd.PersistentFlags().Lookup("pingCount"))
	viper.BindPFlag("ping.timeout", rootCmd.PersistentFlags().Lookup("pingTimeout"))
	viper.BindPFlag("ping.privilege", rootCmd.PersistentFlags().Lookup("pingPrivilege"))
	viper.BindPFlag("redirect.url", rootCmd.PersistentFlags().Lookup("redirectURL"))
	viper.BindPFlag("log.dir", rootCmd.PersistentFlags().Lookup("logDir"))
	viper.BindPFlag("log.file", rootCmd.PersistentFlags().Lookup("logFile"))
	viper.BindPFlag("log.random", rootCmd.PersistentFlags().Lookup("logRandom"))
	viper.BindPFlag("log.append", rootCmd.PersistentFlags().Lookup("logAppend"))
	viper.BindPFlag("log.connected", rootCmd.PersistentFlags().Lookup("logConnected"))
	viper.BindPFlag("log.syslog", rootCmd.PersistentFlags().Lookup("sysLog"))
	viper.BindPFlag("daemon.enable", rootCmd.Flags().Lookup("daemon"))
	viper.BindPFlag("daemon.pidFile", rootCmd.Flags().Lookup("daemonPidFile"))
	viper.BindPFlag("cycle.enable", rootCmd.Flags().Lookup("cycle"))
	viper.BindPFlag("cycle.duration", rootCmd.Flags().Lookup("cycleDuration"))
	viper.BindPFlag("cycle.retry", rootCmd.Flags().Lookup("cycleRetry"))

	// 隐藏默认的完成命令
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Cobra还支持本地标志，这些标志仅在直接调用此操作时运行
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initHomeDir 初始化用户主目录
func initHomeDir() {
	homeDir, homeError = os.UserHomeDir()
	if homeError != nil {
		log.Println("UserHomeDir Error:", homeError)
		homeDir = getCurrentAbDir()
		log.Println("Using Exec Dir as HOME: ", homeDir)
	}
}

// initConfig 读取配置文件和环境变量（如果设置）
func initConfig() {
	if cfgFile != "" {
		// 使用标志指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 在用户主目录中搜索名为"HustWebAuth"的配置文件（不带扩展名）
		viper.AddConfigPath(homeDir)
		cfgFile = filepath.Join(homeDir, "HustWebAuth.yaml")

		viper.SetConfigType("yaml")
		viper.SetConfigName("HustWebAuth")
	}

	viper.AutomaticEnv() // 读取匹配的环境变量

	// 如果找到配置文件，则读取它
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file: " + viper.ConfigFileUsed())
		// 从配置文件中读取各项配置
		account = viper.GetString("auth.account")
		password = viper.GetString("auth.password")
		serviceType = viper.GetString("auth.serviceType")
		encrypt = viper.GetBool("auth.encrypt")
		userAgent = viper.GetString("auth.userAgent")
		pingIP = viper.GetString("ping.ip")
		pingCount = viper.GetInt("ping.count")
		pingTimeout = viper.GetDuration("ping.timeout")
		pingPrivilege = viper.GetBool("ping.privilege")
		redirectURL = viper.GetString("redirect.url")
		logDir = viper.GetString("log.dir")
		logFile = viper.GetString("log.file")
		logRandom = viper.GetBool("log.random")
		logAppend = viper.GetBool("log.append")
		logConnected = viper.GetBool("log.connected")
		sysLog = viper.GetBool("log.syslog")
		daemonEnable = viper.GetBool("daemon.enable")
		daemonPidFile = viper.GetString("daemon.pidFile")
		cycleEnable = viper.GetBool("cycle.enable")
		cycleDuration = viper.GetDuration("cycle.duration")
		cycleRetry = viper.GetInt("cycle.retry")
	}
}

// GetUserAgent 获取User-Agent字符串
// 优先使用配置文件或命令行参数中定义的User-Agent
// 如果未定义，则使用默认值
func GetUserAgent() string {
	if userAgent != "" {
		// 验证User-Agent是否为有效字符串
		if len(strings.TrimSpace(userAgent)) == 0 {
			log.Println("警告: User-Agent配置为空，将使用默认值")
		} else {
			return userAgent
		}
	}
	// 默认User-Agent
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36"
}

// saveConfig 保存配置到文件
func saveConfig() {
	if saveCfg {
		err := viper.WriteConfigAs(cfgFile)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Save config file: " + cfgFile)
	}
}
