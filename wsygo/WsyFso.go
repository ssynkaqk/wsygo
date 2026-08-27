package Wsy
import (	
	"io"
	"os"
	"fmt"
	"regexp"
	"strings"
	"io/ioutil"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"gopkg.in/ini.v1"
)
type WsyFso struct{}
// Exists 检查文件或目录是否存在
//
// 示例：
//
//	exists := Lin.Fso.Exists("/path/to/file")
func (f *WsyFso) Exists(path string) bool {
	exists := false
	_, err := os.Stat(path)
	if err == nil || os.IsExist(err) {
		exists = true
	}
	return exists
}

// IsDir 检查路径是否为目录
//
// 示例：
//
//	isDir := Lin.Fso.IsDir("/path/to/directory")
func (f *WsyFso) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	isDir := info.IsDir()
	return isDir
}

// IsFile 检查路径是否为文件
// 参数：
//   - path: 要检查的路径
//
// 返回：
//   - bool: 是否为文件
//
// 示例：
//
//	isFile := Lin.Fso.IsFile("/path/to/file")
func (f *WsyFso) IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	isFile := !info.IsDir()
	return isFile
}


// CopyFile 复制文件到指定路径
// 如果目标文件已存在，则替换它，并设置权限为 755
//
// 参数：
//   - src: 源文件路径
//   - dst: 目标文件路径
//
// 返回：
//   - error: 错误信息
//
// 示例：
//
//	err := Lin.Fso.CopyFile("/path/to/source/file", "/path/to/destination/file")
func (f *WsyFso) CopyFile(src string, dst string) error {
	// 检查源文件是否存在
	if !f.Exists(src) {
		Logs("ERROR", "FSO", "源文件不存在: %s", src)
		return fmt.Errorf("源文件不存在: %s", src)
	}

	// 创建目标目录（如果不存在）
	dstDir := filepath.Dir(dst)
	if !f.Exists(dstDir) {
		if err := f.MakeDir(dstDir); err != nil {
			Logs("ERROR", "FSO", "创建目标目录失败: %v", err)
			return fmt.Errorf("创建目标目录失败: %v", err)
		}
	}

	// 如果目标文件已存在，先删除
	if f.Exists(dst) {
		if err := f.DelFile(dst); err != nil {
			Logs("ERROR", "FSO", "删除现有目标文件失败: %v", err)
			return fmt.Errorf("删除现有目标文件失败: %v", err)
		}
		Logs("INFO", "FSO", "已删除现有目标文件: %s", dst)
	}

	// 复制文件
	inputFile, err := os.Open(src)
	if err != nil {
		Logs("ERROR", "FSO", "打开源文件失败: %v", err)
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(dst)
	if err != nil {
		Logs("ERROR", "FSO", "创建目标文件失败: %v", err)
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer outputFile.Close()

	// 复制内容
	if _, err := io.Copy(outputFile, inputFile); err != nil {
		Logs("ERROR", "FSO", "复制文件内容失败: %v", err)
		return fmt.Errorf("复制文件内容失败: %v", err)
	}

	// 设置目标文件权限为 755
	if err := os.Chmod(dst, 0777); err != nil {
		Logs("ERROR", "FSO", "设置文件权限失败: %v", err)
		return fmt.Errorf("设置文件权限失败: %v", err)
	}
	Logs("INFO", "FSO", "文件复制完成: %s -> %s，权限设置为 777", src, dst)
	return nil
}
// AbsPath 返回绝对路径（如果已是绝对路径则直接返回，否则拼接当前工作目录）
func (f *WsyFso) AbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, path)
	}
	return path // 获取工作目录失败时，原样返回
}

// IsPath 规范化路径，删除多余的斜杠并确保路径末尾不带斜杠径
//
// 示例：
//	normalPath, isValid := Lin.Fso.IsPath("/opt//mofei/")  // 返回 "/opt/mofei", true
//	normalPath, isValid := Lin.Fso.IsPath("")             // 返回 "", false (空路径)
func (f *WsyFso) IsPath(path string) string {
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "\\", "/")
	re := regexp.MustCompile(`/+`)
	path = re.ReplaceAllString(path, "/")
	if path != "/" && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}
	return path
}

