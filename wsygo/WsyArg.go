package Wsy

import (
	"fmt"
	"os"
	"strings"
)

type WsyArgs struct{}

// removeBrackets 如果第一个字符是!，则过滤掉
func removeBrackets(value string) string {
	if len(value) > 0 && value[0] == '!' {
		return value[1:]
	}
	return value
}

// New 解析命令行参数，支持三种格式：
// 方式一：main.go sud --send 112 --aaaaa aaad aa -c a '--11 22'
// 方式二：main.go --sud 55 --send 112 --aaaaa aaad aa -c a '--11 22'
// 方式三：main.go --sud 55 host dd22 --send 112 --aaaaa aaad aa -c a '--11 22'
// Get := Wsy.Args.New()
// if Get["case"] != "" {
// 	switch Get["case"] {
// 		case "send":
// 			DevSend.SendData()
// 	}
// }
/*
if _, exists := Get["d"]; exists {
	
}
*/
func (m WsyArgs) New() map[string]string {
	Get := make(map[string]string)
	if len(os.Args) <= 1 {
		return Get
	}
	args := os.Args[1:]
	valCount := 1
	i := 0
	if !strings.HasPrefix(args[0], "-") {
		Get["case"] = args[0]
		i = 1
	}
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			key := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(arg, "--"), "-"))
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				Get[key] = removeBrackets(args[i+1])
				i += 2
			} else {
				Get[key] = ""
				i++
			}
		} else {
			Get[fmt.Sprintf("val%d", valCount)] = arg
			valCount++
			i++
		}
	}
	return Get
}
// Get(Get,"val1","m","日志内容不能为空!")
func (m WsyArgs) Get(Get map[string]string,Val,Def string, args ...string) string {
	var NewVal string
	if len(args) > 0 {
		NewVal = args[0]
	}
	NewVals := Str.IIF(Get[Val] != "",Get[Val],Get[Def])
	if NewVals != "" {
		return NewVals
	}
	return NewVal
}
// NewStr(Get,"val1","LOGS功能还未开发!")
// NewStr(Get,"u","LOGS功能还未开发!")
func (m WsyArgs) NewStr(Get map[string]string,index string, args ...string) string {
	var DelText string
	if len(args) > 0 {
		DelText = args[0]
	}
	if Str.IsIn(index, "val") {
		return Str.IIF(DelText=="",Get[index],DelText)
	}else{
		if _, exists := Get[index]; exists {
			return Str.IIF(Get[index] !="",Get[index],DelText)
		}
	}
	return DelText
}

//判断是否存在 -c aa -b bb
//Exists(Get,"c")
func (m WsyArgs) Exists(Get map[string]string,index string, args ...string) bool {
	if _, exists := Get[index]; exists {
		return true
	}
	return false
}
