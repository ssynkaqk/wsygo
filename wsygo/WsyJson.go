package Wsy

import (
	"fmt"
	"time"
	"strings"
	"encoding/json"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/tidwall/pretty"
)

type WsyJson struct {
	data string  //Add方法
}

// ===================== 以下是链式 JSON 构建器 =====================
// 用法：
//   jv := WsyJson{}.New()
//   jv.Add("code", 1).Add("msg", "ok")
//   str := jv.String()   // 得到 JSON 字符串
//   n := jv.Get("code").Int()    // 得到整数
//   b := jv.Get("success").Bool() // 得到布尔值

// New: 创建链式 JSON 构建器
func (WsyJson) New() *WsyJson {
	return &WsyJson{data: "{}"}
}
/*

    mAAA := Wsy.NewMap{
		"code": "mAAA",
		"msg":  "mAAA",
	}
    mAAAAA := Wsy.NewMap{
		"code": "mAAAAA",
		"msg":  "mAAAAA",
	}

    jv := Wsy.Json.New()
    jv.Add("code", 1)
    jv.Add("msg", mAAA)
    jv.Add("data.id", mAAAAA)

    jv2 := Wsy.Json.New()
    jv2.Add("code", 1)
    jv2.Add("msg", "操作成功")
    jv2.Add("data.id", 123)

    jv4 := Wsy.Json.New()
    jv4.Add("code",jv.AddToData())
    jv4.Add("msg", jv2)
    
    Wsy.Echo(jv4.AddToData(1))
    return Wsy.Json.ToIndent(jv4.AddToData(1))
*/
// Add: 链式添加字段，支持 *WsyJson、WsyJson 类型自动嵌套，字符串为 JSON 时自动转对象/数组
// 支持用法：
//   jv.Add("arr", "[1,2,3]") // 自动转为数组
//   jv.Add("obj", "{\"a\":1}") // 自动转为对象 
//   jv.Add("arr", []interface{}{1,2,3}) // 直接传数组
//   jv.Add("obj", map[string]interface{}{...}) // 直接传对象
//   jv.Add("", map[string]interface{}{...}) // 批量平铺
func (j *WsyJson) Add(path string, value interface{}) *WsyJson {
	// 如果 value 是 *WsyJson 或 WsyJson，自动转为嵌套对象
	switch v := value.(type) {
	case *WsyJson:
		value = gjson.Parse(v.AddToData()).Value()
	case WsyJson:
		value = gjson.Parse(v.AddToData()).Value()
	case string:
		str := v
		if len(str) > 0 && str[0] == '[' {
			value = j.ToArray(str)
		} else if len(str) > 0 && str[0] == '{' {
			value = j.ToMap(str)
		}
	case []interface{}:
		value = v
	case map[string]interface{}:
		if path == "" {
			for k, val := range v {
				j.Add(k, val)
			}
			return j
		} else {
			value = v
		}
	default:
		// 其他类型尝试 json.Marshal + 解析为数组
		data, err := json.Marshal(v)
		if err == nil {
			str := string(data)
			if len(str) > 0 && str[0] == '[' {
				value = j.ToArray(str)
			} else if len(str) > 0 && str[0] == '{' {
				value = j.ToMap(str)
			}
		}
	}
	j.data = j.Set(j.data, path, value)
	return j
}