// AddPath 连接多个路径部分
//
// 示例：
//	path := Lin.Fso.AddPath("/opt/.mofei", "mofeidata")  // 返回 "/opt/.mofei/mofeidata"
//	path := Lin.Fso.AddPath("/opt/.mofei/", "/sub/dir")  // 返回 "/opt/.mofei/sub/dir"
//	path := Lin.Fso.AddPath("/opt/.mofei/", "/sub/dir", "/suba/dira")  // 返回 "/opt/.mofei/sub/dir/suba/dira"
func (f *WsyFso) AddPath(Path string, Names ...string) string {
	result := Path
	for _, name := range Names {
		result += "/" + name
	}
	return f.IsPath(result)
}

// Chmod 设置文件或目录的权限
// 参数：
//   - path: 文件或目录路径，支持通配符 *
//   - permission: 可选，权限值（如777或644），默认为777
//
// 返回：
//   - error: 错误信息，成功返回nil
//
// 示例：
//   - Fso.Chmod("/opt/.mofei")  // 只设置该文件或目录权限为777
//   - Fso.Chmod("/opt/.mofei/", 755)  // 递归设置该目录及其所有内容的权限为755
//   - Fso.Chmod("/opt/.mofei/*", 644)  // 递归设置该目录下所有内容的权限为644
func (f *WsyFso) Chmod(path string, permission ...int) error {
	var mode os.FileMode
	perm := 777
	if len(permission) > 0 && permission[0] > 0 {
		perm = permission[0]
	}
	mode = os.FileMode((perm / 100 * 64) + ((perm % 100) / 10 * 8) + perm%10)
	if strings.HasSuffix(path, "/") || strings.Contains(path, "*") {
		if strings.Contains(path, "*") {
			path = strings.TrimSuffix(path, "*")
		}
		if !f.Exists(path) {
			return fmt.Errorf("路径不存在: %s", path)
		}
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
		return filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
			if err != nil || subPath == path {
				return nil
			}
			return os.Chmod(subPath, mode)
		})
	}
	if !f.Exists(path) {
		return fmt.Errorf("权限设置失败：文件或目录不存在: %s", path)
	}
	return os.Chmod(path, mode)
}

// MakeDir 创建目录，如果是文件则删除
// 参数：
//   - path: 要创建的目录路径
//
// 返回：
//   - error: 错误信息
//
// 示例：
//
//	err := Lin.Fso.MakeDir("/path/to/directory")
func (f *WsyFso) MakeDir(value string) error {
	if f.Exists(value) {
		if f.IsDir(value) {
			return nil
		}
		if err := os.Remove(value); err != nil { // 如果路径已存在但不是目录，先删除
			return fmt.Errorf("cannot remove existing file: %v", err)
		}
	}
	err := os.MkdirAll(value, 0777)
	if err != nil {
		return err
	}
	return nil
}

// MakeDirFile 创建多级目录，如果是文件则删除
// 参数：
//   - dirPath: 目录路径（不是文件路径）
//
// 返回：
//   - bool: 成功返回true，失败返回false
//
// 说明：
//
//	此函数只处理目录路径，不要传入文件路径。
//	对于文件路径，应该先用filepath.Dir()提取目录部分，再传给此函数。
//
// 示例：
//   - Lin.Fso.MakeDirFile("/path/to/directory")
//   - dirPath := filepath.Dir("/path/to/file.txt"); Lin.Fso.MakeDirFile(dirPath)
func (f *WsyFso) MakeDirFile(value string) bool {
	DirPath := filepath.Dir(value) //提取目录
	f.MakeDir(DirPath)             //创建目录
	if f.IsDir(value) {            //判断是否目录
		f.DelDir(value) //删除目录
	}
	return true
}


