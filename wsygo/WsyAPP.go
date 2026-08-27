package Wsy

import (
	"io"
	"os"
	"fmt"
	"sync"
	"time"
	"bytes"
	"strings"
	"os/exec"
	"archive/zip"
	"encoding/xml"
	"path/filepath"
	"encoding/base64"
	"github.com/avast/apkparser"
)

// WsyAPP 应用管理结构体
type WsyAPP struct{}


func (a *WsyAPP) New(path string) (map[string]string, error) {
	result, err := a.ToXml(path)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// List 批量解析多个APK文件（并发处理，最多10个goroutine同时处理）
// 参数:
//   - file: APK文件路径，多个文件用逗号分隔
//
// 返回:
//   - []map[string]string，每个元素包含一个文件的信息
//   - error，如果解析失败则返回错误信息
func (a *WsyAPP) List(file string, value ...bool) ([]map[string]string, error) {
	if file == "" {
		return nil, Error("文件路径不能为空")
	}
	files := strings.Split(file, ",")
	// 过滤空文件路径
	validFiles := make([]string, 0, len(files))
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f != "" {
			validFiles = append(validFiles, f)
		}
	}
	if len(validFiles) == 0 {
		return []map[string]string{}, nil
	}
	// 使用低负载并发，避免批量解析时把设备资源打满
	const maxWorkers = 2
	result := make([]map[string]string, len(validFiles))
	var wg sync.WaitGroup
	
	// 创建任务通道
	tasks := make(chan int, len(validFiles))
	for i := range validFiles {
		tasks <- i
	}
	close(tasks)
	// 启动worker goroutines
	workerCount := len(validFiles)
	if workerCount > maxWorkers {
		workerCount = maxWorkers
	}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range tasks {
				f := validFiles[idx]
				info, err := a.ToXml(f, value...)
				if err != nil {
					result[idx] = map[string]string{
						"file"    : f,
						"md5"     : "",
						"size"    : "",
						"role"    : "",
						"label"   : "",
						"name"    : "",
						"package" : "",
						"version" : "",
						"start"   : "",
					}
				} else {
					result[idx] = info
				}
				// 让出CPU和IO，避免在低性能设备上持续冲高负载
				time.Sleep(120 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	return result, nil
}

//使用获取所有应用列表
func (a WsyAPP) Get() (string, error) {
	output, err := Shell.Run("pm list packages -f")
	if err != nil {
		return "", Error("获取应用列表失败: %s", err)
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return "", Error("获取应用列表为空")
	}
	lines := strings.Split(output, "\n")
	apkPaths := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "package:") {
			continue
		}
		lastEq := strings.LastIndex(line, "=")
		if lastEq <= 0 {
			continue
		}
		apkPath := strings.TrimPrefix(line[:lastEq], "package:")
		apkPath = strings.TrimSpace(apkPath)
		if apkPath == "" {
			continue
		}
		apkPaths = append(apkPaths, apkPath)
	}
	if len(apkPaths) == 0 {
		return "", Error("获取应用列表为空")
	}
	return strings.Join(apkPaths, ","), nil
}
// ListAll 列出设备上所有已安装的APK并解析信息
// 返回:
//   - []map[string]string，每个元素包含一个应用的详细信息
//   - error，如果执行失败则返回错误信息
func (a WsyAPP) ListInfo(filter string, value ...bool) ([]map[string]string, error) {
	apkFiles, err := a.Get()
	if err != nil {
		return nil, err
	}
	pathTypeMap := make(map[string]string)
	systemPaths := []string{"/system/", "/vendor/", "/product/", "/oem/"}
	for _, apkPath := range strings.Split(apkFiles, ",") {
		apkPath = strings.TrimSpace(apkPath)
		if apkPath == "" {
			continue
		}
		appType := "user"
		for _, sysPath := range systemPaths {
			if strings.HasPrefix(apkPath, sysPath) {
				appType = "system"
				break
			}
		}
		pathTypeMap[apkPath] = appType
	}
	result, err := a.List(apkFiles, value...)
	if err != nil {
		return nil, err
	}
	for i := range result {
		if filePath, ok := result[i]["file"]; ok {
			if appType, exists := pathTypeMap[filePath]; exists {
				result[i]["userType"] = appType
			}
		}
	}
	return result, nil
}

