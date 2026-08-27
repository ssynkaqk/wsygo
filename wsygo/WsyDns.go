package Wsy

import (
	"fmt"
	"net"
	"regexp"
	"net/http"
	"strings"
	"time"
	"crypto/tls"
	"context"
	"github.com/go-resty/resty/v2"
	"github.com/miekg/dns"
)

type WsyDns struct {
	Timeout     int
	DnsOld     []string
	DnsDoh     []string
	DnsDot     []string
}

// WsyDnsResult 通用DNS解析结果结构体
type WsyDnsResult struct {
	ip     string
	server string
	err    error
}
func (d *WsyDns) SetInit() { //初始化配置
	
	d.Timeout = Str.IIFS(d.Timeout != 0, d.Timeout, int(2)).(int)    //DNS超时时间

	d.DnsOld  = []string{
		"223.5.5.5:53",       // 阿里DNS主服务器 - 国内访问速度快，解析准确
		"223.6.6.6:53",       // 阿里DNS备用服务器 - 阿里云提供的备用DNS
		"119.29.29.29:53",    // 腾讯DNS - 腾讯云DNS服务，稳定可靠
		"114.114.114.114:53", // 114DNS - 国内老牌DNS服务商，历史悠久
		"180.76.76.76:53",    // 百度DNS - 百度提供的公共DNS服务
		"101.226.4.6:53",     // 360DNS主服务器 - 360安全DNS，防劫持
		"218.30.118.6:53",    // 360DNS备用服务器 - 360DNS备用节点
		"182.254.116.116:53", // 电信DNS - 中国电信官方DNS
		"8.8.8.8:53",         // Google DNS - 全球最大的公共DNS服务
	}
	d.DnsDoh = []string{
		"https://doh.pub/dns-query",        // 腾讯 DoH
		"https://doh.360.cn/dns-query",     // 360 DoH
		"https://dns.alidns.com/dns-query", // 阿里 DoH
		"https://cloudflare-dns.com/dns-query", // Cloudflare DoH
		"https://dns.google/dns-query",     // Google DoH
	}
	d.DnsDot = []string{
		"dns.alidns.com:853", // 阿里 DoT
		"dot.pub:853",        // 腾讯 DoT
		"dot.360.cn:853",     // 360 DoT
		"8.8.8.8:853",        // Google DoT
		"9.9.9.9:853",        // Quad9 DoT
	}
}
// New 使用自定义DNS解析域名
func (d *WsyDns) New(host string) (string, error) {
	d.SetInit()
	host, err := d.HostName(host)
	if err != nil {
		return "", err
	}
	// 1. 尝试传统DNS
	if ip, err := d.DNS(host); err == nil {
		return ip, nil
	}
	// 2. 尝试系统DNS
	if ip, err := d.Local(host); err == nil {
		return ip, nil
	}
	// 3. 尝试DoH
	if ip, err := d.DOH(host); err == nil {
		return ip, nil
	}
	// 4. 尝试DoT
	if ip, err := d.DOT(host); err == nil {
		return ip, nil
	}
	// 5. 尝试nslookup
	if ip, err := d.Nslookup(host); err == nil {
		return ip, nil
	}
	// 6. 尝试ping
	if ip, err := d.Ping(host); err == nil {
		return ip, nil
	}
	return "", fmt.Errorf("所有DNS解析失败: %s", host)
}
// DNS 使用传统DNS解析域名
func (d *WsyDns) DNS(host string) (string, error) {
	c := &dns.Client{Timeout: time.Duration(d.Timeout) * time.Second}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)
	m.RecursionDesired = true
	resultChan := make(chan WsyDnsResult, len(d.DnsOld))
	for _, dnsServer := range d.DnsOld {
		go func(server string) {
			r, _, err := c.Exchange(m, server)
			if err != nil {
				resultChan <- WsyDnsResult{"", server, err}
				return
			}
			if r == nil {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("DNS响应为空")}
				return
			}
			for _, ans := range r.Answer {
				if a, ok := ans.(*dns.A); ok {
					ip := a.A.String()
					resultChan <- WsyDnsResult{ip, server, nil}
					return
				}
			}
			resultChan <- WsyDnsResult{"", server, fmt.Errorf("未找到A记录")}
		}(dnsServer)
	}
	// 等待第一个成功的结果
	for i := 0; i < len(d.DnsOld); i++ {
		result := <-resultChan
		if result.err == nil && result.ip != "" {
			//Logs("INFO", "HTTP", "DNS解析成功: %s -> %s (使用DNS: %s)", host, result.ip, result.server)
			return result.ip, nil
		}
	}
	return "", fmt.Errorf("DNS解析失败: %s", host)
}
// DOH 使用DoH解析域名
// 参数:
//   - hostname: 域名
// 返回:
//   - ip: 解析结果
//   - err: 错误信息
func (d *WsyDns) DOH(hostname string) (string, error) {
	// 创建DNS查询消息
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(hostname), dns.TypeA)
	m.RecursionDesired = true
	resultChan := make(chan WsyDnsResult, len(d.DnsDoh))
	for _, dohServer := range d.DnsDoh {
		go func(server string) {
			msgBytes, err := m.Pack()
			if err != nil {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("DNS消息打包失败: %v", err)}
				return
			}
			resp, err := resty.New().
			//SetLogger(nil).
			SetTimeout(time.Duration(d.Timeout) * time.Second).
			R().
			SetHeader("Content-Type", "application/dns-message").
			SetHeader("Accept", "application/dns-message").
			SetHeader("User-Agent", "WSYGO-DoH-Client/1.0").
			SetBody(msgBytes).
			Post(server)
			if err != nil {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("DoH请求失败: %v", err)}
				return
			}
			if resp.StatusCode() != http.StatusOK {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("DoH服务器返回错误状态码: %d", resp.StatusCode())}
				return
			}
			dnsResp := new(dns.Msg)
			err = dnsResp.Unpack(resp.Body())
			if err != nil {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("解析DoH响应失败: %v", err)}
				return
			}
			for _, ans := range dnsResp.Answer {
				if a, ok := ans.(*dns.A); ok {
					ip := a.A.String()
					resultChan <- WsyDnsResult{ip, server, nil}
					return
				}
			}
			resultChan <- WsyDnsResult{"", server, fmt.Errorf("未找到A记录")}
		}(dohServer)
	}
	for i := 0; i < len(d.DnsDoh); i++ {
		result := <-resultChan
		if result.err == nil && result.ip != "" {
			//Logs("INFO", "HTTP", "DoH解析成功: %s -> %s (使用DoH: %s)", hostname, result.ip, result.server)
			return result.ip, nil
		}
	}
	return "", fmt.Errorf("DoH解析失败: %s", hostname)
}
// DOT 使用DoT解析域名
func (d *WsyDns) DOT(hostname string) (string, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(hostname), dns.TypeA)
	m.RecursionDesired = true
	resultChan := make(chan WsyDnsResult, len(d.DnsDot))
	for _, dotServer := range d.DnsDot {
		go func(server string) {
			tlsConfig := &tls.Config{
				ServerName:         strings.Split(server, ":")[0],
				InsecureSkipVerify: false,
			}
			c := &dns.Client{
				Net:     "tcp-tls",
				Timeout: time.Duration(d.Timeout) * time.Second,
				TLSConfig: tlsConfig,
			}
			r, _, err := c.Exchange(m, server)
			if err != nil {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("DoT请求失败: %v", err)}
				return
			}
			if r == nil {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("DoT响应为空")}
				return
			}
			for _, ans := range r.Answer {
				if a, ok := ans.(*dns.A); ok {
					ip := a.A.String()
					resultChan <- WsyDnsResult{ip, server, nil}
					return
				}
			}
			resultChan <- WsyDnsResult{"", server, fmt.Errorf("未找到A记录")}
		}(dotServer)
	}
	for i := 0; i < len(d.DnsDot); i++ {
		result := <-resultChan
		if result.err == nil && result.ip != "" {
			//Logs("INFO", "HTTP", "DoT解析成功: %s -> %s (使用DoT: %s)", hostname, result.ip, result.server)
			return result.ip, nil
		}
	}
	return "", fmt.Errorf("DoT解析失败: %s", hostname)
}
// Local 使用系统DNS解析域名
func (d *WsyDns) Local(host string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.Timeout) * time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err == nil && len(ips) > 0 {
		for _, v := range ips {
			if v.To4() != nil {
				//Logs("INFO", "HTTP", "系统DNS解析成功: %s => %s", host, v.String())
				return v.String(), nil
			}
		}
	}
	return "", fmt.Errorf("系统DNS解析失败: %s", host)
}
// Nslookup 使用nslookup解析域名
// 参数:
//   - host: 域名
// 返回:
//   - ip: 解析结果
//   - err: 错误信息
func (d *WsyDns) Nslookup(host string) (string, error) {
	ip4 := Shell.Shell(fmt.Sprintf("nslookup %s", host))
	sp4 := strings.Split(ip4, "\n")
	for i, line := range sp4 {
		line = strings.TrimSpace(line)
		if Str.IsIn(line, host) && Str.IsIn(line, "Name") {
			if i+1 < len(sp4) {
				nextLine := strings.TrimSpace(sp4[i+1])
				if strings.Contains(nextLine, ":") {
					parts := strings.SplitN(nextLine, ":", 2)
					if len(parts) == 2 {
						ip := strings.TrimSpace(parts[1])
						if net.ParseIP(ip) != nil {
							//Logs("INFO", "HTTP", "nslookup解析成功: %s => %s", host, ip)
							return ip, nil
						}
					}
				}
			}
		}
	}
	return "", fmt.Errorf("nslookup解析失败: %s", host)
}
// Ping 使用ping解析域名
// 参数:
//   - host: 域名
// 返回:
//   - ip: 解析结果
//   - err: 错误信息
func (d *WsyDns) Ping(host string) (string, error) {
	ip5 := Shell.Shell(fmt.Sprintf("ping -c 1 -W 2 %s", host))
	sp5 := strings.Split(ip5, "\n")
	for _, line := range sp5 {
		if strings.Contains(line, "PING") && strings.Contains(line, "(") && strings.Contains(line, ")") {
			start := strings.Index(line, "(")
			end := strings.Index(line, ")")
			if start != -1 && end != -1 && end > start {
				ip := strings.TrimSpace(line[start+1 : end])
				if net.ParseIP(ip) != nil {
					//Logs("INFO", "HTTP", "PING解析成功: %s => %s", host, ip)
					return ip, nil
				}
			}
		}
	}
	return "", fmt.Errorf("PING解析失败: %s", host)
}

