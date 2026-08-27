package Wsy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"reflect"
	"math"
	"math/rand"
	"time"
	"unicode"
	"github.com/google/uuid"
)

type WsyStr struct{}

// IIF 实现类似三元表达式的条件判断，返回字符串
func (s WsyStr) IIF(condition interface{}, trueValue string, falseValue string) string {
	// 判断条件是否为真
	isTrue := false
	if b, ok := condition.(bool); ok {
		isTrue = b
	} else if s, ok := condition.(string); ok {
		isTrue = s != "" && s != "0" && strings.ToLower(s) != "false"
	}

	if isTrue {
		return trueValue
	}
	return falseValue
}

// IIFx 实现类似三元表达式的条件判断，返回任意类型  (已用)
func (s WsyStr) IIFS(condition interface{}, trueValue interface{}, falseValue interface{}) interface{} {
	isTrue := false
	if b, ok := condition.(bool); ok {
		isTrue = b
	} else if str, ok := condition.(string); ok {
		isTrue = str != "" && str != "0" && strings.ToLower(str) != "false"
	}
	if isTrue {
		return trueValue
	}
	return falseValue
}

// Valid 验证字符串是否符合指定的类型
// 参数：
//   - value: 要验证的字符串
//   - typeCode: 类型代码（0-25）
//
// 返回：
//   - bool: 是否符合指定类型
//
// 类型代码说明：
//   - 0: 不验证
//   - 1: 数字和英文字母
//   - 2: 英文
//   - 3: 数字
//   - 4: 文本
//   - 5: 汉字
//   - 6: 日期 (YYYY-MM-DD)
//   - 7: 手机号
//   - 8: 邮箱
//   - 9: 价格
//   - 10: 中文
//   - 11: 汉字、数字和英文字母
//   - 12: 邮政编码
//   - 13: 时间格式 (YYYY-MM-DD HH:MM:SS)
//   - 14: 用户名
//   - 15: 整数
//   - 16: 浮点数
//   - 17: 邮政编码
//   - 18: QQ号
//   - 19: 电话号码
//   - 20: 手机号
//   - 21: URL
//   - 22: 域名
//   - 23: IP地址
//   - 24: 通用文本
//   - 25: MAC地址
//
// 示例：
//
//	isValid := str.Valid("abc123", 1)  // 返回 true
//	isValid = str.Valid("abc", 3)      // 返回 false
func (s WsyStr) Valid(value string, typeCode int) bool {
	// 转换类型码为字符串
	typeStr := fmt.Sprintf("%d", typeCode)

	// 类型0不验证
	if typeCode == 0 {
		return true
	}
	// 空值验证
	if value == "" {
		return false
	}
	// 正则表达式模式
	patterns := map[string]string{
		"1":  `^[a-zA-Z0-9]+$`,                                                                                        // 数字和英文字母
		"2":  `^[A-Za-z]+$`,                                                                                           // 英文
		"3":  `^[0-9]+$`,                                                                                              // 数字
		"4":  `^[\s\S]+$`,                                                                                             // 文本
		"5":  `^[\p{Han}]+$`,                                                                                          // 汉字
		"6":  `^\d{4}-\d{2}-\d{2}$`,                                                                                   // 日期
		"7":  `^(\+?\d{2,3})?0?1[3458]\d{9}$`,                                                                         // 手机号
		"8":  `^(\w+(?:[-+.]\w+)*)@((?:[\da-zA-Z][\da-zA-Z-]{0,61})?[\da-zA-Z]\.)+([a-zA-Z]{2,4}(?:\.[a-zA-Z]{2})?)$`, // 邮箱
		"9":  `^\d+(\.\d+)?$`,                                                                                         // 价格
		"10": `^[\p{Han}]+$`,                                                                                          // 中文
		"11": `^[a-zA-Z0-9\p{Han}]+$`,                                                                                 // 汉字、数字和英文字母
		"12": `^[1-9]\d{5}$`,                                                                                          // 邮政编码
		"13": `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`,                                                                 // 时间格式
		"14": `^[a-zA-Z]\w{2,19}$`,                                                                                    // 用户名
		"15": `^[-+]?\d+$`,                                                                                            // 整数
		"16": `^[-+]?\d+(\.\d+)?$`,                                                                                    // 浮点数
		"17": `^\d{6}$`,                                                                                               // 邮政编码
		"18": `^[1-9]\d{4,9}$`,                                                                                        // QQ号
		"19": `^((\(\+?\d{2,3}\))|(\+?\d{2,3}\-))?(\(0?\d{2,3}\)|0?\d{2,3}-)?[1-9]\d{4,7}(\-\d{1,4})?$`,               // 电话号码
		"20": `^(\+?\d{2,3})?0?1[3458]\d{9}$`,                                                                         // 手机号
		"21": `^(?:(https|http|ftp|rtsp|mms):\/\/)?[a-zA-Z0-9_\-\.]+(?::\d+)?(?:\/[a-zA-Z0-9_\-\.\/?%&=]*)?$`,         // URL
		"22": `^(([\da-zA-Z][\da-zA-Z-]{0,61})?[\da-zA-Z]\.)+([a-zA-Z]{2,4}(?:\.[a-zA-Z]{2})?)$`,                      // 域名
		"23": `^((25[0-5]|2[0-4]\d|(1\d|[1-9])?\d)\.){3}(25[0-5]|2[0-4]\d|(1\d|[1-9])?\d)$`,                           // IP地址
		"24": `^[\-\s@(_%,.)β\/:a-zA-Z0-9\p{Han}]+$`,                                                                  // 通用文本
		"25": `^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`,                                                                  // MAC地址
		"26": `^[a-zA-Z0-9,_-]+$`,                                                                                     // devxx,dev01,dev02，支持字母数字和,-_符号
		"27": `^https?://[a-zA-Z0-9.-]+(:[0-9]{1,5})?$`,                                                               // 以http://或https://开头  支持端口
		"28": `^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+(/\.?[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)*)?$`, // APP包名/组件，如 com.xxsoft.mintool 或 com.xxsoft.mintool/.MainActivity
		"29": `^https?://(([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}|((25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(25[0-5]|2[0-4]\d|1?\d?\d))(:[0-9]{1,5})?(/[a-zA-Z0-9_\-\.\/?%&=]*)?$`, // 完整URL：http(s)必填 + 域名/IP必填 + 端口可选 + 路径可选
		"30": `^(#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{4}|[0-9A-Fa-f]{6}|[0-9A-Fa-f]{8})|rgba?\(\s*([01]?\d{1,2}|2[0-4]\d|25[0-5])\s*,\s*([01]?\d{1,2}|2[0-4]\d|25[0-5])\s*,\s*([01]?\d{1,2}|2[0-4]\d|25[0-5])(\s*,\s*(0|1|0?\.\d+))?\s*\))$`, // 颜色：#RGB/#RGBA/#RRGGBB/#RRGGBBAA 或 rgb()/rgba()
	}
	// 获取对应的正则表达式
	pattern, exists := patterns[typeStr]
	if !exists {
		return false
	}
	// 编译正则表达式
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	// 执行验证
	return regex.MatchString(value)
}


