package Wsy

import (
	"strings"
)

type WsyMap struct{}
// ToJson 将 map 或切片对象转换为 JSON 字符串
// 可选参数 escapeUnicode（bool），如 ToJson(data, true) 启用 Unicode 转义
func (m WsyMap) ToJson(data interface{}, escapeUnicode ...bool) string {
    if data == nil {
        if _, ok := data.(map[string]interface{}); ok {
            return "[]" //{}
        }
        return "[]"
    }
    return Json.ToJson(data, escapeUnicode...)
}
/*
    Wsy.Map.ReName(row, map[string]string{
        "GroupId": "Gid",
        "AgentId": "Aid",
    })
    Wsy.Map.Edit(row, map[string]interface{}{
        "Gid": 5,
        "Aid": 10086,
        "Other": "自定义字段",
    })
*/

// DelField 删除 map 中指定的字段，支持可变参数
// 示例：Map.DelField(row, "PassWord", "Sex") 或 Map.DelField(row, "PassWord,Sex")
func (m WsyMap) Del(row map[string]interface{}, fields ...string) {
    if row == nil {return}
    for _, field := range fields {
        for _, f := range strings.Split(field, ",") {
            f = strings.TrimSpace(f)
            if f != "" {
                delete(row, f)
            }
        }
    }
}

// Only 保留 map 中指定的字段，删除其他字段，支持可变参数
// 示例：Map.Only(row, "Id,Name,Age") 或 Map.Only(row, "Id", "Name", "Age")
func (m WsyMap) Only(row map[string]interface{}, fields ...string) {
    if row == nil {return}
    keepFields := make(map[string]bool)
    for _, field := range fields {
        for _, f := range strings.Split(field, ",") {
            f = strings.TrimSpace(f)
            if f != "" {
                keepFields[f] = true
            }
        }
    }
    // 删除不在保留列表中的字段
    for k := range row {
        if !keepFields[k] {
            delete(row, k)
        }
    }
}
// EditField 批量设置 map 的字段值
// 示例：Map.EditField(row, map[string]interface{}{ "Gid": 5, "Aid": 10086, "Name": "加一个字段" })
func (m WsyMap) Edit(row map[string]interface{}, valueMap map[string]interface{}) {
    if row == nil {return}
    for k, v := range valueMap {
        row[k] = v
    }
}

// MvField 批量重命名 map 的字段
// 示例：Map.MvField(row, map[string]string{"GroupId": "Gid", "AgentId": "Aid"})
func (m WsyMap) ReName(row map[string]interface{}, renameMap map[string]string) {
    if row == nil {return}
    for oldKey, newKey := range renameMap {
        if v, ok := row[oldKey]; ok {
            row[newKey] = v
            delete(row, oldKey)
        }
    }
}

// ToLower 返回一个所有key都转为小写的新map
func (m WsyMap) ToLower(row map[string]interface{}) map[string]interface{} {
    if row == nil {
        return nil
    }
    newMap := make(map[string]interface{}, len(row))
    for k, v := range row {
        newMap[strings.ToLower(k)] = v
    }
    return newMap
}

// ToStrMap 将 map[string]interface{} 转换为 map[string]string
// 所有值都会通过 Str.ToString 转换为字符串
// 示例：strMap := Map.ToStrMap(data)
func (m WsyMap) ToStrMap(row map[string]interface{}) map[string]string {
    if row == nil {
        return nil
    }
    result := make(map[string]string, len(row))
    for k, v := range row {
        result[k] = Str.ToString(v)
    }
    return result
}


// ForEachEx 遍历 []map[string]interface{}，回调返回 return false 时中断遍历
// 示例：Map.ForEachEx(arr, func(i int, row map[string]interface{}) bool { ... })
func (m WsyMap) ForBool(arr []map[string]interface{}, fn func(i int, row map[string]interface{}) bool) {
	for i, row := range arr {
		if !fn(i, row) {
			break
		}
	}
}
// ForEach 遍历 []map[string]interface{}，对每一行执行回调
// 示例：Map.ForEach(arr, func(i int, row map[string]interface{}) { ... })
func (m WsyMap) ForEach(arr []map[string]interface{}, fn func(i int, row map[string]interface{})) {
	for i, row := range arr {
		fn(i, row)
	}
}
//是否存在的值
func (m WsyMap) Exists(row map[string]interface{} ,value string) bool{
	if _, ok := row[value]; ok {
        return true
    }
    return false
}




