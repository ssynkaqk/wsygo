package Wsy

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// WsyKey 定义一个空结构体
type WsyKey struct{}

// MD5 返回给定字符串的MD5哈希值
//
// 参数:
//   - value: 要计算哈希的字符串
//   - args: 可选，支持以下取值（顺序不敏感）：
//       - 16/32：指定输出长度（默认32）
//       - true/false：指定输出大小写（true => 小写，false => 大写）
//
// 示例:
//   - key.MD5("test") -> 32位小写
//   - key.MD5("test", false) -> 32位大写
//   - key.MD5("test", false, 32) -> 32位大写
//   - key.MD5("test", 32, false) -> 32位大写
//   - key.MD5("test", true) -> 32位小写
func (h WsyKey) MD5(value string, args ...interface{}) string {
	hash := md5.Sum([]byte(value))
	s := hex.EncodeToString(hash[:]) // 默认32位小写
	length, upper := 32, false        // false=小写 true=大写
	for _, a := range args {
		switch v := a.(type) {
		case bool:
			upper = !v // true=>小写(false大写), false=>大写(true大写)
		case int:
			if v == 16 || v == 32 {
				length = v
			}
		}
	}
	if length == 16 {
		s = s[8:24]
	}
	if upper {
		s = strings.ToUpper(s)
	}
	return s
}
// SHA256 返回给定字符串的SHA256哈希值
// 参数:
//
//  value: 要计算哈希的字符串
//  args: 可选（顺序不敏感），支持：
//       - bool：true => 小写，false => 大写
//
// 示例: key.SHA256("test"), key.SHA256("test",true), key.SHA256("test",false)
func (h WsyKey) SHA256(value string, args ...bool) string {
	hash := sha256.Sum256([]byte(value))
	result := hex.EncodeToString(hash[:])
	upper := false // 默认小写
	if len(args) > 0 {
		upper = !args[0] // true=>小写, false=>大写
	}
	if upper {
		result = strings.ToUpper(result)
	}
	return result
}
// EnCode 加密字符串并返回URL安全格式的结果
// 使用AES-256-CBC算法，并将Base64结果中的特殊字符(+,/,=)替换为URL安全字符(_,-,.)
// 参数:
//
//	value: 要加密的文本
//	key: 可选，加密密钥，不提供则使用默认密钥Config.CryptoKey
//
// 示例:
//
//	encrypted := Wsy.Key.EnCode("Hello World") // 使用默认密钥
//	encrypted := Wsy.Key.EnCode("Hello World", "custom_key") // 使用自定义密钥
func (h WsyKey) EnCode(value string, args ...string) string {
	key := Set.Key
	if len(args) > 0 && args[0] != "" {
		key = args[0]
	}
	hasher := md5.New()
	keyBytes := make([]byte, 32)
	hasher.Write([]byte(key))
	copy(keyBytes, hasher.Sum(nil))
	hasher.Reset()
	hasher.Write(keyBytes[0:16])
	copy(keyBytes[16:32], hasher.Sum(nil))
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return ""
	}
	blockSize := block.BlockSize()
	padding := blockSize - len(value)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	paddedText := append([]byte(value), padText...)
	iv := []byte("1234567890123456")
	mode := cipher.NewCBCEncrypter(block, iv)
	cipherText := make([]byte, len(paddedText))
	mode.CryptBlocks(cipherText, paddedText)
	encoded := base64.StdEncoding.EncodeToString(cipherText)
	urlSafe := strings.NewReplacer(
		"+", "_",
		"/", "-",
		"=", ".",
	).Replace(encoded)
	return urlSafe
}

// DeCode 解密URL安全格式的加密字符串
// 先将URL安全字符(_,-,.)替换回标准Base64字符(+,/,=)，然后进行解密
// 如果解密失败（包括密钥错误、数据损坏等情况），返回空字符串
// 参数:
//
//	encryptedText: 要解密的文本
//	key: 可选，解密密钥，不提供则使用默认密钥Config.CryptoKey
//
// 示例:
//
//	decrypted := Wsy.Key.DeCode(encrypted) // 使用默认密钥
//	decrypted := Wsy.Key.DeCode(encrypted, "custom_key") // 使用自定义密钥
func (h WsyKey) DeCode(encryptedText string, args ...string) string {
	// 处理可选的key参数
	key := Set.Key
	if len(args) > 0 && args[0] != "" {
		key = args[0]
	}
	if encryptedText == "" {
		return ""
	}
	// 使用defer和recover来捕获可能的panic
	defer func() {
		if r := recover(); r != nil {
			// 如果发生panic，返回空字符串
		}
	}()
	standardBase64 := strings.NewReplacer(
		"_", "+",
		"-", "/",
		".", "=",
	).Replace(encryptedText)
	cipherText, err := base64.StdEncoding.DecodeString(standardBase64)
	if err != nil {
		return ""
	}
	hasher := md5.New()
	keyBytes := make([]byte, 32)
	hasher.Write([]byte(key))
	copy(keyBytes, hasher.Sum(nil))
	hasher.Reset()
	hasher.Write(keyBytes[0:16])
	copy(keyBytes[16:32], hasher.Sum(nil))
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return ""
	}
	if len(cipherText) < block.BlockSize() {
		return ""
	}
	// 检查密文长度是否为块大小的倍数
	if len(cipherText)%block.BlockSize() != 0 {
		return ""
	}
	iv := []byte("1234567890123456")
	mode := cipher.NewCBCDecrypter(block, iv)
	paddedText := make([]byte, len(cipherText))
	mode.CryptBlocks(paddedText, cipherText)
	length := len(paddedText)
	if length == 0 {
		return ""
	}
	padding := int(paddedText[length-1])
	if padding <= 0 || padding > length || padding > 16 {
		return ""
	}
	for i := 0; i < padding; i++ {
		if paddedText[length-1-i] != byte(padding) {
			return ""
		}
	}
	value := paddedText[:length-padding]
	return string(value)
}

// Allow 检查域名是否在授权列表中
// 授权码格式：域名列表,过期时间（如：192.168.2.5|api.dev.wsyos.com|*.wsyos.com|*,2014-12-31）
// 匹配规则：* 全部放行，精确匹配，*.domain.com 匹配子域名
func (h WsyKey) Allow(Host, AuthCode string) bool {
    AuthDeCode := Key.DeCode(AuthCode)
    if AuthDeCode == "" {
        return false
    }
    parts := strings.Split(AuthDeCode, ",")
    if len(parts) != 2 || !Date.ToBothTime(Date.Now(), parts[1]) {
        return false
    }
    host := Host
    if idx := strings.Index(host, ":"); idx > 0 {
        host = host[:idx]
    }
    for _, allowed := range strings.Split(parts[0], "|") {
        allowed = strings.TrimSpace(allowed)
        if allowed == "" {
            continue
        }
        if allowed == "*" || allowed == host {
            return true
        }
        if strings.HasPrefix(allowed, "*.") {
            suffix := allowed[2:]
            if strings.HasSuffix(host, "."+suffix) || host == suffix {
                return true
            }
        }
    }
    return false
}