// ToUpper 将字符串转换为大写
// 示例：
//
//	upperCase := str.ToUpper("hello")  // 返回 "HELLO"
func (s WsyStr) ToUpper(value string) string {
	return strings.ToUpper(value)
}

// ToLower 将字符串转换为小写
// 示例：
//
//	lowerCase := str.ToLower("HELLO")  // 返回 "hello"
func (s WsyStr) ToLower(value string) string {
	return strings.ToLower(value)
}



// Capitalize 将字符串的首字母转换为大写
// 示例：
//
//	str := LinStr{}
//	capitalized := str.Capitalize("hello")  // 返回 "Hello"
func (s WsyStr) Capitalize(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

// Trim 去除字符串两端的空白字符
// 示例：
//
//	str := LinStr{}
//	trimmed := str.Trim("  hello  ")  // 返回 "hello"
func (s WsyStr) Trim(value string) string {
	return strings.TrimSpace(value)
}

// IsEmpty 检查值是否为空
//
// 示例：
//	isEmpty1 := str.IsNull(nil)               // 返回 true
//	isEmpty2 := str.IsNull("")                // 返回 true
//	isEmpty3 := str.IsNull([]string{})        // 返回 true
//	isEmpty4 := str.IsNull(map[string]int{})  // 返回 true
//	isEmpty5 := str.IsNull("hello")           // 返回 false
//	isEmpty6 := str.IsNull(0)                 // 返回 false
//	isEmpty7 := str.IsNull(false)             // 返回 false
func (s WsyStr) IsNull(object interface{}) bool {
	if object == nil {
		return true
	}
	switch v := object.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case []string:
		return len(v) == 0
	case []int:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	case map[string]string:
		return len(v) == 0
	case map[string]int:
		return len(v) == 0
	}
	// 使用反射处理其他类型
	rv := reflect.ValueOf(object)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() == 0
	case reflect.String:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return true
		}
		return s.IsNull(rv.Elem().Interface())
	case reflect.Struct:
		return false
	default:
		return false
	}
}

// IsSame 检查两个字符串是否相同,忽略大小写
//
// 示例：
//
//	isSame := str.IsSame("hello", "HELLO")  // 返回 true
func (s WsyStr) IsSame(str1 string, str2 string) bool {
	return strings.EqualFold(str1, str2)
}

// IsEqual 比较两个字符串是否完全一致（区分大小写）
//
// 示例：
//
//	isEqual := str.IsEqual("hello", "hello")  // 返回 true
//	isEqual := str.IsEqual("hello", "Hello")  // 返回 false
func (s WsyStr) IsEqual(str1 string, str2 string) bool {
	return str1 == str2
}

// IsIn 检查字符串是否包含子字符串,忽略大小写
//
// 示例：
//	contains := str.Contains("hello world", "world")  // 返回 true
func (s WsyStr) IsIn(value string, substr string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(substr))
}



// IsHasFix 检查字符串的开头/结尾是否与另一个字符串匹配。
// - 默认从左（前缀）：等价 strings.HasPrefix
// - 方向传入 "R"：等价 strings.HasSuffix
// - 默认不区分大小写；如果额外传入 bool 且为 true，则改为区分大小写
//
// 示例：
//   Str.IsHasFix("abcdefg","ab") -> true
//   Str.IsHasFix("abcdefg","bc") -> false
//   Str.IsHasFix("abcdefg","fg","R") -> true
//   Str.IsHasFix("abcdefg","bc","R") -> false
//   Str.IsHasFix("Abcdefg","ab") -> true
//   Str.IsHasFix("Abcdefg","ab", true) -> false  (区分大小写)
func (s WsyStr) IsHasFix(value, match string, args ...interface{}) bool {
	if value == "" || match == "" { return false }
	dir := "L"
	ignoreCase := true
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if v == "R" { dir = "R" }
		case bool:
			if v == true { ignoreCase = false }
		}
	}
	v := value
	m := match
	if ignoreCase {
		v = strings.ToLower(v)
		m = strings.ToLower(m)
	}
	if dir == "R" { return strings.HasSuffix(v, m) }
	return strings.HasPrefix(v, m)
}
// ToHasFix 存在就删除匹配到的前缀/后缀。
// - 默认删除前缀（L）
// - 方向传入 "R" 删除后缀
// - 默认不区分大小写；如果额外传入 bool 且为 true，则改为区分大小写
//
// 示例：
//   Str.ToHasFix("abcdefg","ab") -> "cdefg"
//   Str.ToHasFix("abcdefg","fg","R") -> "abcde"
//   Str.ToHasFix("Abcdefg","ab") -> "cdefg"
//   Str.ToHasFix("Abcdefg","ab", true) -> "Abcdefg" (区分大小写，不删除)
func (s WsyStr) ToHasFix(value, match string, args ...interface{}) string {
	if value == "" || match == "" { return value }
	dir := "L"
	ignoreCase := true
	for _, a := range args {
		switch v := a.(type) {
		case string:
			if v == "R" { dir = "R" }
		case bool:
			if v == true { ignoreCase = false }
		}
	}
	vr := []rune(value)
	mr := []rune(match)
	if len(mr) == 0 || len(mr) > len(vr) { return value }
	vl := value
	ml := match
	if ignoreCase {
		vl = strings.ToLower(vl)
		ml = strings.ToLower(ml)
	}
	if dir == "R" {
		if !strings.HasSuffix(vl, ml) { return value }
		return string(vr[:len(vr)-len(mr)])
	}
	if !strings.HasPrefix(vl, ml) { return value }
	return string(vr[len(mr):])
}
// Cut 按字符(rune)长度截取字符串，避免中文截断乱码。
// - n 固定为 int（第二个参数）
// - 可选参数顺序不限：
//   - bool：方向（默认左；true 表示从右截取）
//   - string：当原字符串被截断时追加的后缀 suffix（取最后一个 string）
//
// 示例：
//   Str.Cut("fcdata", 2)                 -> "fc"
//   Str.Cut("fcdata", 2, true)           -> "ta"
//   Str.Cut("这是一个测试", 4)        -> "这是一个"
//   Str.Cut("这是一个测试", 4, "~")   -> "这是一个~"
//   Str.Cut("这是一个测试", 4, true, "...") -> "这是一个测试..."
//   Str.Cut("这是一个测试", 4, "...", true) -> "这是一个测试..."
func (s WsyStr) Cut(value string, n int, args ...interface{}) string {
	if value == "" || n <= 0 { return "" }
	rs := []rune(value)
	if n >= len(rs) { return value }
	right, suf := false, ""
	for _, a := range args {
		if v, ok := a.(bool); ok { right = v; continue }
		if v, ok := a.(string); ok { suf = v }
	}
	out := string(rs[:n])
	if right { out = string(rs[len(rs)-n:]) }
	if suf != "" { out += suf }
	return out
}
// GetPart 按分隔符获取第 N 段（从 1 开始），支持从左或从右开始数。
// - 默认从左（false）
// - 可选参数顺序不限：
//   - int：第几段（从 1 开始，默认 1）
//   - bool：方向（true 表示从右开始数）
//
// 示例：
//   Str.GetPart("acb#de", "#")      -> "acb"
//   Str.GetPart("acb#de", "#", 2)   -> "de"
//   Str.GetPart("a_b_c_d_e", "_")    -> "a"
//   Str.GetPart("a_b_c_d_e", "_", 2) -> "b"
//   Str.GetPart("a_b_c_d_e", "_", true) -> "e"
//   Str.GetPart("a_b_c_d_e", "_", true, 2) -> "d"
func (s WsyStr) GetPart(value, sep string, args ...interface{}) string {
	right, idx := false, 1
	for _, a := range args {
		if v, ok := a.(int); ok && v > 0 {
			idx = v
			continue
		}
		if v, ok := a.(bool); ok { right = v }
	}
	if value == "" { return "" }
	if sep == "" {
		if idx == 1 { return value }
		return ""
	}
	parts := strings.Split(value, sep)
	if idx > len(parts) { return "" }
	if right { return parts[len(parts)-idx] }
	return parts[idx-1]
}