// RemoveFile 删除文件
// 参数：
//   - path: 文件路径
//
// 返回：
//   - error: 错误信息
//
// 示例：
//
//	err := Lin.Fso.RemoveFile("/path/to/file")
func (f *WsyFso) DelFile(path string) error {
	if !f.Exists(path) {
		return nil
	}
	if f.IsDir(path) {
		return fmt.Errorf("cannot use RemoveFile on directory: %s", path)
	}
	err := os.Remove(path)
	if err != nil {
		return err
	}
	return nil
}
// Rename 重命名文件
// 参数：

//   - saveFile: 文件路径
//   - saveName：名称

//   - IsDel: 判断是否删除目标文件夹
//
// 返回：
//   - string: 目标文件路径
//   - error: 错误信息
//
// 示例：
//
//	saveFile, err := Lin.Fso.Rename("/path/to/source/file.tmp", "/path/to/source/file")
//	saveFile, err := Lin.Fso.Rename("/path/to/source/file.tmp", "file",true) //删除目录
//	if err != nil {
//		fmt.Println(err)
//	}
//	fmt.Println(saveFile)

func (f *WsyFso) ReName(saveFile,saveName string,IsDel ...bool) (string,error) {
	IsDelPath := true
	if len(IsDel) > 0 { IsDelPath = IsDel[0] }
	if saveName == "" {
		return "",fmt.Errorf("文件名称不能为空: %s", saveFile)
	}
	if !f.Exists(saveFile) || saveFile == "" {
		return "",fmt.Errorf("文件不存在: %s", saveFile)
	}
	saveNameDir := filepath.Base(saveName)
	saveFileDir := filepath.Dir(saveFile)
	NewFile := f.AddPath(saveFileDir,saveNameDir)
	if f.Exists(NewFile) {
		fileInfo, fileErr := os.Stat(NewFile)
		if fileErr != nil {
			return "",fmt.Errorf("获取目标文件信息失败: %v (path: %s)", fileErr, NewFile)
		}
		if fileInfo.IsDir() {
			if IsDelPath {
				if err := os.RemoveAll(NewFile); err != nil {
					return "",fmt.Errorf("删除目标文件失败: %v (path: %s)", err, NewFile)
				}
			}else{
				return "",fmt.Errorf("目标文件是一个目录: %s", NewFile)
			}
		}else{
			if err := os.Remove(NewFile); err != nil {
				return "",fmt.Errorf("删除目标文件失败: %v (path: %s)", err, NewFile)
			}
		}
	}
	if err := os.Rename(saveFile, NewFile); err != nil {
		return "", err
	}
	if err := f.Chmod(NewFile, 777); err != nil {
		return "", err
	}
	return NewFile, nil
}

// DelDir 删除目录及其所有内容
// 参数：
//   - path: 目录路径
//
// 返回：
//   - error: 错误信息
//
// 示例：
//
//	err := Lin.Fso.DelDir("/path/to/directory")
func (f *WsyFso) DelDir(path string) error {
	if !f.Exists(path) {
		return nil
	}
	if !f.IsDir(path) {
		return fmt.Errorf("not a directory: %s", path)
	}
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	return nil
}

// GetFileSize 获取文件大小
// 参数：
//   - path: 文件路径
//
// 返回：
//   - int64: 文件大小（字节）
//   - error: 错误信息
//
// 示例：
//
//	size, err := Lin.Fso.GetFileSize("/path/to/file")
func (f *WsyFso) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return 0, fmt.Errorf("cannot get size of directory: %s", path)
	}
	size := info.Size()
	return size, nil
}

// GetExtension 获取文件扩展名
// 参数：
//   - path: 文件路径
//
// 返回：
//   - string: 扩展名（带点）
//
// 示例：
//
//	ext := Lin.Fso.GetExtension("/path/to/file.txt")
func (f *WsyFso) GetExtension(path string) string {
	ext := filepath.Ext(path)
	return ext
}

