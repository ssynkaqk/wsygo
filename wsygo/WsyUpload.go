package Wsy

import (
	"os"
	"strings"
	"path/filepath"
	"mime/multipart"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"
	_ "image/gif"
	"github.com/gin-gonic/gin"
)

// WsyUpload 文件上传工具
type WsyUpload struct {
	SiteHttp    string   // 站点HTTP地址  http://en.oss.wsyos.com
	SaveFolder  string   // 保存文件夹 /wsy/oss/upload
	SavePath    string   // 保存 Y/M/D
	SaveName    string   // 文件名，YYYYMMDDHHMMSS
	AcceptExts  string   // 允许的扩展名（以逗号分隔，如"jpg|png"，"*"表示全部）
	AcceptSize  string   // 最大文件大小（MB）
	Encryption  string   // 加密密钥链接
	ValidPixel  string   // 验证图片高宽 尺寸
}

func (u *WsyUpload) Config(value string) string {
	vdata := map[string]string{
		"SiteHttp"   : u.SiteHttp,
		"SavePath"   : Str.IIF(u.SavePath   != "", u.SavePath, "YM"),
		"SaveName"   : Str.IIF(u.SaveName   != "", u.SaveName, "YMDHISV"),
		"SaveFolder" : Str.IIF(u.SaveFolder == "" || u.SaveFolder == "/", "/wsy/oss/upload", Fso.IsPath(u.SaveFolder)),
		"AcceptExts" : Str.IIF(u.AcceptExts != "", u.AcceptExts, "jpg|png|gif|apk|zip|rar|7z|tar|gz|bz2|iso"),
		"AcceptSize" : Str.IIF(u.AcceptSize != "" && Str.Valid(u.AcceptSize, 3), u.AcceptSize, "30"),
		"Encryption" : Str.IIF(u.Encryption != "" && u.Encryption == "true", "true", "false"),
	}
	return vdata[value]
}

// Save 保存单个文件
func (u *WsyUpload) Save(ctx *gin.Context) (map[string]interface{}, error) {
	file, err := ctx.FormFile("file")
	if err != nil {
		return nil, Error("获取文件失败: " + err.Error())
	}
	if err := u.Valid(file); err != nil {
		return nil, err
	}
	// 获取完整的保存路径（包含文件名）
	fullPath := u.GetName(file.Filename)
	// 使用完整路径保存文件
	if err := ctx.SaveUploadedFile(file, fullPath); err != nil {
		return nil, Error("保存文件失败: " + err.Error())
	}
	//转换目录
	InPath, InUrl := u.BackFile(fullPath)
	//Ext
	FileExt := filepath.Ext(file.Filename)
	//返回数据
	RelBack := map[string]interface{}{
		"name": Date.ToTime(u.Config("SaveName")),
		"path": InPath,
		"size": file.Size,
		"type": u.GetPath(file.Filename),
		"exts": FileExt,
		"url" : InUrl,
	}
	//获取APK信息
	if Str.IsSame(FileExt,".apk") {
		appInfo, appErr := APP.ToXml(fullPath,false,true)
		if appErr != nil {
			return nil, appErr
		}
		if appErr == nil {
			RelBack["label"]    = appInfo["label"]
			RelBack["version"]  = appInfo["version"]
			RelBack["package"]  = appInfo["package"]
			RelBack["logoPath"] = ""
			RelBack["logo"]     = ""
			if appInfo["label"] != "" {
				RelBack["name"] = appInfo["label"]
			}
			if appInfo["image"] != "" {
				SavePng,SaveErr := u.SaveBase64ToPng(appInfo["image"],u.GetName("logo.png"))
				if SaveErr == nil && SavePng != "" {
					LogoPath, LogoUrl := u.BackFile(SavePng)
					RelBack["logo"]     = LogoUrl
					RelBack["logoPath"] = LogoPath
				}
			}
		}
	}
	//读取图片宽高
	if Str.IsSame(FileExt,".jpg") || Str.IsSame(FileExt,".jpeg") || Str.IsSame(FileExt,".png") || Str.IsSame(FileExt,".gif") {
		var width, height int
		var imgErr error
		if u.ValidPixel != "" {
			width, height, imgErr = u.IsPixel(fullPath, u.ValidPixel, true)
		} else {
			width, height, imgErr = u.IsPixel(fullPath)
		}
		if imgErr != nil {
			return nil, imgErr
		}
		RelBack["width"]  = width
		RelBack["height"] = height
	}
	return RelBack, nil
}

