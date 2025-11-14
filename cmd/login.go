/*
Copyright © 2022 a76yyyy q981331502@163.com
*/

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

var register bool

// HTTP客户端连接池，复用TCP连接
var (
	httpClient *http.Client
	httpOnce   sync.Once
)

// getHTTPClient 获取单例HTTP客户端，使用连接池
func getHTTPClient() *http.Client {
	httpOnce.Do(func() {
		transport := &http.Transport{
			MaxIdleConns:        10,               // 最大空闲连接数
			IdleConnTimeout:     30 * time.Second, // 空闲连接超时时间
			DisableCompression:  false,            // 启用压缩
			MaxIdleConnsPerHost: 5,                // 每个主机的最大空闲连接数
		}
		
		httpClient = &http.Client{
			Timeout:   10 * time.Second, // 设置请求超时
			Transport: transport,
		}
	})
	return httpClient
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Hust web auth only once",
	Long:  `Hust web auth only once.`,
	Run: func(cmd *cobra.Command, args []string) {
		res, err := Login()
		if err != nil {
			log.Fatal(err)
		}
		log.Println(res)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")
	loginCmd.PersistentFlags().BoolVarP(&register, "register", "r", false, "Register Mac address")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// Get cookie of the auth page
func GetCookie(url string) (*http.Cookie, error) {
	client := getHTTPClient()
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, errors.New("no cookies found")
	}
	return cookies[0], err
}

// Login to auth the network
func login(loginUrl string, queryString string, account string, password string, serviceType string, encrypt bool, cookie *http.Cookie) (string, error) {
	trueurl := strings.Split(loginUrl, "/eportal/")[0] + "/eportal/InterFace.do?method=login"

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
	
	req, _ := http.NewRequest("POST", trueurl, &buf)
	req.AddCookie(cookie)

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), err
}

// RegisterMAC register the mac address, only for the first time
func RegisterMAC(loginUrl string, userIndex string, cookie *http.Cookie) (string, error) {
	trueurl := strings.Split(loginUrl, "/eportal/")[0] + "/eportal/InterFace.do?method=registerMac"
	client := getHTTPClient()
	
	// 使用strings.Builder更高效地构建POST数据
	var buf bytes.Buffer
	buf.Grow(64) // 预分配缓冲区大小，减少内存重新分配
	buf.WriteString("mac=&userIndex=")
	buf.WriteString(url.QueryEscape(userIndex))
	
	req, _ := http.NewRequest("POST", trueurl, &buf)
	req.AddCookie(cookie)

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

// Hust web auth once.
func Login() (res string, err error) {
	url, queryString, connected, err := GetLoginUrl()
	if err != nil {
		return "", err
	}
	if connected {
		if logConnected {
			return "The network is connected, no authentication required", nil
		}
		return "", nil
	}

	cookie, err := GetCookie(url)
	if err != nil {
		return "", err
	}

	login_res, err := login(url, queryString, account, password, serviceType, encrypt, cookie)
	if err != nil {
		return "", err
	}
	if len(strings.Split(login_res, "\"result\":\"success\"")) == 2 {
		res = "Login success!"
	} else {
		return "", errors.New("Login fail: " + login_res)
	}

	if register {
		// 使用sync.Pool来复用JSON解析的缓冲区，减少内存分配
		var resJson map[string]interface{}
		loginResBytes := []byte(login_res) // 避免重复转换
		err = json.Unmarshal(loginResBytes, &resJson)
		if err == nil {
			if userIndex, ok := resJson["userIndex"].(string); ok {
				res, err := RegisterMAC(url, userIndex, cookie)
				if err != nil {
					register = false
					return "", err
				}
				return res, nil
			}
		}
		register = false
		return "Unsupport register service. ", nil
	}
	return res, nil
}