// GetMD5 计算文件的MD5值
// 参数:
//   - value: 可选，文件路径。不提供时使用当前执行文件路径
//
// 返回:
//   - string: MD5值（32位十六进制）
//
// 示例:
//
//	md5Value := Lin.Fso.GetMD5()          // 计算当前执行文件的MD5
//	md5Value := Lin.Fso.GetMD5("file.txt") // 计算指定文件的MD5
func (f *WsyFso) GetMD5(value ...string) string {
	var IsBack, IsValue string
	if len(value) == 0 {
		exePath, exePathErr := os.Executable()
		if exePathErr == nil {
			IsValue = exePath
		} else {
			IsValue = os.Args[0]
		}
	} else {
		IsValue = value[0]
	}
	data, err := os.ReadFile(IsValue)
	if err == nil {
		hash := md5.Sum(data)
		IsBack = hex.EncodeToString(hash[:])
	} else {
		IsBack = ""
	}
	return IsBack
}

// 获取绝对文件路径或文件名
// GetPath("name")   wsydata
// GetPath("path")   /data/wsydata
// GetPath("/data/wsydata/wsydata","name")   wsydata
// GetPath("/data/wsydata/wsydata","path")   /data/wsydata

// GetPath 获取文件路径相关信息
// 参数：
//   - value[0]: "name"获取文件名，"path"获取路径，或指定文件路径
//   - value[1]: 可选，当value[0]为文件路径时，指定获取"name"或"path"
//
// 返回：
//   - string: 请求的路径信息
//   - error: 错误信息
//
// 示例：
//
//	name, _ := GetPath("name")  // 获取当前程序文件名
//	path, _ := GetPath("path")  // 获取当前程序路径
//	name, _ := GetPath("/usr/bin/app", "name")  // 获取指定文件名
func (f *WsyFso) GetPath(value ...string) string {
	var IsValue, EXEPath, IsName, IsDirs string
	if len(value) == 0 {
		exePath, exePathErr := os.Executable()
		if exePathErr == nil {
			EXEPath = exePath
		} else {
			EXEPath = os.Args[0]
		}
		return EXEPath
	} else {
		if value[0] == "" {
			Logs("ERROR", "FSO", "value is null")
			return ""
		}
		IsValue = value[0]
	}
	if IsValue == "name" || IsValue == "path" {
		exePath, exePathErr := os.Executable()
		if exePathErr == nil {
			EXEPath = exePath
		} else {
			EXEPath = os.Args[0]
		}
	} else {
		if len(value) == 2 {
			EXEPath = value[0]
			IsValue = value[1]
			if IsValue == "" {
				Logs("ERROR", "FSO", "value2 is null")
				return ""
			}
		} else {
			Logs("ERROR", "FSO", "value2 is null")
			return ""
		}
	}
	if !strings.Contains(EXEPath, "/") {
		//Logs("ERROR", "path no find")
		return ""
	}
	parts := strings.Split(EXEPath, "/")
	if len(parts) > 2 {
		Islens := parts[:len(parts)-1]
		IsDirs = strings.Join(Islens, "/") //计算出目录
		IsName = parts[len(parts)-1]       //计算出文件名
	} else {
		IsDirs = "/"
		IsName = parts[len(parts)-1]
	}
	if IsValue == "name" {
		return IsName
	} else {
		return IsDirs
	}
}