//apk文件，更多信息，显示LOGO编码
func (a *WsyAPP) ToXml(apkFile string,value ...bool) (map[string]string, error) {
	apkFile = Fso.AddPath(apkFile)
	if !Fso.Exists(apkFile) { return nil, Error("APK文件不存在") }
	if !strings.HasSuffix(strings.ToLower(apkFile), ".apk") { return nil, Error("不是APK文件") }
	type InfoData struct {
		Package     string   `xml:"package,attr"`
		VersionName string   `xml:"versionName,attr"`
		App struct {
				Label string `xml:"label,attr"`
				Icon  string `xml:"icon,attr"`
			} `xml:"application"`
		RoleData []struct {
				Name string `xml:"name,attr"`
			} `xml:"uses-permission"`
	}
	// 解析 XML manifest
	var list InfoData
	var More,Logo bool = false,false

	if len(value) > 0 && value[0] {
		More = true
	}
	if len(value) == 2 && value[1] {
		Logo = true
	}

	XmlData, err := a.OpenToXml(apkFile)
	if err != nil {
		return nil, Error("打开APK失败: %s", err)
	}
	if err := xml.Unmarshal([]byte(XmlData), &list); err != nil {
		return nil, Error("解析Manifest XML失败: %s", err)
	}
	// 获取文件大小
	fileInfo, fileErr := os.Stat(apkFile)
	if fileErr != nil {
		return nil, Error("打开APK文件失败：%s", fileErr)
	}
	RelData := map[string]string{
		"md5"     : Fso.GetMD5(apkFile),
		"file"    : apkFile,
		"size"    : Str.ToString(fileInfo.Size()),
		"label"   : list.App.Label,
		"name"    : fileInfo.Name(),
		"package" : list.Package,
		"version" : list.VersionName,
		"start"   : a.Activity(XmlData),
	}
	Logs("INFO", "APP", "解析: package=%s label=%s version=%s file=%s", list.Package, list.App.Label, list.VersionName, apkFile)
	if More {
		Permissions := make([]string, 0, len(list.RoleData))
		for _, p := range list.RoleData {
			if p.Name != "" {
				Permissions = append(Permissions, p.Name)
			}
		}
		RelData["role"] = strings.Join(Permissions, ",")
	}
	if Logo {
		RelData["logo"]  = list.App.Icon
		RelData["image"] = a.IconToBase64(apkFile, list.App.Icon)
	}
	return RelData, nil
}

// Activity 从 AndroidManifest.xml 文本中提取启动页 Activity（MAIN + LAUNCHER）。
// 返回值保持 Manifest 原样（可能是 ".MainActivity" 或 "com.xx.MainActivity"）。
func (a *WsyAPP) Activity(manifestXml string) string {
	dec := xml.NewDecoder(strings.NewReader(manifestXml))
	inComp, inFilter := false, false
	compName := ""
	hasMain, hasLaunch := false, false
	pkg := ""
	attrName := func(attrs []xml.Attr) string { for _, at := range attrs { if at.Name.Local == "name" { return at.Value } }; return "" }
	attrPkg := func(attrs []xml.Attr) string { for _, at := range attrs { if at.Name.Local == "package" { return at.Value } }; return "" }
	isMain := func(v string) bool { return v == "android.intent.action.MAIN" || v == "MAIN" || strings.HasSuffix(v, ".MAIN") }
	isLauncher := func(v string) bool {
		return v == "android.intent.category.LAUNCHER" || v == "android.intent.category.LEANBACK_LAUNCHER" ||
			v == "LAUNCHER" || v == "LEANBACK_LAUNCHER" ||
			strings.HasSuffix(v, ".LAUNCHER") || strings.HasSuffix(v, ".LEANBACK_LAUNCHER")
	}

	filter := func(name string) string {
		// 统一输出启动页相对名：如果是 "包名.xxx" 则转成 ".xxx"
		if pkg != "" && strings.HasPrefix(name, pkg+".") {
			suffix := strings.TrimPrefix(name, pkg)
			suffix = strings.TrimPrefix(suffix, ".")
			return "." + suffix
		}
		return name
	}

	for {
		tok, err := dec.Token()
		if err != nil { break }
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "manifest":
				if pkg == "" {
					pkg = attrPkg(t.Attr)
				}
				case "activity", "activity-alias":
					inComp, compName = true, attrName(t.Attr)
				case "intent-filter":
					if inComp { inFilter, hasMain, hasLaunch = true, false, false }
				case "action":
					if inComp && inFilter && isMain(attrName(t.Attr)) { hasMain = true }
				case "category":
					if inComp && inFilter && isLauncher(attrName(t.Attr)) { hasLaunch = true }
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "intent-filter":
				if inComp && inFilter && hasMain && hasLaunch && compName != "" { return filter(compName) }
				inFilter = false
			case "activity", "activity-alias":
				inComp, compName = false, ""
			}
		}
	}
	return ""
}