// Has 判断列表中是否包含指定值（不区分大小写）
// 支持：
//  1) 字符串列表（默认用 "," 分隔，可自定义分隔符）
//     Str.Has("cn,en", "cn") == true
//     Str.Has("cn|en", "cn", "|") == true
//  2) 字符串数组
//     Str.Has([]string{"cn","en"}, "cn") == true
//  3) interface{} 数组
//     Str.Has([]interface{}{"cn","en"}, "cn") == true
func (s WsyStr) Has(value interface{}, find string, sep ...string) bool {
	var parts []string
	separator := ","
	find = strings.TrimSpace(find)
	if find == "" || value == nil {
		return false
	}
	if len(sep) > 0 && strings.TrimSpace(sep[0]) != "" { separator = sep[0] }
	findLower := strings.ToLower(find)
	switch v := value.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" { return false }
		parts = strings.Split(v, separator)
	case []string:
		parts = v
	case []interface{}:
		parts = make([]string, 0, len(v))
		for _, it := range v { parts = append(parts, s.ToString(it)) }
	default:
		str := strings.TrimSpace(s.ToString(value))
		if str == "" { return false }
		parts = strings.Split(str, separator)
	}
	for _, p := range parts {
		if strings.ToLower(strings.TrimSpace(p)) == findLower { return true }
	}
	return false
}


// ReplaceLoc 使用正则表达式替换字符串中的占位符
// 参数：
//   - value: 原始字符串，包含占位符，例如 "{work.path}/tool" 或 "[work.path]/tool"
//   - newValue: 用于替换占位符的新值，例如 "/opt/.mofei"
//   - format: 可选参数，占位符的格式，可选值为 "[]"、"{}" 或 "all"
//     如果不提供，默认同时处理 {} 和 [] 两种格式
//
// 返回：
//   - string: 替换后的字符串
//
// 示例：
//
//	// 默认同时替换方括号和花括号格式的占位符
//	result1 := str.ReplaceLoc("{work.path}/[tool]", "/opt/.mofei")        // 返回 "/opt/.mofei/opt/.mofei"
//	// 只替换方括号格式的占位符
//	result2 := str.ReplaceLoc("[work.path]/tool", "/opt/.mofei", "[]")    // 返回 "/opt/.mofei/tool"
//	// 只替换花括号格式的占位符
//	result3 := str.ReplaceLoc("{work.path}/tool", "/opt/.mofei", "{}")    // 返回 "/opt/.mofei/tool"
//	// 明确指定同时替换两种格式
//	result4 := str.ReplaceLoc("{path}/[tool]", "/opt/.mofei", "all")      // 返回 "/opt/.mofei/opt/.mofei"
func (s WsyStr) ReplaceLoc(value string, newValue string, format ...string) string {
	result := value
	selectedFormat := "{}[]"
	if len(format) > 0 && (format[0] == "[]" || format[0] == "{}") {
		selectedFormat = format[0]
	}
	// 根据指定的格式处理
	if selectedFormat == "[]" || selectedFormat == "{}[]" {
		re := regexp.MustCompile(`\[[^\]]*\]`)
		result = re.ReplaceAllString(result, newValue)
	}
	if selectedFormat == "{}" || selectedFormat == "{}[]" {
		re := regexp.MustCompile(`\{[^}]*\}`)
		result = re.ReplaceAllString(result, newValue)
	}
	return result
}

// Replace 替换字符串中的子字符串
// 示例：
//	replaced := str.Replace("hello world", "world", "golang")  // 返回 "hello golang"
func (s WsyStr) Replace(value string, old string, new string) string {
	return strings.Replace(value, old, new, -1)
}
// ToReplace 过滤指定字符并处理重复字符
// 参数:
//
//	value: 要处理的字符串
//	charToFilter: 要过滤的字符，如":"
//	replaceWith: (可选)替换成的字符，默认为","
//
// 示例:
//
//	str := LinStr{}
//	// 将MAC地址中的":"去除并转为大写
//	macFormatted := str.ToUpper(str.ToReplace("00:1A:2B:3C:4D:5E", ":", ""))  // 返回 "001A2B3C4D5E"
//	// 将多个":"替换为"-"
//	formatted := str.ToReplace("00::1A::2B:::3C", ":", "-")  // 返回 "00-1A-2B-3C"
func (s WsyStr) ToReplace(value string, charToFilter string, replaceWith ...string) string {
	// 设置替换字符
	replace := ""
	if len(replaceWith) > 0 {
		// 明确传入了替换字符（即使是空字符串）
		replace = replaceWith[0]
	}

	// 如果替换字符为空字符串，直接移除目标字符
	if replace == "" {
		// 处理连续多个字符的情况
		regex := regexp.MustCompile(regexp.QuoteMeta(charToFilter) + "+")
		return regex.ReplaceAllString(value, "")
	}

	// 处理替换为其他字符的情况
	regex := regexp.MustCompile(regexp.QuoteMeta(charToFilter) + "+")
	return regex.ReplaceAllString(value, replace)
}
// ToSpace 去除字符串中的重复空格，保留单个空格分隔的单词
//
// 示例：
//	cleaned := str.ToSpace("hello   world  go   lang")  // 返回 "hello world go lang"
//	cleaned = str.ToSpace("  multiple   spaces   ")     // 返回 "multiple spaces"
func (s WsyStr) ToSpace(value string) string {
	if value == "" {
		return ""
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, " ")
	var validParts []string
	for _, part := range parts {
		if part != "" {
			validParts = append(validParts, part)
		}
	}
	return strings.Join(validParts, " ")
}

