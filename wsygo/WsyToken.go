package Wsy

import "strings"

type WsyToken struct{}

// Set 生成 token，过期时间为第一个参数（秒），后面为业务字段
// 调用示例：
//   Set(72000, "a", "b")
//   Set(72000, "a", "b", "c")
// 处理流程：
//   1) 使用第1个参数作为过期秒数
//   2) 计算过期时间戳(当前时间戳 + expire)
//   3) 将加密后的时间戳放在第 1 段，其余业务参数依次加密放在后面
//   4) 以 "|" 连接：enc(时间戳)|enc(a)|enc(b)|...，再整体 Key.EnCode
func (t *WsyToken) Set(expire int, values ...string) string {
	if len(values) == 0 || expire <= 0 { return "" }
	expireTs := Date.GetTimeStamp() + int64(expire)
	// 第 1 段为过期时间戳
	encodedParts := make([]string, 0, len(values)+1)
	encodedParts = append(encodedParts, Key.EnCode(Str.ToString(expireTs),Set.Token))
	// 后续为业务字段
	for _, v := range values {
		strVal := Str.ToString(v)
		if strVal == "" { return "" }
		encodedParts = append(encodedParts, Key.EnCode(strVal,Set.Token))
	}
	tokenStr := strings.Join(encodedParts, "|")
	return Key.EnCode(tokenStr,Set.Token)
}
// Expire 判断 token 是否未过期（true 表示未过期）
// 解密 -> 拆分 -> 取第 1 段时间戳与当前时间对比
func (t *WsyToken) Expire(token string) bool { 
	if token == "" { return false }
	plain := Key.DeCode(token,Set.Token)
	if plain == "" { return false }
	parts := strings.Split(plain, "|")
	if len(parts) == 0 { return false }
	// 第 1 段为时间戳
	expireStr := Key.DeCode(parts[0],Set.Token)
	expireTs := Str.ToInt64(expireStr)
	if expireTs == 0 { return false }
	return Date.GetTimeStamp() <= expireTs
}
// Get 解析 token。
// index 语义：
//   - index=0（默认）：返回过期时间戳
//   - index>=1：返回第 N 个业务字段（第 1 个业务字段对应整体中的第 2 段）
func (t *WsyToken) Get(token string, index ...int) string {
	if token == "" { return "" }
	i := 0
	if len(index) > 0 {
		i = index[0]
	}
	plain := Key.DeCode(token,Set.Token)
	if plain == "" { return "" }
	parts := strings.Split(plain, "|")
	// 至少需要 1 段（时间戳），没有任何段直接返回空
	if len(parts) == 0 { return "" }
	// index=0：返回时间戳（第 1 段）
	if i == 0 { return Key.DeCode(parts[0],Set.Token) }
	// 业务字段从第 2 段开始，对调用者而言 index=1 表示第 1 个业务字段，
	// 因此 index 的合法范围是 [1, len(parts)-1]
	if i < 1 || i > len(parts)-1 { return "" }
	target := parts[i]
	return Key.DeCode(target,Set.Token)
}
// Gets 解密 token 并返回所有字段的 map[string]string
// 约定：
//   - val0 ：过期时间戳
//   - val1 ：第 1 个业务字段
//   - val2 ：第 2 个业务字段
//   - 以此类推
// 解密失败或空 token 返回空 map
func (t *WsyToken) Gets(token string) map[string]string {
	result := make(map[string]string)
	if token == "" { return result }
	plain := Key.DeCode(token,Set.Token)
	if plain == "" { return result }
	parts := strings.Split(plain, "|")
	if len(parts) == 0 {
		return result
	}
	for i, p := range parts {
		result["val"+Str.ToString(i)] = Key.DeCode(p,Set.Token)
	}
	return result
}
// Refresh 刷新 token：保留原业务字段，重新设置过期时间
func (t *WsyToken) Refresh(token string, expire int) string {
	if token == "" || expire <= 0 { return "" }
	plain := Key.DeCode(token,Set.Token)
	if plain == "" { return "" }
	parts := strings.Split(plain, "|")
	// 至少需要 1 段时间戳 + 1 段业务字段
	if len(parts) < 2 { return "" }
	decodedVals := make([]string, 0, len(parts)-1)
	// 从第 2 段开始为业务字段
	for _, p := range parts[1:] { // 业务字段部分
		v := Key.DeCode(p,Set.Token)
		if v == "" { return "" }
		decodedVals = append(decodedVals, v)
	}
	return t.Set(expire, decodedVals...)
}
// Valid 校验 token 基本合法性：可解密且有至少2段（1 段时间戳 + 至少1段业务）
func (t *WsyToken) Valid(token string) bool {
	if token == "" { return false }
	plain := Key.DeCode(token,Set.Token)
	if plain == "" { return false }
	parts := strings.Split(plain, "|")
	return len(parts) >= 2
}
// Count 返回 token 中字段总数（含业务字段与时间戳，时间戳位于第 1 段）
// 解密失败或空 token 返回 0
func (t *WsyToken) Count(token string) int {
	if token == "" { return 0 }
	plain := Key.DeCode(token,Set.Token)
	if plain == "" { return 0 }
	parts := strings.Split(plain, "|")
	return len(parts)
}