func (a *WsyAPP) SaveImage(apkFile,saveFile string) (map[string]string, error) {
	RelInfo, RelErr := a.ToXml(apkFile,false,true)
	if RelErr != nil {
		return nil, RelErr
	}
	SaveData, SaveErr := a.SaveBase64ToPng(RelInfo["image"], saveFile); 
	if SaveErr != nil {
		return nil, SaveErr
	}
	delete(RelInfo,"image")
	RelInfo["logo"] = SaveData
	return RelInfo, nil
}

// OpenToXml 使用 avast/apkparser 解析 AndroidManifest.xml
func (a *WsyAPP) OpenToXml(file string) (string, error) {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("", "\t")
	zipErr, resErr, manErr := apkparser.ParseApk(file, enc)
	if zipErr != nil {
		return "", Error("打开APK失败！")
	}
	if resErr != nil {
		Logs("WARN", "APK", "资源解析警告！")
	}
	if manErr != nil {
		return "", Error("解析AndroidManifest.xml失败！")
	}
	return buf.String(), nil
}

// IconToBase64 从APK中提取指定路径的图标并转换为Base64
// 参数:
//   - apkfile: APK文件路径
//   - logoFile: APK内部图标文件路径（如 res/drawable/icon_bird.png）
//
// 返回:
//   - Base64编码的图片数据（格式：data:image/png;base64,...），未找到返回空字符串
func (a *WsyAPP) IconToBase64(apkfile string, logoFile string) string {
	if !Fso.Exists(apkfile) || logoFile == "" {
		return ""
	}
	zr, err := zip.OpenReader(apkfile)
	if err != nil {
		return ""
	}
	defer zr.Close()
	// 通过路径精确查找文件
	for _, f := range zr.File {
		if strings.ToLower(f.Name) == strings.ToLower(logoFile) {
			rc, err := f.Open()
			if err != nil {
				return ""
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil || len(data) == 0 {
				return ""
			}
			base64Data := base64.StdEncoding.EncodeToString(data)
			return "data:image/png;base64," + base64Data
		}
	}
	return ""
}
// SaveBase64ToPng 将base64编码的图片数据保存为PNG格式图片文件
// 参数:
//   - base64Data: base64编码的图片数据（支持 data:image/png;base64, 格式或纯base64字符串）
//   - saveFile: 保存文件的路径（为空则自动生成文件名）
//
// 返回:
//   - 保存的文件路径，失败返回空字符串
func (a *WsyAPP) SaveBase64ToPng(base64Data string, saveFile string) (string, error) {
	saveFile = Fso.AddPath(saveFile)
	if base64Data == "" || saveFile == "" || !strings.HasSuffix(strings.ToLower(saveFile), ".png") {
		return "", Error("图片数据、目标文件不正确或目标文件只允许.PNG格式!")
	}
	
	if idx := strings.Index(base64Data, ","); idx >= 0 {
		base64Data = base64Data[idx+1:]
	}
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", Error("无效的Base64!")
	}
	if err := os.WriteFile(saveFile, data, 0644); err != nil {
		return "", Error("写入失败，无效的目标文件！")
	}
	return saveFile,nil
}