// IPV6 使用IPv6解析域名
// 参数:
//   - host: 域名
// 返回:
//   - ip: 解析结果
//   - err: 错误信息
func (d *WsyDns) IPV6(host string) (string, error) {
	d.SetInit()
	host, err := d.HostName(host)
	if err != nil {
		return "", err
	}
	// 检查本机是否支持IPv6
	c := &dns.Client{Timeout: time.Duration(d.Timeout) * time.Second}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeAAAA)
	m.RecursionDesired = true
	resultChan := make(chan WsyDnsResult, len(d.DnsOld))
	for _, dnsServer := range d.DnsOld {
		go func(server string) {
			r, _, err := c.Exchange(m, server)
			if err != nil {
				resultChan <- WsyDnsResult{"", server, err}
				return
			}
			if r == nil {
				resultChan <- WsyDnsResult{"", server, fmt.Errorf("DNS响应为空")}
				return
			}
			for _, ans := range r.Answer {
				if aaaa, ok := ans.(*dns.AAAA); ok {
					ip := aaaa.AAAA.String()
					resultChan <- WsyDnsResult{ip, server, nil}
					return
				}
			}
			resultChan <- WsyDnsResult{"", server, fmt.Errorf("未找到AAAA记录")}
		}(dnsServer)
	}
	for i := 0; i < len(d.DnsOld); i++ {
		result := <-resultChan
		if result.err == nil && result.ip != "" {
			//Logs("INFO", "HTTP", "DNS IPv6解析成功: %s -> %s (使用DNS: %s)", host, result.ip, result.server)
			return result.ip, nil
		}
	}
	return "", fmt.Errorf("DNS IPv6解析失败: %s", host)
}
// HostName 从URL中提取主机名
// 参数:
//   - domain: 域名
// 返回:
//   - host: 主机名
//   - err: 错误信息
func (d *WsyDns) HostName(domain string) (string, error) {
	re := regexp.MustCompile(`^(?:https?://)?([^/]+)`)
	matches := re.FindStringSubmatch(domain)
	if len(matches) < 2 {
		return "", fmt.Errorf("无法解析URL: %s", domain)
	}
	host := matches[1]
	// 如果包含端口号，提取IP/域名部分
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return "", fmt.Errorf("端口解析失败: %v", err)
		}
	}
	// 如果已经是IP地址，直接返回
	if net.ParseIP(host) != nil {
		return host, nil
	}
	return host, nil
}