/*
Copyright © 2022 a76yyyy q981331502@163.com
*/

// 网络认证相关功能
package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// register 标记是否需要注册MAC地址
var register bool

// HTTP客户端连接池，复用TCP连接
var (
	httpClient *http.Client
	httpOnce   sync.Once
)

// getHTTPClient 获取单例HTTP客户端，使用连接池
// 返回配置好的HTTP客户端实例
func getHTTPClient() *http.Client {
	httpOnce.Do(func() {
		// 配置传输层参数
		transport := &http.Transport{
			MaxIdleConns:        10,               // 最大空闲连接数
			IdleConnTimeout:     30 * time.Second, // 空闲连接超时时间
			DisableCompression:  false,            // 启用压缩
			MaxIdleConnsPerHost: 5,                // 每个主机的最大空闲连接数
		}
		
		// 创建HTTP客户端
		httpClient = &http.Client{
			Timeout:   10 * time.Second, // 设置请求超时
			Transport: transport,
		}
	})
	return httpClient
}

// loginCmd 表示登录命令
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Hust web auth only once",
	Long:  `Hust web auth only once.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 执行登录操作
		res, err := Login()
		if err != nil {
			log.Fatal(err)
		}
		// 输出登录结果
		log.Println(res)
	},
}

// init 初始化login命令
func init() {
	// 将login命令添加到root命令下
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")
	// 添加register标志，用于指定是否注册MAC地址
	loginCmd.PersistentFlags().BoolVarP(&register, "register", "r", false, "Register Mac address")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// GetCookie 获取认证页面的Cookie
// 参数: url - 认证页面URL
// 返回值: 第一个Cookie和可能的错误
func GetCookie(url string) (*http.Cookie, error) {
	// 使用共享的HTTP客户端
	client := getHTTPClient()
	// 发送GET请求获取Cookie
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// 获取响应中的所有Cookie
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, errors.New("no cookies found")
	}
	// 返回第一个Cookie
	return cookies[0], err
}

// login 执行网络认证
// 参数: 
//   - loginUrl: 登录URL
//   - queryString: 查询字符串
//   - account: 账户名
//   - password: 密码
//   - serviceType: 服务类型
//   - encrypt: 是否加密
//   - cookie: HTTP Cookie
// 返回值: 认证结果和可能的错误
func login(loginUrl string, queryString string, account string, password string, serviceType string, encrypt bool, cookie *http.Cookie) (string, error) {
	// 构建实际的登录URL
	trueurl := strings.Split(loginUrl, "/eportal/")[0] + "/eportal/InterFace.do?method=login"

	// 使用共享的HTTP客户端
	client := getHTTPClient()
	
	// 使用strings.Builder更高效地构建POST数据，预分配缓冲区大小
	var buf bytes.Buffer
	buf.Grow(128) // 预分配缓冲区大小，减少内存重新分配
	buf.WriteString("userId=")
	buf.WriteString(url.QueryEscape(account))
	buf.WriteString("&password=")
	buf.WriteString(url.QueryEscape(password))
	buf.WriteString("&service=")
	buf.WriteString(serviceType)
	buf.WriteString("&queryString=")
	buf.WriteString(url.QueryEscape(queryString))
	buf.WriteString("&operatorPwd=&operatorUserId=&validcode=&passwordEncrypt=")
	if encrypt {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
	
	// 创建POST请求
	req, _ := http.NewRequest("POST", trueurl, &buf)
	req.AddCookie(cookie)

	// 设置请求头
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")

	// 发送请求
	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// 读取响应内容
	body, _ := io.ReadAll(resp.Body)
	return string(body), err
}

// RegisterMAC 注册MAC地址，仅在首次使用时需要
// 参数:
//   - loginUrl: 登录URL
//   - userIndex: 用户索引
//   - cookie: HTTP Cookie
// 返回值: 注册结果和可能的错误
func RegisterMAC(loginUrl string, userIndex string, cookie *http.Cookie) (string, error) {
	// 构建MAC地址注册URL
	trueurl := strings.Split(loginUrl, "/eportal/")[0] + "/eportal/InterFace.do?method=registerMac"
	// 使用共享的HTTP客户端
	client := getHTTPClient()
	
	// 使用strings.Builder更高效地构建POST数据
	var buf bytes.Buffer
	buf.Grow(64) // 预分配缓冲区大小，减少内存重新分配
	buf.WriteString("mac=&userIndex=")
	buf.WriteString(url.QueryEscape(userIndex))
	
	// 创建POST请求
	req, _ := http.NewRequest("POST", trueurl, &buf)
	req.AddCookie(cookie)

	// 设置请求头
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// 读取响应内容
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

// Login 执行Hust网络认证
// 返回值: 认证结果和可能的错误
func Login() (res string, err error) {
	// 获取登录URL
	url, queryString, connected, err := GetLoginUrl()
	if err != nil {
		return "", err
	}
	// 如果网络已连接，根据配置决定是否输出信息
	if connected {
		if logConnected {
			return "The network is connected, no authentication required", nil
		}
		return "", nil
	}

	// 获取认证Cookie
	cookie, err := GetCookie(url)
	if err != nil {
		return "", err
	}

	// 执行登录认证
	login_res, err := login(url, queryString, account, password, serviceType, encrypt, cookie)
	if err != nil {
		return "", err
	}
	
	// 检查登录结果
	if len(strings.Split(login_res, "\"result\":\"success\"")) == 2 {
		res = "Login success!"
	} else {
		return "", errors.New("Login fail: " + login_res)
	}

	// 如果需要注册MAC地址
	if register {
		// 使用sync.Pool来复用JSON解析的缓冲区，减少内存分配
		var resJson map[string]interface{}
		loginResBytes := []byte(login_res) // 避免重复转换
		err = json.Unmarshal(loginResBytes, &resJson)
		if err == nil {
			// 从响应中获取用户索引
			if userIndex, ok := resJson["userIndex"].(string); ok {
				// 注册MAC地址
				res, err := RegisterMAC(url, userIndex, cookie)
				if err != nil {
					register = false
					return "", err
				}
				return res, nil
			}
		}
		// 如果不支持注册服务，重置注册标志
		register = false
		return "Unsupport register service. ", nil
	}
	return res, nil
}