// IsPixel 获取图片宽高，可选验证是否与指定尺寸一致
// 用法:
//   IsPixel(filePath)                           - 返回宽高，不验证
//   IsPixel(filePath, "1920x1080")              - 返回宽高，验证是否匹配（x 分隔）
//   IsPixel(filePath, "1920|1080")              - 返回宽高，验证是否匹配（| 分隔）
//   IsPixel(filePath, 1920, 1080)               - 返回宽高，验证是否匹配
//   IsPixel(filePath, "1920", "1080")           - 返回宽高，验证是否匹配
//   IsPixel(filePath, "1920", "1080", false)    - 返回宽高，false 跳过验证
//   单参数只验证宽度:
//   IsPixel(filePath, 1920)                     - 只验证宽度=1920
//   IsPixel(filePath, "1920")                   - 只验证宽度=1920（string 转 int）
//   宽/高传 0 表示跳过该维度验证:
//   IsPixel(filePath, 1920, 0)                  - 只验证宽度=1920
//   IsPixel(filePath, 0, 1080)                  - 只验证高度=1080
//   IsPixel(filePath, "1920x0")                 - 只验证宽度=1920
func (u *WsyUpload) IsPixel(filePath string, args ...any) (int, int, error) {
	sep := ""
	f, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	imgW, imgH := cfg.Width, cfg.Height
	if len(args) == 0 {
		return imgW, imgH, nil
	}
	var targetW, targetH int
	gotW := false
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if Str.Valid(v, 3) {
				if !gotW {
					targetW, gotW = Str.ToInt(v), true
				} else {
					targetH = Str.ToInt(v)
				}
				continue
			}
			if strings.Contains(v, "x") {
				sep = "x"
			} else if strings.Contains(v, "|") {
				sep = "|"
			} else {
				return imgW, imgH, Error("无效像素参数")
			}
			w, h := Str.GetPart(v, sep), Str.GetPart(v, sep, 2)
			if !Str.Valid(w, 3) || !Str.Valid(h, 3) {
				return imgW, imgH, Error("无效像素参数")
			}
			targetW, targetH, gotW = Str.ToInt(w), Str.ToInt(h), true
		case int:
			if !gotW {
				targetW, gotW = v, true
			} else {
				targetH = v
			}
		case bool:
			if !v {
				return imgW, imgH, nil
			}
		}
	}
	if targetW > 0 && imgW != targetW {
		return imgW, imgH, Error("图片尺寸不符合要求")
	}
	if targetH > 0 && imgH != targetH {
		return imgW, imgH, Error("图片尺寸不符合要求")
	}
	return imgW, imgH, nil
}

// 获取文件分类
func (u *WsyUpload) BackFile(fullPath string) (string,string) {
	//转换URL目录
	NewPath := Fso.GetPath(u.Config("SaveFolder"),fullPath)
	if strings.HasPrefix(fullPath, NewPath) {
		fullPath = strings.TrimPrefix(fullPath, NewPath)
	}
	//拼接URL
	InUrl := u.Config("SiteHttp") + fullPath
	if u.Config("Encryption") == "true" {
		InUrl    = Key.EnCode(InUrl)
		fullPath = Key.EnCode(fullPath)
	}
	return fullPath, InUrl
}
// SaveBase64ToPng 将base64编码的图片数据保存为PNG格式图片文件
// 参数:
//   - base64Data: base64编码的图片数据（支持 data:image/png;base64, 格式或纯base64字符串）
//   - saveFile: 保存文件的路径（为空则自动生成文件名）
//
// 返回:
//   - 保存的文件路径，失败返回空字符串
func (u *WsyUpload) SaveBase64ToPng(base64Data string, saveFile string) (string, error) {
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

// 获取文件分类
func (u *WsyUpload) GetPath(value string) string {
	vdata := map[string]string{
		"image": "jpg|jpeg|png|gif|bmp|webp|svg|ico",
		"video": "mp4|avi|mkv|flv|swf|ogv|mov|wmv|webm|3gp",
		"audio": "mp3|wav|flac|aac|ogg|mid|wma",
		"zip":   "zip|rar|7z|tar|gz|bz2|iso",
		"file":  "pdf|doc|docx|txt|xls|xlsx|ppt|pptx|exe|msi|deb|rpm|dmg|pkg|psd",
		"apk":   "apk|ipa|aab",
	}
	ext := strings.ToLower(filepath.Ext(value))
	if ext != "" {
		ext = ext[1:]
		for category, exts := range vdata {
			extList := strings.Split(exts, "|")
			for _, e := range extList {
				if e == ext {
					return category
				}
			}
		}
	}
	return "file"
}
// GetName 合并getSavePath和generateName功能，返回完整的保存路径
// 同时负责创建必要的目录结构
func (u *WsyUpload) GetName(value string) string {
	savePath := Fso.AddPath(u.Config("SaveFolder"),u.GetPath(value),Date.ToTime(u.Config("SavePath")))
	os.MkdirAll(savePath, 0755)
	fullPath := Fso.AddPath(savePath, Date.ToTime(u.Config("SaveName")) + filepath.Ext(value))
	return fullPath
}
// 验证文件
func (u *WsyUpload) Valid(file *multipart.FileHeader) error {
	// 检查文件大小（转换MB为字节）
	maxSizeInBytes := int64(Str.ToInt64(u.Config("AcceptSize")) * 1024 * 1024)
	if file.Size > maxSizeInBytes {
		return Error("文件过大，最大允许 %dMB", Str.ToInt64(u.Config("AcceptSize")))
	}

	if u.Config("AcceptExts") != "*" {
		FileExts := strings.Split(u.Config("AcceptExts"), "|")
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != "" {
			ext = ext[1:]
			allowed := false
			for _, allowedExt := range FileExts {
				if allowedExt == ext {
					allowed = true
					break
				}
			}
			if !allowed {
				return Error("不支持的文件类型: .%s", ext)
			}
		}
	}
	return nil
}