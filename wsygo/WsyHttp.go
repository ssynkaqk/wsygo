package Wsy

import (
	"fmt"
	"net"
	"time"
	"regexp"
	"net/http"
	"net/url"
	"strings"
	"context"
	"crypto/tls"
	"github.com/go-resty/resty/v2"
)

type WsyHttp struct {
	client         *resty.Client
	Retry          int
	Header         map[string]string
	Timeout        int
	DNSTimeout     int
	ConnectTimeout int
}

func (h *WsyHttp) Errorf(format string, v ...interface{}) {
	Logs("RESTY",format,v...)
}
func (h *WsyHttp) Warnf(format string, v ...interface{}) {
	Logs("RESTY",format,v...)
}
func (h *WsyHttp) Debugf(format string, v ...interface{}) {
	Logs("RESTY",format,v...)
}

func (h *WsyHttp) SetInit() { //初始化配置
	h.Retry          = Str.IIFS(h.Retry != 0, h.Retry, int(0)).(int)           //重试次数 默认不重试
	h.Timeout        = Str.IIFS(h.Timeout != 0, h.Timeout, int(30)).(int)      //请求超时时间
	Dns.Timeout      = Str.IIFS(h.DNSTimeout != 0, h.DNSTimeout, int(2)).(int)    //DNS超时时间
	h.ConnectTimeout = Str.IIFS(h.ConnectTimeout != 0, h.ConnectTimeout, int(10)).(int)  //连接超时（秒）
}
// Get 发送GET请求
func (h *WsyHttp) Get(WebUrl string, data ...interface{}) (string, error) {
	return h.Web("GET", WebUrl, data...)
}
// Post 发送POST请求
func (h *WsyHttp) Post(WebUrl string, data ...interface{}) (string, error) {
	return h.Web("POST", WebUrl, data...)
}
// Put 发送PUT请求
func (h *WsyHttp) Put(WebUrl string, data ...interface{}) (string, error) {
	return h.Web("PUT", WebUrl, data...)
}
// Down 下载文件
func (h *WsyHttp) Down(isUrl string, isPath string, isDel ...bool) (string, error) {
	return Down.New(isUrl, isPath, isDel...)
}
// GetIPV6 发送GET请求
func (h *WsyHttp) Gets(WebUrl string, data ...interface{}) (string, error) {
	return h.WebIPV6("GET", WebUrl, data...)
}

// PostIPV6 发送POST请求
func (h *WsyHttp) Posts(WebUrl string, data ...interface{}) (string, error) {
	return h.WebIPV6("POST", WebUrl, data...)
}
// initClient 初始化HTTP客户端
func (h *WsyHttp) initClient() {
	h.SetInit()
	if h.client == nil {
		h.client = resty.New()
		h.client.SetDebug(false)
		h.client.SetLogger(&WsyHttp{})
		// 基础配置
		h.client.SetRetryCount(h.Retry)
		h.client.SetTimeout(time.Duration(h.Timeout) * time.Second)
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		h.client.SetTLSClientConfig(tlsConfig)
		// 简化的Transport配置，使用自定义DNS
		h.client.SetTransport(&http.Transport{
			TLSClientConfig:       tlsConfig, // 在Transport中也设置TLS配置，跳过证书验证
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 解析地址
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				
				// 如果是域名，使用自定义DNS解析
				if net.ParseIP(host) == nil {
					ip, err := Dns.New(host)
					if err != nil {
						return nil, err
					}
					host = ip
				}
				// 使用解析后的IP连接
				targetAddr := net.JoinHostPort(host, port)
				dialer := &net.Dialer{
					Timeout:   time.Duration(h.ConnectTimeout) * time.Second,
					KeepAlive: 30 * time.Second,
				}
				return dialer.DialContext(ctx, network, targetAddr)
			},
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ForceAttemptHTTP2:     false, // 禁用HTTP/2
			DisableKeepAlives:     false, // 启用KeepAlive
		})
	}
}