// Install 安装APK
// 参数:
//   - path: APK文件路径或目录路径
//   - options: 安装选项
//   - "": 默认安装（清除数据）
//   - "r": 保留数据安装
//   - "f": 强制安装
//   - "d": 降级安装
//   - "g": 自动授权
//
// 返回:
//   - success: 安装成功
//   - 其他: 错误信息
func (a WsyAPP) Install(path string, options ...string) string {
	if !Fso.Exists(path) {
		return "路径不存在"
	}
	// 获取安装选项
	opt := ""
	if len(options) > 0 && options[0] != "" {
		opt = "-" + options[0]
	}
	// 目录批量安装
	if Fso.IsDir(path) {
		return a.installDir(path, opt)
	}
	// 单文件安装
	if !strings.HasSuffix(strings.ToLower(path), ".apk") {
		return "不是APK文件"
	}
	return a.installFile(path, opt)
}

// installFile 安装单个APK文件
func (a WsyAPP) installFile(path string, opt string) string {
	Shell.Shell("settings put global package_verifier_enable 0")
	Shell.Shell("settings put global verifier_verify_adb_installs 0")
	Shell.Shell("settings put global package_verifier_user_consent -1")
	// 强制安装APK
	cmd := fmt.Sprintf("pm install -f -g %s %s", opt, path)
	result := Shell.Shell(cmd)
	if result == "" {
		Logs("ERROR", "APP", "强制安装无响应: %s", filepath.Base(path))
		return "安装无响应"
	}
	if strings.Contains(strings.ToLower(result), "success") {
		Logs("INFO", "APP", "安装成功: %s", filepath.Base(path))
		return "success"
	}
	Logs("ERROR", "APP", "安装失败: %s, %s", filepath.Base(path), result)
	return result
}

// installDir 批量安装目录中的APK
func (a WsyAPP) installDir(path string, opt string) string {
	var success, fail int = 0, 0
	files, err := filepath.Glob(filepath.Join(path, "*.apk"))
	if err != nil {
		return "读取目录失败"
	}
	if len(files) == 0 {
		return "目录无APK文件"
	}
	// 打印开始信息
	Logs("INFO", "APP", "开始批量安装，共找到 %d 个APK文件", len(files))
	successFiles := make([]string, 0)
	failFiles := make([]string, 0)
	for _, file := range files {
		if !Fso.IsFile(file) {
			continue
		}
		result := a.installFile(file, opt)
		fileName := filepath.Base(file)
		if result == "success" {
			success++
			successFiles = append(successFiles, fileName)
		} else {
			fail++
			failFiles = append(failFiles, fileName)
		}
	}
	// 打印详细结果
	Logs("INFO", "APP", "----------------------------------------")
	Logs("INFO", "APP", "批量安装完成，结果统计：")
	Logs("INFO", "APP", "总数: %d", len(files))
	Logs("INFO", "APP", "成功: %d", success)
	Logs("INFO", "APP", "失败: %d", fail)
	if len(successFiles) > 0 {
		Logs("INFO", "APP", "----------------------------------------")
		Logs("INFO", "APP", "安装成功的应用:")
		for _, name := range successFiles {
			Logs("INFO", "APP", "  - %s", name)
		}
	}
	if len(failFiles) > 0 {
		Logs("INFO", "APP", "----------------------------------------")
		Logs("INFO", "APP", "安装失败的应用:")
		for _, name := range failFiles {
			Logs("INFO", "APP", "  - %s", name)
		}
	}
	Logs("INFO", "APP", "----------------------------------------")
	if fail == 0 && success > 0 {
		return "success"
	}
	return fmt.Sprintf("完成: %d成功, %d失败", success, fail)
}

