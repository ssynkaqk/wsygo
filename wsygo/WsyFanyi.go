package Wsy

import (
	"strings"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"time"
)

type WsyFanyi struct{}

var (
    AliYunKey    = "123456"
    AliYunSecret = "123456"
	AliYunWebUrl = "http://mt.cn-hangzhou.aliyuncs.com/api/translate/web/general"
)

func (f WsyFanyi) New(text, sourceLang, targetLang string, value ...string) string {
	respBody, err := f.AliYun(text, sourceLang, targetLang)
	if err != nil {
		return Json.Err(err.Error())
	}
	return Json.Ok("翻译成功", respBody)
}
//阿里云
func (f WsyFanyi) AliYun(text, sourceLang, targetLang string) (map[string]string, error) {
	if text == "" { return nil, Error("待翻译文本不能为空") }
	if sourceLang == "" || targetLang == "" { return nil, Error("翻译语言 / 目标语言不能为空") }
	body := Map.ToJson(map[string]string{
		"FormatType":     "text",
		"SourceLanguage": f.AliYunLang(sourceLang),
		"TargetLanguage": f.AliYunLang(targetLang),
		"SourceText":     text,
		"Scene":          "general",
	})
	u, err := url.Parse(AliYunWebUrl)
	if err != nil {
		return nil, Error("URL解析失败: %v", err)
	}
	path       := u.Path
	date       := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	nonce      := Str.Random("uuid")
	bodySum    := md5.Sum([]byte(body))
	bodyMD5    := base64.StdEncoding.EncodeToString(bodySum[:])
	authHeader := f.AliYunSign(bodyMD5, date, nonce, path, AliYunKey, AliYunSecret)
	Http.Header = map[string]string{
		"Accept":                 "application/json",
		"Content-Type":           "application/json",
		"Content-MD5":            bodyMD5,
		"Date":                   date,
		"Host":                   u.Hostname(),
		"Authorization":          authHeader,
		"x-acs-signature-nonce":  nonce,
		"x-acs-signature-method": "HMAC-SHA1",
		"x-acs-version":          "2019-01-02",
	}
	respBody, _ := Http.Post(AliYunWebUrl, body)
	if Json.Valid(respBody) {
		if Json.Get(respBody, "Code").String() != "200" {
			Message := Json.Get(respBody, "Message").String()
			return nil,Error("翻译失败：%s", Message)
		}else{
			Content := Json.Get(respBody, "Data.Translated").String()
			RelData := map[string]string{
				"text":    text,
				"source":  sourceLang,
				"content": Content,
				"target":  targetLang,
			}
			return RelData, nil
		}
	}
	return nil, Error("请求失败！")
}
// AliYunSign 签名
func (f WsyFanyi) AliYunSign(bodyMD5, date, nonce, path, accessKeyID, accessKeySecret string) string {
	stringToSign := strings.Join([]string{
		"POST",
		"application/json",
		bodyMD5,
		"application/json",
		date,
		"x-acs-signature-method:HMAC-SHA1",
		"x-acs-signature-nonce:" + nonce,
		"x-acs-version:2019-01-02",
		path,
	}, "\n")
	mac := hmac.New(sha1.New, []byte(accessKeySecret))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return "acs " + accessKeyID + ":" + signature
}

// 语言转换  zh-cn,zh_cn,zh  ->  zh
func (f WsyFanyi) AliYunLang(value string) string {
	MapData := map[string][]string{
		"zh":    {"zh_CN", "zh-CN", "zh-hans", "zh-hans-cn", "chs", "zhongwen", "中文", "简体"},
		"zh-tw": {"zh_TW", "zh-TW", "zh-hant", "zh-hant-tw", "cht", "繁体"},
		"en":    {"en_US", "en-US", "en_GB", "en-GB", "英文", "英语"},
		"ja":    {"ja_JP", "ja-JP", "日文", "日语"},
		"ko":    {"ko_KR", "ko-KR", "韩文", "韩语"},
		"fr":    {"fr_FR", "fr-FR", "法文", "法语"},
		"de":    {"de_DE", "de-DE", "德文", "德语"},
		"ru":    {"ru_RU", "ru-RU", "俄文", "俄语"},
		"th":    {"th_TH", "th-TH", "泰文", "泰语"},
		"hi":    {"hi_IN", "hi-IN", "印地文", "印地语"},
		"pt":    {"pt_BR", "pt-BR", "pt_PT", "pt-PT", "葡语", "葡萄牙语", "巴西语"},
		"vi":    {"vi_VN", "vi-VN", "越文", "越南语"},
		"es":    {"es_ES", "es-ES", "es_MX", "es-MX", "西文", "西班牙语", "墨西哥语"},
		"it":    {"it_IT", "it-IT", "意文", "意大利语"},
		"ar":    {"ar_SA", "ar-SA", "阿文", "阿拉伯语"},
		"tr":    {"tr_TR", "tr-TR", "土耳其语"},
    }
    for country, aliases := range MapData {
        if Str.IsSame(value, country) {
            return country
        }
        for _, a := range aliases {
            if Str.IsSame(value, a) {
                return country
            }
        }
    }
    return value
}