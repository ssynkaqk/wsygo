package Wsy

import (
	"strconv"
	"strings"
)

type WsyRole struct{}

var number int = 100
// Num 返回默认的200位权限字符串（全为'0'）
// 返回：200位全'0'的二进制字符串
// 示例：
//   result := Wsy.Role.Num()
//   // result = "0000000000000000000000000000000000000..."（200位全'0'）
func (r *WsyRole) Num() string {
	return strings.Repeat("0", number)
}

// ToNum 将权限字符串补齐到标准长度（200位）
// 参数：
//   - s: 要补齐的权限字符串
// 返回：补齐后的200位二进制字符串，不足部分用'0'补齐
// 说明：
//   - 如果字符串长度大于200位，截取前200位（超出部分会被丢弃）
//   - 如果字符串长度等于200位，直接返回
//   - 如果字符串长度不足200位，在末尾补'0'到200位
//   - 验证字符串中的每个字符，如果不是'0'或'1'，自动改为'0'
func (r *WsyRole) ToNum(s string) string {
	expectedLen := len(r.Num())
	// 先验证并修正字符串中的每个字符
	result := []byte(s)
	for i := 0; i < len(result); i++ {
		if result[i] != '0' && result[i] != '1' {
			result[i] = '0'
		}
	}
	s = string(result)
	// 处理长度
	if len(s) > expectedLen {
		return s[:expectedLen]  // 大于200位，截取前200位
	}
	if len(s) == expectedLen {
		return s  // 等于200位，直接返回
	}
	return s + strings.Repeat("0", expectedLen-len(s)) // 小于200位，补齐到200位
}
// Get 检查指定位置的权限
// 参数：
//   - key: 权限字符串（200位二进制字符串，如 "101010..."），每个位置代表一个权限项，'1'表示有权限，'0'表示无权限
//   - index: 要检查的权限位置（1-200），注意位置从1开始计数
// 返回：true表示有权限，false表示无权限
// 示例：
//   hasPermission := Wsy.Role.Get("101010...", 1)  // 检查第1个权限
//   hasPermission := Wsy.Role.Get("101010...", 3)  // 检查第3个权限
func (r *WsyRole) Get(key string, index int) bool {
	if key == "" {
		return false
	}
	// 使用ToNum验证和补齐key
	key = r.ToNum(key)
	if index <= 0 || index > len(key) {
		return false
	}
	ch := key[index-1]
	if ch != '0' && ch != '1' {
		return false
	}
	return ch == '1'
}

// Set 修改权限字符串指定位置的值
// 参数：
//   - key: 权限字符串（200位二进制字符串）
//   - index: 要修改的权限位置（1-200），注意位置从1开始计数
//   - value: 要设置的值（0或1），0表示无权限，1表示有权限
// 返回：修改后的200位二进制字符串
// 说明：
//   - 如果index超出范围（小于1或大于200），返回补齐后的原字符串
//   - 如果value不是0或1，返回补齐后的原字符串
//   - 返回的字符串始终是200位长度
// 示例：
//   result := Wsy.Role.Set("101010...", 13, 0)  // 将第13位设置为0
//   result := Wsy.Role.Set("101010...", 13, 1)  // 将第13位设置为1
func (r *WsyRole) Set(key string, index int, value int) string {
	// 先补齐到标准长度
	key = r.ToNum(key)
	// 检查index是否有效
	if index < 1 || index > len(key) {
		return key
	}
	// 检查value是否有效
	if value != 0 && value != 1 {
		return key
	}
	// 转换为字节数组进行修改
	result := []byte(key)
	result[index-1] = byte('0' + value)
	return string(result)
}
// Valid 验证用户权限，直接返回JSON错误信息（用于Web接口）
// 参数：
//   - Data: 包含用户信息的map，必须包含 "role" 和 "qx" 字段
//     - "role": 用户角色（"admin"、"team"、"user"）
//     - "qx": 权限字符串（200位二进制字符串）
//   - index: 要验证的权限位置（1-200）
// 返回：
//   - 空字符串 ""：表示有权限，验证通过
//   - JSON错误字符串：表示无权限（如 "您没有权限访问！"）
// 说明：
//   - "admin" 角色直接通过验证，无需检查权限字符串
//   - "team" 和 "user" 角色需要检查权限字符串对应位置是否为'1'
// 示例：
//   user := map[string]string{
//       "role": "user",
//       "qx": "101010...",
//   }
//   result := Wsy.Role.Valid(user, 2)
//   if result != "" {
//       return result  // 返回错误信息
//   }
func (r *WsyRole) Valid(Data map[string]string, index int) string {
    if Data["role"] == "team" || Data["role"] == "user" {
		qx := r.ToNum(Data["qx"])
        if !r.Get(qx, index) { 
			return Json.Err("您没有访问权限！")
		}
    }
	return ""
}