// DownMakeFile 准备下载文件的路径
// 参数：
//   - path: 文件路径
//
// 返回：
//   - error: 错误信息
//
// 示例：
//
//	err := Lin.Fso.DownMakeFile("/path/to/download/file")
func (f *WsyFso) DownMakeFile(path string) error {
	// 参数验证
	if path == "" {
		Logs("ERROR", "FSO", "文件路径不能为空")
		return fmt.Errorf("文件路径不能为空")
	}
	// 获取目录路径
	dir := filepath.Dir(path)
	// 检查目录是否存在，如果不存在则创建
	if !f.Exists(dir) {
		Logs("INFO", "FSO", "目录不存在，将创建: %s", dir)
		if err := os.MkdirAll(dir, 0777); err != nil {
			return fmt.Errorf("创建目录失败: %v (path: %s)", err, dir)
		}
		Logs("INFO", "FSO", "已创建目录: %s", dir)
	}
	// 检查目标路径是否为目录，如果是则删除
	if f.Exists(path) {
		fileInfo, err := os.Stat(path)
		if err != nil {
			Logs("ERROR", "FSO", "获取文件信息失败: %v (path: %s)", err, path)
			return fmt.Errorf("获取文件信息失败: %v (path: %s)", err, path)
		}

		if fileInfo.IsDir() {
			Logs("WARN", "FSO", "目标路径是一个目录，将删除: %s", path)
			if err := os.RemoveAll(path); err != nil {
				Logs("ERROR", "FSO", "删除目录失败: %v", err)
				return fmt.Errorf("删除目录失败: %v (path: %s)", err, path)
			}
			Logs("INFO", "FSO", "已删除目录: %s", path)
		} else {
			// 如果是文件，先删除以确保可以写入新文件
			Logs("INFO", "FSO", "目标文件已存在，将覆盖: %s", path)
			if err := os.Remove(path); err != nil {
				Logs("ERROR", "FSO", "删除现有文件失败: %v", err)
				return fmt.Errorf("删除现有文件失败: %v (path: %s)", err, path)
			}
		}
	}
	return nil
}

// DownMake 准备下载文件的路径
func (f *WsyFso) DownMake(path string) error {
	// 参数验证
	if path == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	dir := filepath.Dir(path)
	// 检查目录是否存在，如果不存在则创建
	if !f.Exists(dir) {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return fmt.Errorf("创建目录失败: %v (path: %s)", err, dir)
		}
		Logs("INFO", "FSO", "已创建目录: %s", dir)
	}
	// 检查path是否存在
	if f.Exists(path) {
		fileInfo, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("获取文件信息失败: %v (path: %s)", err, path)
		}

		if fileInfo.IsDir() {
			Logs("WARN", "FSO", "目标路径是一个目录，将删除: %s", path)
			if err := os.RemoveAll(path); err != nil {
				Logs("ERROR", "FSO", "删除目录失败: %v", err)
				return fmt.Errorf("删除目录失败: %v (path: %s)", err, path)
			}
			Logs("INFO", "FSO", "已删除目录: %s", path)
		} else {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("删除现有文件失败: %v (path: %s)", err, path)
			}
		}
	}
	return nil
}

// ReadIniAll 读取整个 ini，转为 map[节名]map[键]值；无节名的键在 map[""] 下
//
// 示例：
//
//	m := Fso.ReadIniAll("/opt/.wsydev/.devsave")
//	// m["nullName"]["country"] == "中国"
//	// m["online"]["text"] == "..."
func (f *WsyFso) ReadIniAll(iniPath string, args ...string) string {
	path := f.AbsPath(iniPath)
	if !f.IsFile(path) {
		return ""
	}
	loadIni, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:             false,
		SkipUnrecognizableLines: true,
	}, path)
	if err != nil {
		return ""
	}
	out := make(map[string]map[string]string)
	for _, sec := range loadIni.Sections() {
		secName := sec.Name()
		if secName == ini.DEFAULT_SECTION {
			secName = "nullName"
		}
		kv := make(map[string]string)
		for _, k := range sec.Keys() {
			kv[k.Name()] = k.String()
		}
		if len(kv) > 0 {
			out[secName] = kv
		}
	}
	return Map.ToJson(out)
}

