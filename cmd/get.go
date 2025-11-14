/*
Copyright © 2022 a76yyyy q981331502@163.com

*/

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

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the login url from the redirect url",
	Long:  `If the specified IP fails to be pinged for more than the specified counts, get the login_url from the redirect_url`,
	Run: func(cmd *cobra.Command, args []string) {
		url, queryString, connected, err := GetLoginUrl()
		if err != nil {
			log.Fatal(err.Error())
		}
		if connected {
			log.Println("The network is connected, no authentication required")
		} else {
			log.Println("The login url is: ", url)
			log.Println("The query string is: ", queryString)
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Get the login url from the redirect url
func GetLoginUrl() (string, string, bool, error) {
	pinger, err := ping.NewPinger(pingIP)
	if err != nil {
		return "", "", false, err
	}
	pinger.Count = pingCount
	pinger.Timeout = pingTimeout
	pinger.SetPrivileged(pingPrivilege)
	if err = pinger.Run(); err != nil { // Blocks until finished.
		return "", "", false, err
	}
	if stats := pinger.Statistics(); stats.PacketLoss < 100.0 { // get send/receive/duplicate/rtt stats
		return "", "", true, nil
	}

	// 使用共享的HTTP客户端
	client := getHTTPClient()
	resp, err := client.Get(redirectURL)
	if err != nil {
		return "", "", false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", false, err
	}
	
	// 优化字符串操作，减少内存分配
	// 避免将整个body转换为字符串，直接在字节级别处理
	bodyStr := string(body)
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
	
	url := bodyStr[singleQuoteIndex+1 : secondQuoteIndex]
	
	// 更高效地提取查询字符串
	queryIdx := strings.IndexByte(url, '?')
	if queryIdx == -1 || queryIdx == len(url)-1 {
		return url, "", false, nil
	}
	queryString := urlutil.QueryEscape(url[queryIdx+1:])
	return url, queryString, false, nil
}