// ToSpaceALL 删除字符串中的所有空白字符（包括空格、Tab、换行等）
//
// 示例：
//	cleaned := str.ToSpaceALL(" a\tb \n c ") // 返回 "abc"
//	cleaned = str.ToSpaceALL("hello world")  // 返回 "helloworld"
func (s WsyStr) ToSpaceALL(value string) string {
	if value == "" {
		return ""
	}
	// 删除所有空白字符：空格、制表符、换行等
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
}

// ToRepeat 过滤字符串中的重复值和空值，只保留唯一值
// 参数:
//   - ATitle: 要处理的字符串
//   - OVal: 分隔符 (可选，默认为 ",")
//
// 返回:
//   - string: 处理后的字符串，包含唯一非空值，并以分隔符结尾
//
// 示例:
//   str := LinStr{}
//   result := str.ToRepeat("a,b,c,a,,") // 返回 "a,b,c,"
//   result := str.ToRepeat("1|2|1||3", "|") // 返回 "1|2|3|"
func (s WsyStr) ToRepeat(ATitle string, OVal ...string) string {
	// 设置分隔符，默认为 ","
	separator := ","
	if len(OVal) > 0 && OVal[0] != "" {
		separator = OVal[0]
	}
	// 如果字符串为空，直接返回空字符串
	if ATitle == "" {
		return ""
	}
	// 分割字符串为数组
	items := strings.Split(ATitle, separator)
	// 使用map来记录已经出现过的唯一值
	seen := make(map[string]bool)
	var uniqueItems []string
	// 遍历分割后的元素
	for _, item := range items {
		// 去除元素两端的空白
		item = strings.TrimSpace(item)
		// 如果元素非空且未曾出现过
		if item != "" && !seen[item] {
			seen[item] = true
			uniqueItems = append(uniqueItems, item)
		}
	}
	result := strings.Join(uniqueItems, separator)
	return result
}

// Split 将字符串按分隔符分割为字符串数组
// 示例：
//	str := LinStr{}
//	parts := str.Split("a,b,c", ",")  // 返回 ["a", "b", "c"]
func (s WsyStr) Split(value string, sep string) []string {
	return strings.Split(value, sep)
}
// Contains 检查字符串是否包含子字符串
// 示例：
//	contains := str.Contains("hello world", "world")  // 返回 true
func (s WsyStr) Contains(value string, substr string) bool {
	return strings.Contains(value, substr)
}

// ToInt64 将任意类型转换为int64类型
// 
// 示例：
//   val1 := str.ToInt64("123")               // 返回 123
//   val2 := str.ToInt64("abc", 100)          // 返回 100 (默认值)
//   val3 := str.ToInt64("12.34")             // 返回 12
//   val4 := str.ToInt64(456)                  // 返回 456
//   val5 := str.ToInt64(int64(789))           // 返回 789
//   val6 := str.ToInt64(float64(123.99))      // 返回 123
//   val7 := str.ToInt64(nil, 999)             // 返回 999
//   var v interface{} = "321"; val8 := str.ToInt64(v) // 返回 321
//   var v2 interface{} = 654; val9 := str.ToInt64(v2)  // 返回 654
func (s WsyStr) ToInt64(value interface{}, defaultVal ...int64) int64 {
	defVal := int64(0)
	if len(defaultVal) > 0 {
		defVal = defaultVal[0]
	}
	if value == nil {
		return defVal
	}
	num, ok := s.ToNumberBase(value)
	if !ok {
		return defVal
	}
	return int64(num)
}

// ToInt 将任意类型转换为int类型
// 
// 示例：
//   val1 := str.ToInt("123")               // 返回 123
//   val2 := str.ToInt("abc", 100)          // 返回 100 (默认值)
//   val3 := str.ToInt("12.34")             // 返回 12
//   val4 := str.ToInt(456)                  // 返回 456
//   val5 := str.ToInt(int64(789))           // 返回 789
//   val6 := str.ToInt(float64(123.99))      // 返回 123
//   val7 := str.ToInt(nil, 999)             // 返回 999
//   var v interface{} = "321"; val8 := str.ToInt(v) // 返回 321
//   var v2 interface{} = 654; val9 := str.ToInt(v2)  // 返回 654
func (s WsyStr) ToInt(value interface{}, defaultVal ...int) int {
	defVal := 0
	if len(defaultVal) > 0 {
		defVal = defaultVal[0]
	}
	if value == nil {
		return defVal
	}
	num, ok := s.ToNumberBase(value)
	if !ok {
		return defVal
	}
	return int(num)
}

// ToFloat64 将任意类型转换为 float64 类型
// 
// 示例：
//   f1 := str.ToFloat64("123.456")         // 返回 123.46
//   f2 := str.ToFloat64("100")             // 返回 100.00
//   f3 := str.ToFloat64("abc")             // 返回 0
//   f4 := str.ToFloat64(123.456789, 4)      // 返回 123.4568
//   f5 := str.ToFloat64(100)                // 返回 100.00
//   f6 := str.ToFloat64(int64(200))         // 返回 200.00
//   f7 := str.ToFloat64(nil)                // 返回 0
//   var v interface{} = "321.99"; f8 := str.ToFloat64(v) // 返回 321.99
//   var v2 interface{} = 654; f9 := str.ToFloat64(v2)     // 返回 654.00
func (s WsyStr) ToFloat64(value interface{}, decimal ...int) float64 {
	if value == nil {
		return 0
	}
	dec := 2
	if len(decimal) > 0 {
		dec = decimal[0]
	}
	num, ok := s.ToNumberBase(value)
	if !ok {
		return 0
	}
	format := "%." + strconv.Itoa(dec) + "f"
	res, _ := strconv.ParseFloat(fmt.Sprintf(format, num), 64)
	return res
}

// ToBool 将任意类型转换为布尔值
// 
// 示例：
//   b1 := str.ToBool("true")               // 返回 true
//   b2 := str.ToBool("false")              // 返回 false
//   b3 := str.ToBool("1")                  // 返回 true
//   b4 := str.ToBool("0")                  // 返回 false
//   b5 := str.ToBool("yes")                // 返回 true
//   b6 := str.ToBool("no")                 // 返回 false
//   b7 := str.ToBool(true)                 // 返回 true
//   b8 := str.ToBool(false)                // 返回 false
//   b9 := str.ToBool(1)                    // 返回 true
//   b10 := str.ToBool(0)                   // 返回 false
//   b11 := str.ToBool("")                  // 返回 false
//   b12 := str.ToBool(nil)                 // 返回 false
func (s WsyStr) ToBool(value interface{}) bool {
	if value == nil {
		return false
	}
	
	switch v := value.(type) {
	case bool:
		return v
	case string:
		str := strings.ToLower(strings.TrimSpace(v))
		if str == "" {
			return false
		}
		// 真值
		if str == "true" || str == "1" || str == "yes" || str == "on" || str == "enabled" {
			return true
		}
		// 假值
		if str == "false" || str == "0" || str == "no" || str == "off" || str == "disabled" {
			return false
		}
		// 其他情况返回false
		return false
	case int, int8, int16, int32, int64:
		return v != 0
	case uint, uint8, uint16, uint32, uint64:
		return v != 0
	case float32, float64:
		return v != 0
	default:
		// 其他类型转换为字符串再判断
		str := s.ToString(v)
		return s.ToBool(str)
	}
}