// ReadIni
//
// 示例：
// Fso.ReadIni("/wsysoft/wsyos.ini", "Config") // 读取整个 section，返回 JSON 字符串
// Fso.ReadIni("/wsysoft/wsyos.ini", "Config", "WebPort") // 读取 section 下指定 key
// Fso.ReadIni("/wsysoft/wsyos.ini", "webport") // 读取默认节下的 key
// Fso.ReadIni("/wsysoft/wsyos.ini", "[Config]") // 读取整个 section（不推荐，建议直接用 section 名）
//
// 说明：section 和 key 查找都不区分大小写，返回的 key 保留 ini 文件原始大小写。
// 注意:如果是无等号键，返回的是true
func (f *WsyFso) ReadIni(iniPath, sectionOrKey string, key ...string) string {
	NewIniPath := f.AbsPath(iniPath)
	if !f.IsFile(NewIniPath) { return "" }
	LoadIni, LoadErr := ini.LoadSources(ini.LoadOptions{
        Insensitive:      false, // 保持大小写
        //AllowBooleanKeys: false,  // 允许无等号键，如 "aa"
		SkipUnrecognizableLines: true, //跳过不可识别的行
    }, NewIniPath)
	if LoadErr != nil {
		return ""
	}
	if len(key) == 0 {
		for _, k := range LoadIni.Section("").Keys() {
			if strings.EqualFold(k.Name(), sectionOrKey) {
				return k.String()
			}
		}
		for _, s := range LoadIni.Sections() {
			if strings.EqualFold(s.Name(), sectionOrKey) {
				mText := map[string]string{}
				for _, k := range s.Keys() {
					mText[k.Name()] = k.String()
				}
				bText, _ := json.Marshal(mText)
				return string(bText)
			}
		}
		return ""
	}
	
	for _, s := range LoadIni.Sections() {
		if strings.EqualFold(s.Name(), sectionOrKey) {
			for _, k := range s.Keys() {
				if strings.EqualFold(k.Name(), key[0]) {
					return k.String()
				}
			}
			break
		}
	}
	return ""
}

// WriteIni
//
// 示例：
// Fso.WriteIni("/wsysoft/wsyos.ini", map[string]map[string]string{
//     "Database": {"DataTexT1": "12345", "boxid": "mac123"},
//     "Config": {"WebPort": "1243"},
// })
// Fso.WriteIni("/wsysoft/wsyos.ini", map[string]string{"Config1": "val1", "OtherKey": "val2"})
// Fso.WriteIni("/wsysoft/wsyos.ini", "Database", map[string]string{"DataTexT1": "12345", "boxid": "mac123"})
// Fso.WriteIni("/wsysoft/wsyos.ini", "Config1", "val1")
// Fso.WriteIni("/wsysoft/wsyos.ini", "Database", "DataTexT1", "12345")
func (f *WsyFso) WriteIni(iniPath string, args ...interface{}) error {
	var LoadIni *ini.File
	var LoadErr error
	NewIniPath := f.AbsPath(iniPath)
	if !f.IsFile(NewIniPath) {
		f.MakeDirFile(NewIniPath) //创建目录
		LoadIni = ini.Empty()
	}else{
		LoadIni, LoadErr = ini.LoadSources(ini.LoadOptions{
			Insensitive:      false, // 保持大小写
			AllowBooleanKeys: true,  // 允许无等号键，如 "aa"
		}, NewIniPath)
		if LoadErr != nil {
			return fmt.Errorf("配置文件语法错误: %v", LoadErr)
		}
	}
	findSection := func(name string) *ini.Section {
		for _, s := range LoadIni.Sections() { if strings.EqualFold(s.Name(), name) { return s } }
		return LoadIni.Section(name)
	}
	switch len(args) {
		case 1:
			switch v := args[0].(type) {
			case map[string]map[string]string: // 方法一
				for secName, keyValues := range v {
					sec := findSection(secName)
                    for k, val := range keyValues {
                        if k == "" { continue }
						// 删除同名(不区分大小写)的已有键，避免大小写重复
                        for _, ek := range sec.Keys() {
                            if strings.EqualFold(ek.Name(), k) {
                                sec.DeleteKey(ek.Name())
                            }
                        }
						// 空值则仅删除旧键并跳过写入
						if val == "" { continue }
						sec.Key(k).SetValue(val)
					}
				}
				return LoadIni.SaveTo(NewIniPath)
			case map[string]string: // 方法二，写入默认节
				sec := LoadIni.Section("")
                for k, val := range v {
                    if k == "" { continue }
                    for _, ek := range sec.Keys() {
                        if strings.EqualFold(ek.Name(), k) {
                            sec.DeleteKey(ek.Name())
                        }
                    }
					if val == "" { continue }
					sec.Key(k).SetValue(val)
				}
				return LoadIni.SaveTo(NewIniPath)
			}
		case 2:
			secName, ok := args[0].(string)
			if !ok { break }
			switch v := args[1].(type) {
			case map[string]string: // 方法二，指定 section
				sec := findSection(secName)
                for k, val := range v {
                    if k == "" { continue }
                    for _, ek := range sec.Keys() {
                        if strings.EqualFold(ek.Name(), k) {
                            sec.DeleteKey(ek.Name())
                        }
                    }
					if val == "" { continue }
					sec.Key(k).SetValue(val)
				}
				return LoadIni.SaveTo(NewIniPath)
			case string: // 方法四，默认节单条写入
				sec := LoadIni.Section("")
                for _, ek := range sec.Keys() {
                    if strings.EqualFold(ek.Name(), secName) {
                        sec.DeleteKey(ek.Name())
                    }
                }
				if v == "" { return LoadIni.SaveTo(NewIniPath) }
				sec.Key(secName).SetValue(v)
				return LoadIni.SaveTo(NewIniPath)
			}
		case 3:
			secName, ok1 := args[0].(string)
			key, ok2 := args[1].(string)
			val, ok3 := args[2].(string)
			if ok1 && ok2 && ok3 && secName != "" && key != "" {
				sec := findSection(secName)
                for _, ek := range sec.Keys() {
                    if strings.EqualFold(ek.Name(), key) {
                        sec.DeleteKey(ek.Name())
                    }
                }
				if val == "" { return LoadIni.SaveTo(NewIniPath) }
				sec.Key(key).SetValue(val)
				return LoadIni.SaveTo(NewIniPath)
			}
		}
	return fmt.Errorf("参数不合法")
}