// Uninstall 卸载应用，支持多个包名（用逗号分隔）
// 参数:
//   - packageNames: 应用包名，多个包名用逗号分隔
//     例如: "com.example.app1,com.example.app2"
//
// 返回:
//   - success: 全部卸载成功
//   - 其他: 错误信息
func (a WsyAPP) Uninstall(packageNames string) string {
	if packageNames == "" {
		return "包名不能为空"
	}

	// 分割包名
	packages := strings.Split(packageNames, ",")
	totalCount := len(packages)
	successCount := 0
	var failedPackages []string

	// 遍历所有包名
	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		// 获取应用安装路径
		cmd := fmt.Sprintf("pm path %s", pkg)
		result := Shell.Shell(cmd)
		if result == "" {
			Logs("ERROR", "APP", "应用未安装: %s", pkg)
			failedPackages = append(failedPackages, pkg)
			continue
		}

		// 检查是否是系统应用
		isSystem := false
		appPath := strings.TrimPrefix(result, "package:")
		systemPaths := []string{"/system/", "/vendor/", "/product/", "/oem/"}
		for _, sysPath := range systemPaths {
			if strings.HasPrefix(appPath, sysPath) {
				isSystem = true
				break
			}
		}

		var uninstallResult string
		if isSystem {
			Logs("INFO", "APP", "检测到系统应用: %s", pkg)
			uninstallResult = a.uninstallSystemApp(pkg, appPath)
		} else {
			uninstallResult = a.uninstallNormalApp(pkg)
		}

		if uninstallResult == "success" {
			successCount++
			Logs("INFO", "APP", "应用卸载成功: %s", pkg)
		} else {
			failedPackages = append(failedPackages, pkg)
			Logs("ERROR", "APP", "应用卸载失败: %s - %s", pkg, uninstallResult)
		}
	}

	// 返回卸载结果
	if successCount == totalCount {
		return "success"
	} else if successCount == 0 {
		return fmt.Sprintf("所有应用卸载失败: %s", strings.Join(failedPackages, ", "))
	} else {
		return fmt.Sprintf("部分应用卸载失败: %s", strings.Join(failedPackages, ", "))
	}
}

// uninstallSystemApp 卸载系统应用
func (a WsyAPP) uninstallSystemApp(packageName, appPath string) string {
	Logs("INFO", "APP", "开始卸载系统应用: %s", packageName)

	// 1. 先禁用应用
	Shell.Shell(fmt.Sprintf("pm disable %s", packageName))

	// 2. 重新挂载系统分区为可写
	Shell.Shell("mount -o rw,remount /system")
	Shell.Shell("mount -o rw,remount /vendor")
	Shell.Shell("mount -o rw,remount /product")
	Shell.Shell("mount -o rw,remount /oem")

	// 3. 删除应用文件
	Shell.Shell(fmt.Sprintf("rm -rf %s", appPath))

	// 4. 卸载应用
	result := Shell.Shell(fmt.Sprintf("pm uninstall --user 0 %s", packageName))

	// 5. 重新挂载系统分区为只读
	Shell.Shell("mount -o ro,remount /system")
	Shell.Shell("mount -o ro,remount /vendor")
	Shell.Shell("mount -o ro,remount /product")
	Shell.Shell("mount -o ro,remount /oem")

	// 检查卸载结果
	if strings.Contains(strings.ToLower(result), "success") {
		Logs("INFO", "APP", "系统应用卸载成功: %s", packageName)
		return "success"
	}

	// 再次检查应用是否还存在
	checkResult := Shell.Shell(fmt.Sprintf("pm path %s", packageName))
	if checkResult == "" {
		Logs("INFO", "APP", "系统应用卸载成功: %s", packageName)
		return "success"
	}

	Logs("ERROR", "APP", "系统应用卸载失败: %s, %s", packageName, result)
	return "系统应用卸载失败"
}

// uninstallNormalApp 卸载普通应用
func (a WsyAPP) uninstallNormalApp(packageName string) string {
	Logs("INFO", "APP", "开始卸载普通应用: %s", packageName)

	// 普通应用直接卸载
	result := Shell.Shell(fmt.Sprintf("pm uninstall %s", packageName))

	if strings.Contains(strings.ToLower(result), "success") {
		Logs("INFO", "APP", "应用卸载成功: %s", packageName)
		return "success"
	}

	Logs("ERROR", "APP", "应用卸载失败: %s, %s", packageName, result)
	return result
}

