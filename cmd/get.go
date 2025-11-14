/*
Copyright © 2022 a76yyyy q981331502@163.com
*/

// 获取登录URL相关功能
package cmd

import (
	"errors"
	"io"
	"log"
	urlutil "net/url"
	"strings"

	ping "github.com/prometheus-community/pro-bing"
	"github.com/spf13/cobra"
)

// getCmd 表示获取命令
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the login url from the redirect url",
	Long:  `If the specified IP fails to be pinged for more than the specified counts, get the login_url from the redirect_url`,
	Run: func(cmd *cobra.Command, args []string) {
		// 获取登录URL、查询字符串和网络连接状态
		url, queryString, connected, err := GetLoginUrl()
		if err != nil {
			// 如果获取失败，记录错误并退出
			log.Fatal(err.Error())
		}
		if connected {
			// 如果网络已连接，无需认证
			log.Println("The network is connected, no authentication required")
		} else {
			// 如果网络未连接，显示登录URL和查询字符串
			log.Println("The login url is: ", url)
			log.Println("The query string is: ", queryString)
		}
	},
}

// init 初始化get命令
func init() {
	// 将get命令添加到root命令下
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// GetLoginUrl 从重定向URL获取登录URL
// 返回值: 登录URL、查询字符串、网络连接状态和可能的错误
func GetLoginUrl() (string, string, bool, error) {
	// 创建ping检测器
	pinger, err := ping.NewPinger(pingIP)
	if err != nil {
		return "", "", false, err
	}
	// 设置ping参数
	pinger.Count = pingCount
	pinger.Timeout = pingTimeout
	pinger.SetPrivileged(pingPrivilege)
	// 执行ping检测
	if err = pinger.Run(); err != nil { // Blocks until finished.
		return "", "", false, err
	}
	// 检查ping统计结果，如果丢包率小于100%，表示网络已连接
	if stats := pinger.Statistics(); stats.PacketLoss < 100.0 { // get send/receive/duplicate/rtt stats
		return "", "", true, nil
	}

	// 使用共享的HTTP客户端
	client := getHTTPClient()
	// 发送GET请求到重定向URL
	resp, err := client.Get(redirectURL)
	if err != nil {
		return "", "", false, err
	}
	defer resp.Body.Close()
	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", false, err
	}
	
	// 优化字符串操作，减少内存分配
	// 避免将整个body转换为字符串，直接在字节级别处理
	bodyStr := string(body)
	// 查找第一个单引号
	singleQuoteIndex := strings.IndexByte(bodyStr, '\'')
	if singleQuoteIndex == -1 {
		return "", "", false, errors.New("invalid response format: no single quote found")
	}
	
	// 查找第二个单引号
	secondQuoteIndex := strings.IndexByte(bodyStr[singleQuoteIndex+1:], '\'')
	if secondQuoteIndex == -1 {
		return "", "", false, errors.New("invalid response format: second quote not found")
	}
	secondQuoteIndex += singleQuoteIndex + 1
	
	// 提取两个单引号之间的URL
	url := bodyStr[singleQuoteIndex+1 : secondQuoteIndex]
	
	// 更高效地提取查询字符串
	queryIdx := strings.IndexByte(url, '?')
	if queryIdx == -1 || queryIdx == len(url)-1 {
		// 如果没有查询字符串或查询字符串为空，返回URL和空查询字符串
		return url, "", false, nil
	}
	// 对查询字符串进行URL编码
	queryString := urlutil.QueryEscape(url[queryIdx+1:])
	return url, queryString, false, nil
}