// DelIni 从INI文件中删除指定的键或节
//
// 示例：
// Fso.DelIni("/wsysoft/wsyos.ini","DataTexT1,Boxid,path,name")
// Fso.DelIni("/wsysoft/wsyos.ini","Database","DataTexT1,Boxid,path,name")
// Fso.DelIni("/wsysoft/wsyos.ini","addd","path")
// Fso.DelIni("/wsysoft/wsyos.ini","addd","path,DataTexT1")
// Fso.DelIni("/wsysoft/wsyos.ini","[addd]")
func (f *WsyFso) DelIni(iniPath string, args ...string) error {
	NewIniPath := f.AbsPath(iniPath)
	if !f.IsFile(NewIniPath) {return fmt.Errorf("文件不存在: %s", NewIniPath)}
	LoadIni, LoadErr := ini.LoadSources(ini.LoadOptions{
        Insensitive:      false, // 保持大小写
        AllowBooleanKeys: true,  // 允许无等号键，如 "aa"
    }, NewIniPath)
	if LoadErr != nil { return fmt.Errorf("INI加载错误: %s", LoadErr) }
	// 删除整个 section
	if len(args) == 1 && strings.HasPrefix(args[0], "[") && strings.HasSuffix(args[0], "]") {
		sectionName := strings.Trim(args[0], "[]")
		for _, section := range LoadIni.Sections() {
			if strings.EqualFold(section.Name(), sectionName) { LoadIni.DeleteSection(section.Name()); return LoadIni.SaveTo(NewIniPath) }
		}
		return nil
	}
	// 删除默认节的 key（支持多个key，逗号分隔）
	if len(args) == 1 {
		keys := strings.Split(args[0], ",")
		sec := LoadIni.Section("")
		for _, k := range sec.Keys() {
			for _, delKey := range keys {
				if strings.EqualFold(k.Name(), strings.TrimSpace(delKey)) {
					sec.DeleteKey(k.Name())
				}
			}
		}
		return LoadIni.SaveTo(NewIniPath)
	}
	// 删除指定 section 下的 key（支持多个key，逗号分隔）
	if len(args) == 2 {
		sectionName := args[0]
		keys := strings.Split(args[1], ",")
		var sec *ini.Section
		for _, s := range LoadIni.Sections() {if strings.EqualFold(s.Name(), sectionName) {sec = s;break}}
		if sec == nil {return nil}
		for _, k := range sec.Keys() {
			for _, delKey := range keys {
				if strings.EqualFold(k.Name(), strings.TrimSpace(delKey)) {sec.DeleteKey(k.Name())}
			}
		}
		return LoadIni.SaveTo(NewIniPath)
	}
	return fmt.Errorf("参数不合法")
}