// Build 将逗号分隔的数字列表转换为200位权限字符串
// 参数：
//   - Data: 逗号分隔的数字字符串，表示有权限的位置（如 "1,3,29,28,5"）
// 返回：200位二进制字符串，对应位置为'1'，其余为'0'
// 说明：
//   - 输入数字范围：1-200，超出范围的数字会被忽略
//   - 无效的数字会被忽略
//   - 自动去除空格
//   - 不需要排序，直接设置对应位置即可
// 示例：
//   // 输入："1,3,29,28,5,7,9"
//   // 输出："1010000000000000000000000000101100000..."（200位）
//   // 位置1,3,5,7,9,28,29为'1'，其余为'0'
//   result := Wsy.Role.Build("1,3,29,28,5,7,9")
//   // result = "1010000000000000000000000000101100000..."（200位）
func (r *WsyRole) Build(Data string) string {
	result := []byte(r.Num())
	if Data == "" {
		return r.ToNum(string(result))
	}
	parts := strings.Split(Data, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		num, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		if num >= 1 && num <= len(r.Num()) {
			result[num-1] = '1'
		}
	}
	return string(result)
}

// Both 验证两个权限字符串的关系
// 参数：
//   - val1: 基础权限字符串（值一），200位二进制字符串
//   - val2: 要验证的权限字符串（值二），200位二进制字符串
// 返回：true表示验证通过，false表示验证失败
// 规则：
//   - 如果值一在某个位置是'0'，值二在相同位置不能是'1'（值二不能有值一没有的权限）
//   - 如果值一在某个位置是'1'，值二可以是'1'或'0'（值二可以有值一有的权限，也可以没有）
// 说明：
//   - 值二必须是值一的子集，值二的所有权限都必须在值一中存在
//   - 如果两个字符串长度不一致，取较短的长度进行比较
// 示例：
//   val1 := "1010000000000000000000000000101100000..."  // 值一
//   val2 := "1100000000000000000000000000101100000..." // 值二（第2位错误，值一为0，值二为1）
//   result := Wsy.Role.Both(val1, val2)  // 返回 false，因为第2位违反规则
//
//   val1 := "1010000000000000000000000000101100000..."  // 值一
//   val2 := "1000000000000000000000000000101100000..." // 值二（正确，第3位值一为1，值二为0，符合规则）
//   result := Wsy.Role.Both(val1, val2)  // 返回 true
//
//   val1 := "1010000000000000000000000000101100000..."  // 值一
//   val2 := "1110000000000000000000000000101100000..." // 值二（正确，第3位值一为1，值二为1，符合规则）
//   result := Wsy.Role.Both(val1, val2)  // 返回 true
func (r *WsyRole) Both(val1, val2 string) bool {
	// 如果任一字符串为空，返回false
	if val1 == "" || val2 == "" { return false }
	val1 = r.ToNum(val1)
	val2 = r.ToNum(val2)
	// 遍历每个位置进行验证
	for i := 0; i < len(val1); i++ {
		// 如果值二在该位置是'1'，值一在该位置也必须是'1'
		if val2[i] == '1' && val1[i] != '1' {
			return false
		}
		// 如果值二在该位置不是'0'或'1'，视为无效
		if val2[i] != '0' && val2[i] != '1' {
			return false
		}
	}
	return true
}
// Fix 强制修正权限字符串，使其符合规则
// 参数：
//   - val1: 基础权限字符串（值一），200位二进制字符串
//   - val2: 要修正的权限字符串（值二），200位二进制字符串
// 返回：修正后的权限字符串
// 规则：
//   - 如果值一在某个位置是'0'，强制将值二在该位置设为'0'（移除值一没有的权限）
//   - 如果值一在某个位置是'1'，保持值二在该位置的值不变（可以是'1'或'0'）
// 说明：
//   - 修正后的值二一定是值一的子集
//   - 如果值二为空，返回与值一长度相同的全'0'字符串
//   - 如果值一为空，返回空字符串
// 示例：
//   val1 := "1010000000000000000000000000101100000..."  // 值一
//   val2 := "1100000000000000000000000000101100000..." // 值二（第2位错误，值一为0，值二为1）
//   result := Wsy.Role.Fix(val1, val2)  // 返回修正后的值二，第2位被强制改为'0'
//   // result = "1000000000000000000000000000101100000..."（第2位已修正为'0'，第3位保持为'0'）
//
//   val1 := "1010000000000000000000000000101100000..."  // 值一
//   val2 := "1110000000000000000000000000101100000..." // 值二（第2位错误，第3位正确）
//   result := Wsy.Role.Fix(val1, val2)  // 返回修正后的值二
//   // result = "1010000000000000000000000000101100000..."（第2位已修正为'0'，第3位保持为'1'）
func (r *WsyRole) Fix(val1, val2 string) string {
	if val1 == "" {
		return r.Num()
	}
	val1 = r.ToNum(val1)
	val2 = r.ToNum(val2)
	result := make([]byte, len(val1))
	for i := 0; i < len(val1); i++ {
		if val1[i] == '0' {
			result[i] = '0'
		} else {
			// 如果值一在该位置是'1'，使用值二在该位置的值
			if i < len(val2) && (val2[i] == '0' || val2[i] == '1') {
				result[i] = val2[i]
			} else {
				result[i] = '0'
			}
		}
	}
	return string(result)
}