// Strat 启动应用
// 参数:
//   - file: 应用文件路径
//
// 返回:
//   - true: 启动成功
//   - false: 启动失败
func (a WsyAPP) Strat(packageName string) bool {
	if packageName == "" { return false }
	out, err := exec.Command("am", "start", "-n", packageName).CombinedOutput()
	if err != nil {
		return false
	}
	// 命令成功后短轮询前台窗口，确认真实切到目标页面
	if strings.Contains(string(out), "Error:") || strings.Contains(string(out), "Exception") || strings.Contains(string(out), "Permission Denial") {
		return false
	}
	time.Sleep(1000 * time.Millisecond) //等待1秒
	if a.Windows(packageName) == "1" {
		return true
	}
	return false
}
//当前运行的应用程序
func (a WsyAPP) Windows(value ...string) string {
	out, err := exec.Command("dumpsys", "window", "windows").CombinedOutput()
	if err != nil { if len(value) == 0 { return "" }; return "0" }
	pick := func(s, key string) string {
		for _, line := range strings.Split(s, "\n") {
			if !strings.Contains(line, key) { continue }
			for _, f := range strings.Fields(line) {
				if !strings.Contains(f, "/") || !strings.Contains(f, ".") { continue }
				f = strings.TrimRight(f, "},)")
				return f
			}
		}
		return ""
	}
	s := string(out)
	comp := pick(s, "mFocusedApp")
	if comp == "" { comp = pick(s, "mCurrentFocus") }
	if len(value) == 0 { return comp }
	if strings.Contains(comp, value[0]) { return "1" }
	return "0"
}

// Enable 启用应用
// pm enable com.xxsoft.mintool
func (a WsyAPP) Enable(packageName string) bool {
	if packageName == "" { return false }
	out, err := exec.Command("pm", "enable", packageName).CombinedOutput()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	if strings.Contains(s, "enabled") || strings.Contains(s, "already enabled") {
		return true
	}
	return false
}

// Disable 禁用应用
// pm disable-user --user 0 com.xxsoft.mintool
func (a WsyAPP) Disable(packageName string) bool {
	if packageName == "" { return false }
	out, err := exec.Command("pm", "disable-user", "--user", "0", packageName).CombinedOutput()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	if Str.IsIn(s, "disabled-user") || Str.IsIn(s, "already disabled") {
		return true
	}
	return false
}

// Hide 隐藏应用
func (a WsyAPP) Hide(packageName string) bool {
	if packageName == "" { return false }
	_, err := exec.Command("pm", "hide", packageName).CombinedOutput()
	if err != nil {
		return false
	}
	return true
}

// Unhide 取消隐藏应用
func (a WsyAPP) Unhide(packageName string) bool {
	if packageName == "" { return false }
	out, err := exec.Command("pm", "unhide", packageName).CombinedOutput()
	if err != nil {
		return false
	}
	if Str.IsIn(string(out), "state: false") {
		return true
	}
	return false
}

// State 返回：是否找到、是否禁用、是否隐藏
// dumpsys package com.xxsoft.mintool
func (a WsyAPP) State(packageName string) (bool, bool, bool) {
	IsFound, IsDisabled, IsHidden := false, false, false
	if packageName == "" { return false, false, false }
	out, err := exec.Command("dumpsys", "package", packageName).CombinedOutput()
	if err != nil { return false, false, false }
	data := string(out)
	if Str.IsIn(data, "unable to find package") { 
		return false, false, false
	}else{
		IsFound = true
	}
	if Str.IsIn(data, "enabled=2") || Str.IsIn(data, "enabled=3") || Str.IsIn(data, "enabled=4") {
		IsDisabled = true
	}
	if Str.IsIn(data, "hidden=true") {
		IsHidden = true
	}
	return IsFound, IsDisabled, IsHidden
}