// Web 通用HTTP请求方法
//
// 用法示例：
//  // 1. 发送GET请求
//  resp, err := http.Get("https://api.example.com/user?id=1", nil)
//
//  // 2. 发送POST请求体，发送JSON
//  resp, err := http.Post("https://api.example.com/login", map[string]interface{}{"username": "admin", "password": "123456"})
//
//  // 3. 发送POST请求，发送表单
//  resp, err := http.Post("https://api.example.com/login", "username=admin&password=123456")
//
//  // 4. 发送PUT请求
//  resp, err := http.Put("https://api.example.com/user/1", map[string]interface{}{"name": "newname"})
//
//  // 5. 发送GET请求，带查询参数,自定义请求头
//  Wsy.Http.Header = map[string]string{"Authorization": "Bearer token123"}
//  resp, err := http.Get("https://api.example.com/user", "id=1")
//
//	// 6. 使用 JSON 字符串传参
//	jsonParams := `{"page":"1","limit":"10"}`
//	resp, err := http.Get("https://api.example.com/users", jsonParams)
//
// 注意：自定义请求头请使用 Wsy.Http.Header 设置
func (h *WsyHttp) Web(method, WebUrl string, data ...interface{}) (string, error) {

	var d interface{}
	var resp *resty.Response
	var err error
	// 初始化客户端
	h.initClient()
	// 解析参数
	if len(data) > 0 { 
		d = data[0] 
	}

	req := h.client.R() // 设置默认请求头（包含Wsy.Http.Header）
	
	h.SetHeaders(req)
	// 处理请求体
	if method == "GET" {
		WebUrl = h.SetGET(req, WebUrl, d)
	} else {
		h.SetPOST(req, d)
	}
	
	// 记录请求开始时间
	start := time.Now()
	// 发送请求
	switch strings.ToUpper(method) {
	case "POST":
		resp, err = req.Post(WebUrl)
	case "PUT":
		resp, err = req.Put(WebUrl)
	case "GET":
		resp, err = req.Get(WebUrl)
	default:
		return "", fmt.Errorf("不支持的HTTP方法: %s", method)
	}
	duration := time.Since(start)
	// 处理响应
	if err != nil {
		Logs("ERROR", "HTTP", "%s 请求失败，URL: %s，耗时: %dms，错误: %v", method, WebUrl, duration.Milliseconds(), err)
		return "", err
	}
	// 检查状态码
	if resp.StatusCode() != 200 {
		Logs("WARN", "HTTP", "%s 请求返回非200状态码: %d，URL: %s，耗时: %dms", method, resp.StatusCode(), WebUrl, duration.Milliseconds())
		return string(resp.Body()), fmt.Errorf("%d", resp.StatusCode())  //针对API返回的错误码
	}
	//Logs("INFO", "HTTP", "%s 请求成功，URL: %s，耗时: %dms，状态码: %d", method, WebUrl, duration.Milliseconds(), resp.StatusCode())
	return string(resp.Body()), nil
}

// SetHeaders 设置默认请求头
func (h *WsyHttp) SetHeaders(req *resty.Request) {
	// 默认请求头
	defaultHeaders := map[string]string{
		"User-Agent"       : "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept"           : "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language"  : "zh-CN,zh;q=0.9,en;q=0.8",
		"Accept-Encoding"  : "gzip, deflate",
		"Connection"       : "keep-alive",
		"WSYGO"            : "Client",
		//"Wsy-Key"          : Key.Encode(Date.ToTime()),
		"Upgrade-Insecure-Requests": "1",
	}
	// 先设置默认请求头
	for k, v := range defaultHeaders {
		req.SetHeader(k, v)
	}
	// 如果设置了自定义Header，覆盖默认值
	if h.Header != nil {
		for k, v := range h.Header {
			req.SetHeader(k, v)
		}
	}
}
// SetGET 处理GET请求的查询参数
func (h *WsyHttp) SetGET(req *resty.Request, WebUrl string, d interface{}) string {
	if s, ok := d.(string); ok && s != "" {
		if strings.Contains(WebUrl, "?") {
			WebUrl = WebUrl + "&" + s
		} else {
			WebUrl = WebUrl + "?" + s
		}
	} else if m, ok := d.(map[string]string); ok {
		req.SetQueryParams(m)
	} else if m, ok := d.(map[string]interface{}); ok {
		pm := make(map[string]string)
		for k, v := range m {
			pm[k] = fmt.Sprintf("%v", v)
		}
		req.SetQueryParams(pm)
	}
	return WebUrl
}
// SetPOST 处理POST请求的请求体
func (h *WsyHttp) SetPOST(req *resty.Request, d interface{}) {
	var contentType string
	var body interface{}
	body = d
	if s, ok := d.(string); ok {
		if strings.HasPrefix(strings.TrimSpace(s), "{") || strings.HasPrefix(strings.TrimSpace(s), "[") {
			contentType = "application/json"
		} else if strings.Contains(s, "=") {
			contentType = "application/x-www-form-urlencoded"
		}
	} else if d != nil {
		contentType = "application/json"
	}
	if contentType != "" {
		req.SetHeader("Content-Type", contentType)
	}
	req.SetBody(body)
}

