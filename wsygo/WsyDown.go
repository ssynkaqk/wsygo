package Wsy

import (
	"io"
	"os"
	"fmt"
	"time"
	"syscall"
	"strings"
	"net/http"
	"path/filepath"
	"net"
	"net/url"
	"context"
)

type WsyDown struct {
	Timeout  int    // 超时时间 分钟
	MinDisk  int64  // 最小磁盘空间要求（MB）
	DelPath  bool   // 是否强制删除文件或目录
	ShowTips bool   // 是否开启进度显示
}

func (d *WsyDown) SetInit() {
	d.Timeout  = Str.IIFS(d.Timeout != 0, d.Timeout, int(30)).(int)
	d.MinDisk  = Str.IIFS(d.MinDisk != 0, d.MinDisk, int64(5)).(int64)
	d.DelPath  = Str.IIFS(d.DelPath, d.DelPath, false).(bool)
	d.ShowTips = Str.IIFS(d.ShowTips, d.ShowTips, true).(bool)
}

func (d *WsyDown) New(url, savePath string, options ...bool) (string, error) {
	d.SetInit()
	if len(options) > 0 { d.DelPath = options[0] }
	if len(options) > 1 { d.ShowTips = options[1] }
	// 1. 准备下载环境
	NewSavePath, err := d.ToSavePath(url, savePath, d.DelPath)
	if err != nil {
		return "", err
	}
	// 2、创建HTTP请求获取文件大小
	resp, err := d.NewHTTP(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	// 3、检查磁盘空间（使用实际文件大小）
	if err := d.CheckDisk(filepath.Dir(NewSavePath), resp.ContentLength); err != nil {
		return "", err
	}
	// 4、创建临时文件
	tempFile := NewSavePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	file.Close()
	
	// 5、执行文件下载
	if err := d.DownFile(resp, tempFile, NewSavePath,resp.ContentLength); err != nil {
		return "", err
	}
	// 6、完成下载
	if _,err := Fso.ReName(tempFile,NewSavePath,d.DelPath); err != nil {
		return "", err
	}
	return NewSavePath, nil
}

// ToValid 检查路径是否包含非法字符
func (d *WsyDown) ToValid(path string) bool {
	invalidChars := "!@#$%^&*()+~;':、<,>。（）?`"
	for _, char := range invalidChars {
		if strings.Contains(path, string(char)) {
			return true
		}
	}
	// 检查路径是否完全由点组成 如 ...
	if path != "" && strings.Count(path, ".") == len(path) && len(path) > 0 {
		return true
	}
	// 检查开头多个点的情况：如 ../ 或 /... 或 /.../
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			firstPart := parts[0]
			if firstPart != "" && strings.Count(firstPart, ".") == len(firstPart) && len(firstPart) > 0 {
				return true
			}
		}
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if lastPart != "" && strings.Count(lastPart, ".") == len(lastPart) && len(lastPart) > 0 {
				return true
			}
		}
		for _, part := range parts {
			if part != "" && strings.Count(part, ".") == len(part) && len(part) > 0 {
				return true
			}
		}
	}
	return false
}
// ToSavePath 准备下载环境
func (d *WsyDown) ToSavePath(Url,Path string,IsDel ...bool) (string,error) {
	if Url == "" { return "", fmt.Errorf("下载URL不能为空")}
	if len(IsDel) > 0 { d.DelPath = IsDel[0] }
	//判断路径是否合法
	if d.ToValid(Path) {
		return "",fmt.Errorf("无效路径: %s", Path)
	}
	//如果路径结尾是/或\，则将URL的文件名拼接上去
	if strings.HasSuffix(Path, "/") || strings.HasSuffix(Path, "\\") {
		Path = filepath.Join(Path, filepath.Base(Url))
	}
	if Path == "" {
		Path = filepath.Base(Url)
	}
	
	IsNewPath := Fso.IsPath(Path)
	//创建目录
	isMakePath := filepath.Dir(IsNewPath)
	// 只有当目录路径不是当前目录时才创建  ，如果存在文件，先判断是否强制删除文件再创建目录
	if isMakePath != "." && isMakePath != "" {
		if Fso.IsFile(isMakePath) {  //如果是个文件
			if d.DelPath {  //如果开启强制删除
				if err := os.Remove(isMakePath); err != nil {
					return "" ,fmt.Errorf("删除文件失败: %v (path: %s)", err, isMakePath)
				}
			}
		}
		if err := os.MkdirAll(isMakePath, 0777); err != nil {
			return "",fmt.Errorf("创建目录失败,当前存在文件: %v (path: %s)", err, isMakePath)
		}
	}
	//创建文件
	if Fso.Exists(IsNewPath) {
		fileInfo, fileErr := os.Stat(IsNewPath)
		if fileErr != nil {
			return "",fmt.Errorf("获取文件信息失败: %v (path: %s)", fileErr, IsNewPath)
		}
		if fileInfo.IsDir() {
			if d.DelPath {
				if err := os.RemoveAll(IsNewPath); err != nil {
					return "",fmt.Errorf("删除目录失败: %v (path: %s)", err, IsNewPath)
				}
			}else{
				IsNewPath = Fso.AddPath(IsNewPath,filepath.Base(IsNewPath))
			}
		}
	}
	return IsNewPath, nil
}