// ReadFile 读取文件内容
// 参数：
//   - path: 文件路径
//
// 返回：
//   - string: 文件内容
//   - error: 错误信息
//
// 示例：
//
//	content, err := Lin.Fso.ReadFile("/path/to/file")
func (f *WsyFso) ReadFile(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteFile 写入文件
// 参数：
//   - path: 文件路径
//   - content: 要写入的内容
//
// 返回：
//   - error: 错误信息
//
// 示例：
//
//	err := Lin.Fso.WriteFile("/path/to/file", "content")
func (f *WsyFso) WriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	if !f.Exists(dir) {
		if err := f.MakeDir(dir); err != nil {
			return err
		}
	}
	if f.Exists(path) && f.IsDir(path) {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("删除目录失败: %v", err)
		}
	}
	unixContent := strings.ReplaceAll(content, "\r\n", "\n")
	err := os.WriteFile(path, []byte(unixContent), 0755)
	if err != nil {
		return err
	}
	// 确保权限设置正确（考虑到 umask 的影响）
	if err := os.Chmod(path, 0777); err != nil {
		return err
	}
	Logs("INFO", "FSO", "成功写入文件 %s 并设置权限为 777", path)
	return nil
}

// WriterMulti 写入单个文件
// 参数：
//   - path: 文件路径（string）
//   - content: 要写入的内容
// 返回：
//   - error: 写入失败时返回错误，否则返回 nil
//
// 示例：
//   err := Lin.Fso.WriterMulti("/tmp/a.txt", "hello world")
func (f *WsyFso) WriterMulti(path string, content string) error {
	if path == "" { return Error("文件路径不能为空！")}
    dir := filepath.Dir(path)
    if !f.Exists(dir) {
        if err := f.MakeDir(dir); err != nil {
            return err
        }
    }
    if f.Exists(path) && f.IsDir(path) {
        if err := os.RemoveAll(path); err != nil {
            return err
        }
    }
    unixContent := strings.ReplaceAll(content, "\r\n", "\n")
    // 追加模式打开文件
    file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
    if err != nil {
        return err
    }
    defer file.Close()
    _, err = file.Write([]byte(unixContent))
    if err != nil {
        return err
    }
    // 确保权限设置正确
    if err := os.Chmod(path, 0777); err != nil {
        return err
    }
    return nil
}

// CleanSpaceEnter 清理字符串，去除多余的换行符和空白字符
// 参数：
//   - value: 要清理的字符串
//
// 返回：
//   - string: 清理后的字符串
//
// 示例：
//   - cleaned := Lin.Fso.CleanSpaceEnter("hello\nworld")  // 返回 "helloworld"
//   - nodeId := Lin.Fso.CleanSpaceEnter(Lin.Fso.ReadFile(nodePath))
func (f *WsyFso) CleanSpaceEnter(value string) string {
	value = f.CleanSpace(value)  //移除所有空白字符
	value = f.CleanEnter(value)  //移除所有换行符
	return value
}

func (f *WsyFso) CleanEnter(value string) string {
	value = strings.ReplaceAll(value, "\n", "")  //移除所有换行符
	value = strings.ReplaceAll(value, "\r", "")  //移除所有回车符
	return value
}
func (f *WsyFso) CleanSpace(value string) string {
	value = strings.TrimSpace(value)             //移除所有空白字符
	return value
}