// UrlEncode 对字符串进行URL编码
func (h *WsyHttp) UrlEncode(value string) string {
	return url.QueryEscape(value)
}

// UrlDecode 对URL编码的字符串进行解码
func (h *WsyHttp) UrlDecode(value string) (string, error) {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return "", fmt.Errorf("URL解码失败: %v", err)
	}
	return decoded, nil
}

// EchoHeaders 打印请求头信息（调试用）
func (h *WsyHttp) EchoHeaders(req *resty.Request) {
	headers := req.Header
	headerCount := len(headers)
	Echo("=== HTTP请求头信息 ===")
	Echo("请求头总数: %d", headerCount)
	for key, values := range headers {
		if len(values) == 1 {
			Echo("%s: %s", key, values[0])
		} else {
			Echo("%s: %v", key, values)
		}
	}
	Echo("========================")
}
// WebIPV6 专门用于IPv6的HTTP请求，支持多种HTTP方法
func (h *WsyHttp) WebIPV6(method, WebUrl string, data ...interface{}) (string, error) {
	var d interface{}
	var resp *resty.Response
	var err error
	// 初始化客户端
	h.SetInit()
	// 解析参数
	if len(data) > 0 { 
		d = data[0] 
	}
	client := resty.New()
	client.SetDebug(false)
	client.SetLogger(&WsyHttp{})
	client.SetRetryCount(h.Retry)
	client.SetTimeout(time.Duration(h.Timeout) * time.Second)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	client.SetTLSClientConfig(tlsConfig)
	transport := &http.Transport{
		TLSClientConfig:       tlsConfig, // 在Transport中也设置TLS配置，跳过证书验证
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			if net.ParseIP(host) == nil {
				ipv6, err := Dns.IPV6(host)
				if err != nil {
					return nil, err
				}
				host = ipv6
			}
			targetAddr := net.JoinHostPort(host, port)
			dialer := &net.Dialer{
				Timeout:   time.Duration(h.ConnectTimeout) * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return dialer.DialContext(ctx, network, targetAddr)
		},
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ForceAttemptHTTP2:     false, // 禁用HTTP/2，避免IPv6兼容性问题
		DisableKeepAlives:     false, // 启用KeepAlive
	}
	client.SetTransport(transport)
	req := client.R() // 设置默认请求头（包含Wsy.Http.Header）
	h.SetHeaders(req)
	// 处理请求体
	if method == "GET" {
		WebUrl = h.SetGET(req, WebUrl, d)
	} else {
		h.SetPOST(req, d)
	}
	// 记录请求开始时间
	start := time.Now()
	// 发送请求
	switch strings.ToUpper(method) {
	case "POST":
		resp, err = req.Post(WebUrl)
	case "PUT":
		resp, err = req.Put(WebUrl)
	case "GET":
		resp, err = req.Get(WebUrl)
	default:
		return "", fmt.Errorf("不支持的HTTP方法: %s", method)
	}
	duration := time.Since(start)
	// 处理响应
	if err != nil {
		Logs("ERROR", "HTTP", "%s 请求失败，URL: %s，耗时: %dms，错误: %v", method, WebUrl, duration.Milliseconds(), err)
		return "", err
	}
	// 检查状态码
	if resp.StatusCode() != 200 {
		Logs("WARN", "HTTP", "%s 请求返回非200状态码: %d，URL: %s，耗时: %dms", method, resp.StatusCode(), WebUrl, duration.Milliseconds())
		return string(resp.Body()), fmt.Errorf("%d", resp.StatusCode())  //针对API返回的错误码
	}
	//Logs("INFO", "HTTP", "%s 请求成功，URL: %s，耗时: %dms，状态码: %d", method, WebUrl, duration.Milliseconds(), resp.StatusCode())
	return string(resp.Body()), nil
}
//判断是否带http和域名
func (h *WsyHttp) IsHTTP(value string) bool {
	return Str.Valid(value,27)
}

//判断是否带http和域名
func (h *WsyHttp) GetHostAndPort(value string) (string,string) {
	if Str.Valid(value,27) {
		re := regexp.MustCompile(`^(http://|https://)?([a-zA-Z0-9.-]+)(?::(\d+))?`)
		matches := re.FindStringSubmatch(value)
		if matches == nil {
			return "", ""
		}
		protocol := matches[1]
		return protocol + matches[2], matches[3]
	}
	return "",""
}