// ToString 将任意类型安全转换为字符串
// 
// 示例：
//   s1 := str.ToString(123)           // "123"
//   s2 := str.ToString(12.34)         // "12.34"
//   s3 := str.ToString(true)          // "true"
//   s4 := str.ToString(nil)           // ""
//   s5 := str.ToString([]byte("abc")) // "abc"
//   s6 := str.ToString("hello")      // "hello"
func (S WsyStr) ToString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}


// 私有通用数值转换方法，所有ToInt/ToInt64/ToFloat64均调用 -不能删除
func (s WsyStr) ToNumberBase(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case int16:
		return float64(v), true
	case int8:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint8:
		return float64(v), true
	case string:
		str := strings.TrimSpace(v)
		if str == "" {
			return 0, false
		}
		num, err := strconv.ParseFloat(str, 64)
		if err != nil {
			// 尝试去掉小数点后再转
			if dotIndex := strings.Index(str, "."); dotIndex != -1 {
				str = str[:dotIndex]
				num, err = strconv.ParseFloat(str, 64)
				if err == nil {
					return num, true
				}
			}
			return 0, false
		}
		return num, true
	default:
		str := strings.TrimSpace(fmt.Sprintf("%v", v))
		if str == "" {
			return 0, false
		}
		num, err := strconv.ParseFloat(str, 64)
		if err != nil {
			if dotIndex := strings.Index(str, "."); dotIndex != -1 {
				str = str[:dotIndex]
				num, err = strconv.ParseFloat(str, 64)
				if err == nil {
					return num, true
				}
			}
			return 0, false
		}
		return num, true
	}
}