// ToData: 返回当前 JSON 字符串，可选缩进格式
//   jv.ToData()         // 紧凑格式
//   jv.ToData(2)        // 缩进格式（只要参数数量>0即缩进）
func (j *WsyJson) AddToData(args ...interface{}) string {
	//rk读出来是nil时
	if j == nil {
		return "{}"
	}
	//正常处理
	if len(args) > 0 {
		return Json.ToJson(j.data)
	}
	return j.data
}
// ===================== 以下是获取json =====================
// Encode: 标准库序列化（无错误返回，出错返回空字符串）
// 示例：
//   jsonStr := WsyJson{}.Encode(map[string]interface{}{"a":1})
func (j WsyJson) Encode(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// Decode: 标准库反序列化
// 示例：
//   var m map[string]interface{}
//   err := WsyJson{}.Decode(jsonStr, &m)
func (j WsyJson) Decode(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

// Get: 路径查询（只读，gjson）
// 示例：
//   v := WsyJson{}.Get(jsonStr, "a.b.0.c").Int()
func (j WsyJson) Get(jsonStr, path string) gjson.Result {
	return gjson.Get(jsonStr, path)
}

// Exists: 路径是否存在（只读，gjson）
// 示例：
//   exists := WsyJson{}.Exists(jsonStr, "a.b")
func (j WsyJson) Exists(jsonStr, path string) bool {
	return gjson.Get(jsonStr, path).Exists()
}

// Query: 支持gjson的复杂路径语法（只读，gjson）
// 示例：
//   results := WsyJson{}.Query(jsonStr, "users.#(age>18)")
func (j WsyJson) Query(jsonStr, query string) []gjson.Result {
	result := gjson.Get(jsonStr, query)
	if result.IsArray() {
		return result.Array()
	}
	return []gjson.Result{result}
}

// Count: 统计数组/对象元素数量（只读，gjson）
// 示例：
//   n := WsyJson{}.Count(jsonStr, "users")
func (j WsyJson) Count(jsonStr, path string) int {
	result := gjson.Get(jsonStr, path)
	if result.IsArray() {
		return len(result.Array())
	}
	if result.IsObject() {
		return len(result.Map())
	}
	return 0
}

// Keys: 获取对象所有键（只读，gjson）
// 示例：
//   keys := WsyJson{}.Keys(jsonStr, "user")
func (j WsyJson) Keys(jsonStr, path string) []string {
	result := gjson.Get(jsonStr, path)
	if !result.IsObject() {
		return nil
	}
	m := result.Map()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values: 获取对象所有值（只读，gjson）
// 示例：
//   vals := WsyJson{}.Values(jsonStr, "user")
func (j WsyJson) Values(jsonStr, path string) []gjson.Result {
	result := gjson.Get(jsonStr, path)
	if !result.IsObject() {
		return nil
	}
	m := result.Map()
	values := make([]gjson.Result, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// ForEach: 遍历JSON数组，对每个元素执行回调函数
// 示例：
//   WsyJson{}.ForEach(jsonStr, "data.items", func(i int, item interface{}) bool {
//       fmt.Printf("索引: %d, 值: %s\n", i, item)
//       return true // 返回 true 继续遍历，返回 false 停止遍历
//   })
func (j WsyJson) ForEach(jsonStr, path string, fn func(i int, item interface{}) bool) {
	result := gjson.Get(jsonStr, path)
	if result.IsArray() {
		items := result.Array()
		for i, item := range items {
			if !fn(i, item.Value()) {
				break // 如果回调返回 false，停止遍历
			}
		}
	}
}

// ForEachMap: 遍历JSON对象，对每个键值对执行回调函数
// 示例：
//   WsyJson{}.ForEachMap(jsonStr, "data", func(key string, value interface{}) bool {
//       fmt.Printf("键: %s, 值: %s\n", key, value)
//       return true // 返回 true 继续遍历，返回 false 停止遍历
//   })
func (j WsyJson) ForEachMap(jsonStr, path string, fn func(key string, value interface{}) bool) {
	result := gjson.Get(jsonStr, path)
	if result.IsObject() {
		m := result.Map()
		for key, value := range m {
			if !fn(key, value.Value()) {
				break // 如果回调返回 false，停止遍历
			}
		}
	}
}

// Set: 设置/修改字段（只写，sjson）
// 示例：
//   New := WsyJson{}.Set(jsonStr, "a.b.c", 123)
func (j WsyJson) Set(jsonStr, path string, value interface{}) string {
	str, _ := sjson.Set(jsonStr, path, value)
	return str
}

// Delete: 删除字段（只写，sjson）
// 示例：
//   New, err := WsyJson{}.Delete(jsonStr, "a.b.c")
func (j WsyJson) Delete(jsonStr, path string) (string, error) {
	str, err := sjson.Delete(jsonStr, path)
	return str, err
}

// Append: 向数组追加元素（只写，sjson）
// 示例：
//   New, err := WsyJson{}.Append(jsonStr, "arr", 456)
func (j WsyJson) Append(jsonStr, path string, value interface{}) (string, error) {
	str, err := sjson.Set(jsonStr, path+".-1", value)
	return str, err
}

// Valid 检查字符串是否为有效的JSON
// 示例: isValid := Lin.Json.Valid(jsonString)
func (j  WsyJson) Valid(jsonInput interface{}) bool {
	// 处理不同类型的输入
	switch v := jsonInput.(type) {
	case string:
		//return gjson.Valid(v)
		// 字符串类型，先检查是否以 { 或 [ 开头
		v = strings.TrimSpace(v)
		if !strings.HasPrefix(v, "{") && !strings.HasPrefix(v, "[") {
			return false
		}
		// 尝试解析为JSON对象或数组
		var result interface{}
		return json.Unmarshal([]byte(v), &result) == nil
	case map[string]interface{}, []interface{}:
		// 对于已经是JSON对象或数组的输入，始终返回true
		return true
	case nil:
		// 空值视为无效JSON
		return false
	default:
		// 尝试将其他类型转换为字符串然后验证
		jsonStr := strings.TrimSpace(fmt.Sprintf("%v", v))
		if !strings.HasPrefix(jsonStr, "{") && !strings.HasPrefix(jsonStr, "[") {
			return false
		}
		var result interface{}
		return json.Unmarshal([]byte(jsonStr), &result) == nil
	}
}
// Msg: 构建标准返回JSON字符串
// 支持用法：
//   WsyJson{}.Msg("1", "操作成功", WsyJson{}.ToArray("[1,2,3]")) // data 字段为数组
//   WsyJson{}.Msg("1", "操作成功", map[string]interface{}{"id":123})
func (j WsyJson) Msg(code string, msg string, data ...interface{}) string {
    AddTips := WsyJson{}.New()
    AddTips.Add("code", code)
    AddTips.Add("msg", msg)
    if len(data) == 0 {
        AddTips.Add("data", "")
    } else if len(data) == 1 {
        AddTips.Add("data", data[0])
    } else {
        AddTips.Add("data", data)
    }
    return AddTips.AddToData(true)
}

func (j WsyJson) Err(msg string, data ...interface{}) string {
	if len(data) == 0 {
		return j.Msg("0", msg, "")
	} else {
		return j.Msg("0", msg, data...)
	}
}

func (j WsyJson) Ok(msg string, data ...interface{}) string {
	if len(data) == 0 {
		return j.Msg("1", msg, "")
	} else {
		return j.Msg("1", msg, data...)
	}
}

// Code: 判断是否成功
// 示例：
//   ok := Json.Code(jsonStr)
func (j WsyJson) Code(data string) bool {
	Code,_ := j.GetCode(data)
	if Code {
		return true
	}
	return false
}

// Code: 判断是否成功
// 示例：
//   ok := Json.Code(jsonStr)
func (j WsyJson) GetCode(data string) (bool,string) {
	if j.Exists(data, "code") {
		if j.Get(data, "code").String() == "1" {
			return true,j.Get(data, "msg").String()
		}
	}
	return false,j.Get(data, "msg").String()
}

// ToJson: 始终进行时间标准化，支持直接传入map、slice等类型，可选是否进行Unicode转义
// 示例：
//   WsyJson{}.ToJson(jsonStr)
//   WsyJson{}.ToJson(map[string]interface{}{...})
//   WsyJson{}.ToJson(obj, false) // 不转义
func (j WsyJson) ToJson(data interface{}, escapeUnicode ...bool) string {
	var jsonStr string
	switch v := data.(type) {
	case string:
		jsonStr = v
	default:
		str := j.Encode(v)
		if str == "" {
			return "{}"
		}
		jsonStr = str
	}
	// 先做时间标准化
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err == nil {
		obj = j.ToTimeStd(obj)
		jsonStrNew, err := json.Marshal(obj)
		if err == nil {
			jsonStr = string(jsonStrNew)
		}
	}
	// 判断是否需要转义
	doEscape := true
	if len(escapeUnicode) > 0 {
		doEscape = escapeUnicode[0]
	}
	if doEscape {
		return j.ToUnicode(jsonStr)
	}
	return jsonStr
}
// 防止 2025-07-13T10:22:11+08:00
func (j WsyJson) ToTimeStd(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			v[k] = j.ToTimeStd(val)
		}
		return v
	case []interface{}:
		for i, val := range v {
			v[i] = j.ToTimeStd(val)
		}
		return v
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
				return t.Format("2006-01-02")
			}
			return t.Format("2006-01-02 15:04:05")
		}
		return v
	default:
		return v
	}
}

// ToUnicode: 将非ASCII字符转为\uXXXX格式
func (j WsyJson) ToUnicode(s interface{}) string {
	str, ok := s.(string)
	if !ok {
		return ""
	}
	var out []rune
	for _, r := range str {
		if r > 127 {
			buf := make([]byte, 0, 6)
			buf = append(buf, '\\', 'u')
			hex := []byte("0123456789abcdef")
			for i := 12; i >= 0; i -= 4 {
				buf = append(buf, hex[(r>>uint(i))&0xF])
			}
			for _, b := range buf {
				out = append(out, rune(b))
			}
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// ToArray: 尝试将字符串或任意对象转为数组（[]interface{}），失败返回空切片
// 示例：arr := WsyJson{}.ToArray(jsonStr)
func (j WsyJson) ToArray(val interface{}) []interface{} {
	// 如果是字符串，尝试解析
	switch v := val.(type) {
	case string:
		if len(v) > 0 && v[0] == '[' {
			res := gjson.Parse(v)
			if res.Exists() && res.IsArray() {
				arr := res.Array()
				out := make([]interface{}, 0, len(arr))
				for _, item := range arr {
					out = append(out, item.Value())
				}
				return out
			}
		}
	case []interface{}:
		return v
	}
	// 其他类型尝试 json.Marshal + 解析
	data, err := json.Marshal(val)
	if err == nil {
		str := string(data)
		if len(str) > 0 && str[0] == '[' {
			res := gjson.Parse(str)
			if res.Exists() && res.IsArray() {
				arr := res.Array()
				out := make([]interface{}, 0, len(arr))
				for _, item := range arr {
					out = append(out, item.Value())
				}
				return out
			}
		}
	}
	return []interface{}{}
}

// ToMap: 尝试将字符串或任意对象转为 map[string]interface{}，失败返回空 map
// 示例：m := WsyJson{}.ToMap(jsonStr)
func (j WsyJson) ToMap(val interface{}) map[string]interface{} {
	switch v := val.(type) {
	case string:
		if len(v) > 0 && v[0] == '{' {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(v), &m); err == nil {
				return m
			}
		}
	case map[string]interface{}:
		return v
	}
	// 其他类型尝试 json.Marshal + 解析
	data, err := json.Marshal(val)
	if err == nil {
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err == nil {
			return m
		}
	}
	return map[string]interface{}{}
}


// ToMapStr 将INI读取的JSON字符串转换为map[string]string
// 
// 示例：
//   config := Fso.ToMapStr(jsonStr)  // 返回 map[string]string
//   port := config["WebPort"]         // "2511"
func (j *WsyJson) ToMapStr(jsonStr string) map[string]string {
	result := make(map[string]string)
	
	// 如果JSON字符串为空，返回空map
	if strings.TrimSpace(jsonStr) == "" {
		return result
	}
	// 解析JSON为map[string]interface{}
	configMap := j.ToMap(jsonStr)
	// 转换为map[string]string
	for k, v := range configMap {
		result[k] = Str.ToString(v)
	}
	return result
}

// ToIndent: 将 JSON 字符串美化缩进输出
func (j WsyJson) ToIndent(jsonStr string) string {
	return string(pretty.Pretty([]byte(jsonStr)))
}