// NewHTTP 创建HTTP请求
func (d *WsyDown) NewHTTP(targetURL string) (*http.Response, error) {
	ip, err := Dns.New(targetURL)
	if err != nil {
		return nil, err
	}
	host := d.GetHostname(targetURL)
	
	client := &http.Client{
		Timeout: time.Duration(d.Timeout) * time.Minute,
	}
	// 如果解析出的 IP 与原始 host 不同，说明进行了 DNS 解析，需要设置自定义 Transport
	if ip != "" && host != "" && ip != host {
		client.Transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				_, port, _ := net.SplitHostPort(addr)
				targetAddr := net.JoinHostPort(ip, port)
				return (&net.Dialer{}).DialContext(ctx, network, targetAddr)
			},
		}
	}
	
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	
	// 设置标准的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("WSYGO", "Client")
	
	// 如果解析出的 IP 与原始 host 不同，说明进行了 DNS 解析，需要设置 Host 头
	if ip != "" && host != "" && ip != host {
		req.Header.Set("Host", host)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %v", err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}
	return resp, nil
}

// DownFile 执行文件下载
func (d *WsyDown) DownFile(resp *http.Response, tempFile, savePath string, totalSize int64) error {
	file, err := os.OpenFile(tempFile, os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("打开临时文件失败: %v", err)
	}
	defer file.Close()
	
	buffer := make([]byte, 64*1024) // 64KB缓冲区
	var downloaded int64
	startTime := time.Now()
	lastUpdateTime := time.Now()
	updateInterval := 200 * time.Millisecond
	
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("写入文件失败: %v", writeErr)
			}
			if d.ShowTips {
				downloaded += int64(n)
				if d.DownTips(downloaded, totalSize, startTime, lastUpdateTime, savePath, updateInterval) {
					lastUpdateTime = time.Now()
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取数据失败: %v", err)
		}
	}
	// 下载完成，强制显示100%进度
	if d.ShowTips {
		d.DownTips(downloaded, totalSize, startTime, time.Time{}, savePath, 0, true)
	}
	fmt.Println()
	return nil
}

// DownTips 更新下载进度显示
func (d *WsyDown) DownTips(downloaded, totalSize int64, startTime, lastUpdateTime time.Time, savePath string, updateInterval time.Duration, force100 ...bool) bool {
	forceShow100 := len(force100) > 0 && force100[0]
	if !d.ShowTips || (!forceShow100 && time.Since(lastUpdateTime) < updateInterval) { return false }
	fileName := filepath.Base(savePath)
	elapsedSeconds := int(time.Since(startTime).Seconds())
	if elapsedSeconds == 0 { elapsedSeconds = 1 }
	speedKB := float64(downloaded) / float64(elapsedSeconds) / 1024
	if totalSize > 0 {
		percentage := float64(downloaded) / float64(totalSize) * 100
		if forceShow100 { percentage = 100.0 }
		// 创建进度条
		filled := int(percentage / 100 * 50)
		bar := ""
		for i := 0; i < filled; i++ { bar += "=" }
		if filled < 50 { bar += ">"; for i := filled + 1; i < 50; i++ { bar += " " } }
		sizeInfo := fmt.Sprintf("%s/%s", d.FormatSize(float64(downloaded)), d.FormatSize(float64(totalSize)))
		fmt.Printf("\r%s  [%s]%3.0f%%   %s   %dKB/s   downTime:%ds\033[K", fileName, bar, percentage, sizeInfo, int(speedKB), elapsedSeconds)
		os.Stdout.Sync()
	} else {
		fmt.Printf("\r%s  %s  %dKB/s  downTime:%ds\033[K", fileName, d.FormatSize(float64(downloaded)), int(speedKB), elapsedSeconds)
		os.Stdout.Sync()
	}
	return true
}

// CheckDisk 检查磁盘可用空间
func (d *WsyDown) CheckDisk(path string, fileSize ...int64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil // 无法获取磁盘空间信息时，允许继续执行
	}
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	requiredBytes  := int64(10 * 1024 * 1024) // 默认10MB
	if len(fileSize) > 0 && fileSize[0] > 0 {
		requiredBytes = fileSize[0]
	} else {
		requiredBytes = d.MinDisk * 1024 * 1024
	}
	if availableBytes > 0 && uint64(requiredBytes) > availableBytes {
		availableSize := d.FormatSize(float64(availableBytes))
		requiredSize := d.FormatSize(float64(requiredBytes))
		return fmt.Errorf("磁盘空间不足: 可用 %s，需要 %s", availableSize, requiredSize)
	}
	return nil
}

// FormatSize 自适应格式化大小
func (d *WsyDown) FormatSize(value float64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := value
	unitIndex := 0
	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024;unitIndex++
	}
	return fmt.Sprintf("%.1f%s", size, units[unitIndex])
}

// GetHostname 从URL中提取主机名
func (d *WsyDown) GetHostname(targetURL string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}