// Random 高级随机字符串/数字/uuid生成器，支持多参数
// 用法举例：
//   str.Random(6)                              // 6位混合字符串，如 "a1bC2d"
//   str.Random(6, true)                        // 6位大写混合字符串，如 "A1BC2D"
//   str.Random(6, "id-", true)                // 6位大写混合字符串，带前缀，如 "id-A2A3A6"
//   str.Random("sz", 4)                      // 4位数字，如 "1234"
//   str.Random("sz", 4, "id-")              // 4位数字，带前缀，如 "id-1234"
//   str.Random("sz", "10000", "99999", "id-") // 指定范围的数字，带前缀，如 "id-12345"
//   str.Random("en", 6, "%id-", true)         // 6位大写英文，带前缀，如 "id-ABCDEF"
//   str.Random("uuid")                        // 生成UUID字符串
func (s WsyStr) Random(args ...interface{}) string {
	typ := "mix"
	length := 6
	prefix := ""
	suffix := ""
	upper := false
	numMin, numMax := 0, 0
	commonHanzi := "的一是在不了有和人这中大为上个国我以要他时来用们生到作地于出就分对成会可主发年动同工也能下过子说产种面而方后多定行学法所民得经十三之进着等部度家电力里如水化高自二理起小物现实加量都两体制机当使点从业本去把性好应开它合还因由其些然前外天政四日那社义事平形相全表间样与关各重新线内数正心反你明看原又么利比或但质气第向道命此变条只没结解问意建月公无系军很情者最立代想已通并提直题党程展五果料象员革位入常文总次品式活设及管特件长求老头基资边流路级少图山统接知较将组见计别她手角期根论运农指几九区强放决西被干做必战先回则任取据处队南给色光门即保治北造百规热领七海口东导器压志世金增争济阶油思术极交受联什认六共权收证改清己美再采转单风切打白教速花带安场身车例真务具万每目至达走积示议声报斗完类八离华名确才科张信马节话米整空元况今集温传土许步群广石记需段研界拉林律叫且究观越织装影算低持音众书布复容儿须际商非验连断深难近矿千周委素技备半办青省列习便响约支般史感劳便团往酸历市克何除消构府称太准精值号率族维划选标写存候毛亲快效斯院查江型眼王按格养易置派层片始却专状育厂京识适属圆包火住调满县局照参红细引听该铁价严龙飞"
	baiJiaXing := []rune("赵钱孙李周吴郑王冯陈褚卫蒋沈韩杨朱秦尤许何吕施张孔曹严华金魏陶姜戚谢邹喻柏水窦章云苏潘葛奚范彭郎鲁韦昌马苗凤花方俞任袁柳酆鲍史唐费廉岑薛雷贺倪汤滕殷罗毕郝邬安常乐于时傅皮卞齐康伍余元卜顾孟平黄和穆萧尹姚邵湛汪祁毛禹狄米贝明臧计伏成戴谈宋茅庞熊纪舒屈项祝董梁杜阮蓝闵席季麻强贾路娄危江童颜郭梅盛林刁钟徐邱骆高夏蔡田樊胡凌霍虞万支柯昝管卢莫经房裘缪干解应宗丁宣贲邓郁单杭洪包诸左石崔吉钮龚程嵇邢滑裴陆荣翁荀羊於惠甄曲家封芮羿储靳汲邴糜松井段富巫乌焦巴弓牧隗山谷车侯宓蓬全郗班仰秋仲伊宫宁仇栾暴甘钭厉戎祖武符刘景詹束龙叶幸司韶郜黎蓟薄印宿白怀蒲台从鄂索咸籍赖卓蔺屠蒙池乔阴郁胥能苍双闻莘党翟谭贡劳逄姬申扶堵冉宰郦雍却璩桑桂濮牛寿通边扈燕冀郏浦尚农温别庄晏柴瞿阎充慕连茹习宦艾鱼容向古易慎戈廖庾终暨居衡步都耿满弘匡国文寇广禄阙东欧殳沃利蔚越夔隆师巩厍聂晁勾敖融冷訾辛阚那简饶空曾毋沙乜养鞠须丰巢关蒯相查后荆红游竺权逯盖益桓公仉督岳帅缑亢况郈有琴归海晋楚闫法汝鄢涂钦岳帅缑亢况郈有琴归海晋楚闫法汝鄢涂钦")
	// 参数解析
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			vLower := strings.ToLower(v)
			switch vLower {
			case "sz", "en", "zm", "cn", "uuid", "name":
				typ = vLower
			case "true", "false":
				// 跳过，bool类型会被后面case bool捕获
			default:
				if n, err := strconv.Atoi(v); err == nil {
					if numMin == 0 {
						numMin = n
					} else {
						numMax = n
					}
				} else {
					if strings.HasPrefix(v, "%") {
						prefix = v[1:]
					} else if strings.HasSuffix(v, "%") {
						suffix = v[:len(v)-1]
					} else {
						prefix = v
					}
				}
			}
		case int:
			if length == 6 {
				length = v
			} else if numMin == 0 {
				numMin = v
			} else {
				numMax = v
			}
		case bool:
			upper = v
		}
	}
	rand.Seed(time.Now().UnixNano())
	var body string
	switch typ {
	case "sz":
		if numMin != 0 || numMax != 0 {
			if numMax <= numMin {
				numMax = numMin + int(math.Pow10(length))-1
			}
			n := rand.Intn(numMax-numMin+1) + numMin
			body = strconv.Itoa(n)
		} else {
			for i := 0; i < length; i++ {
				body += strconv.Itoa(rand.Intn(10))
			}
		}
	case "en":
		letters := "abcdefghijklmnopqrstuvwxyz"
		for i := 0; i < length; i++ {
			body += string(letters[rand.Intn(len(letters))])
		}
	case "zm":
		letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		for i := 0; i < length; i++ {
			body += string(letters[rand.Intn(len(letters))])
		}
	case "cn":
		for i := 0; i < length; i++ {
			body += string([]rune(commonHanzi)[rand.Intn(len([]rune(commonHanzi)))])
		}
	case "uuid":
		body = uuid.New().String()
	case "name":
		xing := string(baiJiaXing[rand.Intn(len(baiJiaXing))])
		ming := ""
		if length > 0 {
			for i := 0; i < length; i++ {
				ming += string([]rune(commonHanzi)[rand.Intn(len([]rune(commonHanzi)))])
			}
		}
		body = xing + ming
	default: // mix
		chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for i := 0; i < length; i++ {
			body += string(chars[rand.Intn(len(chars))])
		}
	}

	// 大写处理
	if upper {
		if len(args) > 0 {
			if b, ok := args[len(args)-1].(bool); ok && b {
				body = strings.ToUpper(body)
			} else {
				body = strings.ToUpper(body)
				prefix = strings.ToUpper(prefix)
				suffix = strings.ToUpper(suffix)
			}
		} else {
			body = strings.ToUpper(body)
			prefix = strings.ToUpper(prefix)
			suffix = strings.ToUpper(suffix)
		}
	}

	return prefix + body + suffix
}
// ToArrString 将任意类型转换为字符串，支持数组类型自动拼接
// 可选参数 sep，默认逗号，可自定义分隔符
// 
// 示例：
//   str := str.ToArrString([]string{"a", "b", "c"}) // 返回 "a,b,c"
//   str = str.ToArrString([]string{"a", "b", "c"}, ":") // 返回 "a:b:c"
//   str = str.ToArrString([]interface{}{1, "b", 3.14}) // 返回 "1,b,3.14"
//   str = str.ToArrString([]int{1, 2, 3}) // 返回 "1,2,3"
//   str = str.ToArrString("hello") // 返回 "hello"
//   str = str.ToArrString(123) // 返回 "123"
//   str = str.ToArrString(nil) // 返回 ""
func (s WsyStr) ToArrString(value interface{}, sep ...string) string {
	separator := ","
	if len(sep) > 0 && sep[0] != "" {
		separator = sep[0]
	}
	
	if value == nil {
		return ""
	}
	
	switch v := value.(type) {
	case string:
		return v
	case []string:
		if len(v) == 0 {
			return ""
		}
		return strings.Join(v, separator)
	case []interface{}:
		if len(v) == 0 {
			return ""
		}
		var arr []string
		for _, item := range v {
			str := s.ToString(item)
			if !s.IsNull(str) {
				arr = append(arr, str)
			}
		}
		return strings.Join(arr, separator)
	case []int:
		if len(v) == 0 {
			return ""
		}
		var arr []string
		for _, item := range v {
			arr = append(arr, s.ToString(item))
		}
		return strings.Join(arr, separator)
	case []int64:
		if len(v) == 0 {
			return ""
		}
		var arr []string
		for _, item := range v {
			arr = append(arr, s.ToString(item))
		}
		return strings.Join(arr, separator)
	case []float64:
		if len(v) == 0 {
			return ""
		}
		var arr []string
		for _, item := range v {
			arr = append(arr, s.ToString(item))
		}
		return strings.Join(arr, separator)
	case []bool:
		if len(v) == 0 {
			return ""
		}
		var arr []string
		for _, item := range v {
			arr = append(arr, s.ToString(item))
		}
		return strings.Join(arr, separator)
	default:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			if rv.Len() == 0 {
				return ""
			}
			var arr []string
			for i := 0; i < rv.Len(); i++ {
				item := rv.Index(i).Interface()
				str := s.ToString(item)
				if !s.IsNull(str) {
					arr = append(arr, str)
				}
			}
			return strings.Join(arr, separator)
		}
		return s.ToString(value)
	}
}
// ToStrArray 将任意类型转换为字符串数组
// 可选参数 sep，默认逗号，可自定义分隔符
// 
// 示例：
//   strArray := str.ToStrArray("a,b,c") // 返回 ["a", "b", "c"]
//   strArray = str.ToStrArray("a:b:c", ":") // 返回 ["a", "b", "c"]
//   strArray = str.ToStrArray([]string{"a","b"}) // 返回 ["a","b"]
//   strArray = str.ToStrArray([]interface{}{1,"b"}) // 返回 ["1","b"]
//   strArray = str.ToStrArray(123)      // 返回 ["123"]
//   strArray = str.ToStrArray(nil)      // 返回 []string{}
func (s WsyStr) ToStrArray(value interface{}, sep ...string) []string {
	separator := ","
	if len(sep) > 0 && sep[0] != "" {
		separator = sep[0]
	}
	if value == nil {
		return []string{}
	}
	switch v := value.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return []string{}
		}
		return strings.Split(v, separator)
	case []string:
		return v
	case []interface{}:
		var arr []string
		for _, item := range v {
			arr = append(arr, s.ToString(item))
		}
		return arr
	default:
		// 其它类型转为字符串
		str := s.ToString(v)
		if str == "" {
			return []string{}
		}
		return []string{str}
	}
}

// ToMap 将字符串转换为map[string]interface{}类型，支持多种格式
// 
// 示例：
//   map1 := str.ToStrMap("name=张三&age=25")           // 返回 map[string]interface{}{"name":"张三","age":"25"}
//   map2 := str.ToStrMap("name:张三,age:25", ":", ",") // 返回 map[string]interface{}{"name":"张三","age":"25"}
//   map3 := str.ToStrMap("a=1&b=2&c=3")               // 返回 map[string]interface{}{"a":"1","b":"2","c":"3"}
//   map4 := str.ToStrMap("")                           // 返回 map[string]interface{}{}
func (s WsyStr) ToStrMap(value string, keyValueSep ...string) map[string]interface{} {
	result := make(map[string]interface{})
	kvSep := "="
	itSep := "&"
	if len(keyValueSep) > 0 && keyValueSep[0] != "" {
		kvSep = keyValueSep[0]
	}
	if len(keyValueSep) > 1 && keyValueSep[1] != "" {
		itSep = keyValueSep[1]
	}
	if strings.TrimSpace(value) == "" {
		return result
	}
	items := strings.Split(value, itSep)
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, kvSep, 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if key != "" {
				val = strings.TrimSpace(val)
				if val == "" {
					result[key] = ""
				} else if strings.ToLower(val) == "null" || strings.ToLower(val) == "nil" {
					result[key] = nil
				} else if strings.ToLower(val) == "true" {
					result[key] = true
				} else if strings.ToLower(val) == "false" {
					result[key] = false
				} else if num, err := strconv.Atoi(val); err == nil {
					result[key] = num
				} else if num, err := strconv.ParseFloat(val, 64); err == nil {
					result[key] = num
				} else {
					result[key] = val
				}
			}
		}
	}
	
	return result
}
// ToMapStr 将map转换为URL查询字符串格式
// 参数：
//   - data: 要转换的map数据
//   - keyValueSep: 键值分隔符，默认为"="
//   - itemSep: 项目分隔符，默认为"&"
//
// 返回：
//   - string: 转换后的查询字符串
//
// 示例：
//   data := map[string]interface{}{"boxid": "devb9e6026e4c7c8ff2", "page": 1}
//   result := str.ToMapStr(data)  // 返回 "boxid=devb9e6026e4c7c8ff2&page=1"
//   result = str.ToMapStr(data, ":", ",")  // 返回 "boxid:devb9e6026e4c7c8ff2,page:1"
func (s WsyStr) ToMapStr(data map[string]interface{}, keyValueSep ...string) string {
	if len(data) == 0 {
		return ""
	}
	kvSep := "="
	itemSep := "&"
	if len(keyValueSep) > 0 && keyValueSep[0] != "" {
		kvSep = keyValueSep[0]
	}
	if len(keyValueSep) > 1 && keyValueSep[1] != "" {
		itemSep = keyValueSep[1]
	}
	var parts []string
	for key, value := range data {
		if key != "" {
			valStr := s.ToString(value)
			parts = append(parts, key+kvSep+valStr)
		}
	}
	return strings.Join(parts, itemSep)
}

// UniqueArray 去除字符串数组中的重复项和空值，保持原有顺序
// 
// 示例：
//   unique := str.ToOnlyArray([]string{"a", "b", "a", "", "c", "b"}) // 返回 ["a", "b", "c"]
//   unique = str.ToOnlyArray("a,b,a,,c,b") // 返回 ["a", "b", "c"]
//   unique = str.ToOnlyArray([]interface{}{"a", "b", "a", nil, "c"}) // 返回 ["a", "b", "c"]
func (s WsyStr) ToOnlyArray(value interface{}) []string {
	var arr []string
	switch v := value.(type) {
	case string:
		arr = s.ToStrArray(v)
	case []string:
		arr = v
	case []interface{}:
		for _, item := range v {
			str := s.ToString(item)
			if !s.IsNull(str) {
				arr = append(arr, s.Trim(str))
			}
		}
	default:
		return []string{}
	}
	// 去重和去空
	uniq := make(map[string]struct{})
	var result []string
	for _, item := range arr {
		item = s.Trim(item)
		if !s.IsNull(item) {
			if _, exists := uniq[item]; !exists {
				uniq[item] = struct{}{}
				result = append(result, item)
			}
		}
	}
	
	return result
}
// ToDiffArray 集合运算，对两个字符串列表做差集/并集/交集
// - 分隔符默认 ","，可通过最后一个参数指定
// - 运算模式（第三个参数）：
//     "-" 差集：A 中有、B 中没有的元素
//     "+" 并集：A 和 B 所有元素（去重）
//     "=" 交集：A 和 B 共同拥有的元素
//
// 示例：
//   Str.ToDiffArray("1,2,3,4,5", "3,5")            -> "3,5"       (默认交集)
//   Str.ToDiffArray("1,2,3,4,5", "3,6")            -> "3"         (默认交集)
//   Str.ToDiffArray("1,2,3,4,5", "3,5", "-")      -> "1,2,4"
//   Str.ToDiffArray("1,2,3,4,5", "3,6", "-")      -> "1,2,4,5"
//   Str.ToDiffArray("1,2,3,4,5", "3,6", "+")      -> "1,2,3,4,5,6"
//   Str.ToDiffArray("1,2,3,4,5", "3,5", "=")      -> "3,5"
//   Str.ToDiffArray("1|2|3|4|5", "3|5", "-", "|") -> "1|2|4"
func (s WsyStr) ToDiffArray(a, b string, args ...string) string {
	mode := "="
	separator := ","
	if len(args) > 0 && args[0] != "" {
		mode = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		separator = args[1]
	}
	toSet := func(str string) map[string]bool {
		m := make(map[string]bool)
		for _, v := range strings.Split(str, separator) {
			if v = strings.TrimSpace(v); v != "" {
				m[v] = true
			}
		}
		return m
	}
	setA, setB := toSet(a), toSet(b)
	var result []string
	for v := range setA {
		if (mode == "-" && !setB[v]) || (mode == "=" && setB[v]) || mode == "+" {
			result = append(result, v)
		}
	}
	if mode == "+" {
		for v := range setB {
			if !setA[v] {
				result = append(result, v)
			}
		}
	}
	return strings.Join(result, separator)
}

// IsDiff 判断 A 是否包含 B 的全部元素（B 是 A 的子集）
// - 分隔符默认 ","，可通过最后一个参数指定
//
// 示例：
//   Str.IsDiffArray("1,2,3,4,5", "3,5")      -> true
//   Str.IsDiffArray("1,2,3,4,5", "5,3")      -> true
//   Str.IsDiffArray("1,2,3,4,5", "5,9")      -> false
//   Str.IsDiffArray("1,2,3,4,5", "9")        -> false
//   Str.IsDiffArray("1|2|3|4|5", "3|5", "|") -> true
func (s WsyStr) IsDiffArray(a, b string, sep ...string) bool {
	separator := ","
	if len(sep) > 0 && sep[0] != "" {
		separator = sep[0]
	}
	setA := make(map[string]bool)
	for _, v := range strings.Split(a, separator) {
		if v = strings.TrimSpace(v); v != "" {
			setA[v] = true
		}
	}
	for _, v := range strings.Split(b, separator) {
		if v = strings.TrimSpace(v); v != "" && !setA[v] {
			return false
		}
	}
	return true
}

// ToTalize 计算字符串按分隔符分割后的元素数量
// 参数:
//   - str: 要计算的字符串，如 "175,176,177,178,179,180,181,182,183"
//   - separator: 分隔符，可选，默认为逗号 ","
//
// 返回:
//   - int: 分割后的元素数量
//
// 示例:
//
//	count1 := Str.ToTalize("175,176,177,178,179,180,181,182,183")      // 返回 9
//	count2 := Str.ToTalize("175,176,177,178,179,180,181,182,183", ",") // 返回 9
//	count3 := Str.ToTalize("175")                                      // 返回 1
//	count4 := Str.ToTalize("a|b|c|d", "|")                             // 返回 4
//	count5 := Str.ToTalize("")                                         // 返回 0
//	count6 := Str.ToTalize(" ")                                        // 返回 0
//	count7 := Str.ToTalize(",,,")                                      // 返回 0
func (h WsyStr) ToTalize(str string, separator ...string) int {
	// 设置默认分隔符
	sep := ","
	if len(separator) > 0 && separator[0] != "" {
		sep = separator[0]
	}
	// 处理边缘情况：空字符串或只有空白
	str = strings.TrimSpace(str)
	if str == "" {
		return 0
	}
	// 1. 如果没有分隔符，只有空格或只有一个元素
	if !strings.Contains(str, sep) {
		if str == "" {
			return 0
		}
		return 1
	}
	// 2. 对于有多个分隔符的情况，使用正则表达式替换连续分隔符
	for strings.Contains(str, sep+sep) {
		str = strings.ReplaceAll(str, sep+sep, sep)
	}
	str = strings.Trim(str, sep)
	if str == "" {
		return 0
	}
	// 4. 计算分隔符出现次数 + 1 就是元素数量
	return strings.Count(str, sep) + 1
}

// ToSplitLine 将文本转换为单行字符串，用逗号分隔
// 参数:
//   - text: 要处理的文本
//   - removeRepeats: 是否移除重复值，默认为true
//
// 返回:
//   - string: 处理后的单行字符串，用逗号分隔
//
// 示例:
//   str := WsyStr{}
//   result := str.ToSplitLine("a\nb\nc\na")  // 返回 "a,b,c"
//   反向函数示例:
//   result := str.ToUnsplitLine("a,b,c")  // 返回 "a\nb\nc"
//   result = str.ToSplitLine("a b c c", false)  // 返回 "a,b,c,c"
func (s WsyStr) ToSplitLine(text string, removeRepeats ...bool) string {
	// 如果文本为空，直接返回
	if text == "" { return "" }
	// 默认移除重复值
	removeRepeat := true
	if len(removeRepeats) > 0 {
		removeRepeat = removeRepeats[0]
	}
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, "\n")
	lines := []string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	result := strings.Join(lines, ",")
	if removeRepeat {
		result = s.ToRepeat(result)
	}
	if result != "" && strings.HasSuffix(result, ",") {
		result = result[:len(result)-1]
	}
	return result
}
// ToLineBreak 将用逗号分隔的字符串转换为多行字符串
// 参数:
//   - text: 要处理的用逗号分隔的字符串
//
// 返回:
//   - string: 处理后的多行字符串，每行一个元素
//
// 示例:
//   str := WsyStr{}
//   result := str.ToLineBreak("a,b,c")  // 返回 "a\nb\nc"
func (s WsyStr) ToLineBreak(text string) string {
	// 如果文本为空，直接返回
	if text == "" { return "" }
	// 按逗号分割字符串
	items := strings.Split(text, ",")
	var lines []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			lines = append(lines, item)
		}
	}
	// 用换行符连接处理后的元素
	return strings.Join(lines, "\n")
}
//ToSplitValid
//ToSplitValid("",1) 检验，默认数字
// 返回的字符串已清理末尾的空白字符、逗号和其他不可见字符，符合shell标准
func (s WsyStr) ToSplitValid(value string,Valid ...int) string {
	var IsValid int
	if len(Valid) == 0 {
		IsValid = 3
	}else {
		IsValid = Valid[0]
	}
 	content := s.ToSplitLine(value)
 	items := strings.Split(content, ",")
 	var validNumbers []string
 	for _, item := range items {
 		item = strings.TrimSpace(item)
 		if item != "" && s.Valid(item, IsValid) {
 			validNumbers = append(validNumbers, item)
 		}
 	}
 	if len(validNumbers) > 0 {
 		content = strings.Join(validNumbers, ",")
 	} else {
 		content = ""
 	}
 	content = s.ToRepeat(content)
 	content = strings.TrimSpace(content)
 	return content
}


// ToSplitRemove 从分隔串 all 中剔除出现在 remove 里的项（按 sep 切段，逐项 TrimSpace 后精确匹配，保留顺序）。
//
// 参数:
//   - all: 待过滤列表，如 "1,2,3,4,5,6"
//   - remove: 要剔除的项，如 "2,5"；为空或字面量 "<nil>" 时不剔除
//   - sep: 分隔符，省略为 ","
//
// 返回:
//   - 过滤后的串，如 "1,3,4,6"；all 为空返回 ""
//
// 示例:
//
//	Wsy.Str.ToSplitRemove("1,2,3,4,5,6", "2,5")     // "1,3,4,6"
//	Wsy.Str.ToSplitRemove("a|b|c", "b", "|")         // "a|c"
func (s WsyStr) ToSplitRemove(all, remove string, sep ...string) string {
	d := ","
	if len(sep) > 0 && sep[0] != "" {
		d = sep[0]
	}
	if all == "" {
		return ""
	}
	ban := map[string]struct{}{}
	if remove != "" && remove != "<nil>" {
		for _, x := range strings.Split(remove, d) {
			if x = strings.TrimSpace(x); x != "" {
				ban[x] = struct{}{}
			}
		}
	}
	var o []string
	for _, x := range strings.Split(all, d) {
		if x = strings.TrimSpace(x); x == "" {
			continue
		}
		if _, hit := ban[x]; !hit {
			o = append(o, x)
		}
	}
	return strings.Join(o, d)
}
// DBCToSBC 将全角字符转换为半角字符
// 参数：
//   - value: 包含全角字符的字符串
//
// 返回：
//   - string: 转换后的半角字符串
//
// 示例：
//   result := str.DBCToSBC("Ｈｅｌｌｏ　Ｗｏｒｌｄ！１２３")  // 返回 "Hello World!123"
func (s WsyStr) DBCToSBC(value string) string {
	if value == "" {
		return ""
	}
	var result strings.Builder
	for _, char := range value {
		// 全角空格转半角空格
		if char == 12288 {
			result.WriteRune(32)
		} else if char >= 65281 && char <= 65374 {
			result.WriteRune(char - 65248)
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// ToBash 将客户端通过textarea编写的代码转换为可执行的bash脚本
// 处理全角字符、换行符、不可见字符等问题，确保生成的脚本可以正常执行
// 
// 参数：
//   - value: 原始文本内容
//
// 返回：
//   - string: 处理后的可执行bash脚本
//
// 示例：
//   script := str.ToBash("echo　hello\r\n# 注释\n  ls -la  ")
//   // 返回处理后的脚本，全角字符转半角，换行符统一，行尾空白去除
func (s WsyStr) ToBash(value string) string {
	if value == "" { return "" }
	value = s.DBCToSBC(value)
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(value, "\n")
	var result strings.Builder
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		var cleaned strings.Builder
		for _, r := range line {
			if (r >= 32 && r <= 126) || r == '\t' || r >= 0x80 {
				cleaned.WriteRune(r)
			}
		}
		line = cleaned.String()
		if i == len(lines)-1 && line == "" {
			continue
		}
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	return strings.TrimRight(result.String(), "\n")
}