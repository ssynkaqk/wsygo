package Wsy

import (
	"os"
	"fmt"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)
// 定义全局命名空间变量
var (
	DB     WsyDB
	TLS    WsyTLS
	APP    WsyAPP
	Sum    WsySum
	Map    WsyMap
	Str    WsyStr
	Key    WsyKey
	Fso    WsyFso
	Gin    WsyGin
	Dns    WsyDns
	Tool   WsyTool
	Role   WsyRole
	Down   WsyDown
	Html   WsyHtml
	Json   WsyJson
	Date   WsyDate
	Http   WsyHttp
	Args   WsyArgs
	Cache  WsyCache
	Redis  WsyRedis
	Shell  WsyShell
	Token  WsyToken
	Timer  WsyTimer
	Fanyi  WsyFanyi
	Upload WsyUpload
)

// 退出软件
func Exit() {
	os.Exit(0)
}

// 退出软件
func Version() {
	fmt.Println(Set.Version)
}
// 返回 android,linux,windows
func OS() string {
	_, err := exec.LookPath("getprop")
	if err == nil {
		return "android"
	}
	if runtime.GOOS == "windows" {
		return "windows"
	}	
	return runtime.GOOS
}

// 返回 arm,arm64,amd64
func Arch() string {
	return runtime.GOARCH
}

// 返回 文件所在路径
func NowPath() string {
	NowPath := Fso.GetPath("path")
	if OS() == "android" {
		return NowPath
	} else {
		if Str.IsIn(NowPath, "/tmp/") || Str.IsIn(NowPath, "/root/") { //判断是否开发模式
			return "/opt/.xxdev"
		}
	}
	return NowPath
}

//电源操作：空：重启 1:重启 2:关机
func Power(Power ...interface{}){
	if len(Power) == 0 || Str.ToString(Power[0]) == "1" {
		Logs("INFO", "TASK", "系统重启中...")
		if OS() == "android" {
			result := Shell.Shell("am start -a android.intent.action.REBOOT")
			if Str.IsIn(result, "Permission Denial") || Str.IsIn(result, "SecurityException") {
				result = Shell.Shell("svc power reboot")
				if result == "" {
					return
				}
			} else {
				return
			}
		}
		Logs("INFO", "TASK", "使用reboot命令重启中...")
		Shell.Shell("reboot")
	} else if Str.ToString(Power[0]) == "2" { // 执行关机
		if OS() == "android" {
			Logs("INFO", "TASK", "关机成功")
			Shell.Shell("am start -a com.android.internal.intent.action.REQUEST_SHUTDOWN --ez android.intent.extra.KEY_CONFIRM false")
			Shell.Shell("reboot -p")
		} else {
			Shell.Shell("poweroff")
			Logs("INFO", "TASK", "关机成功")
		}
	}
}
//输出错误
func Error(value interface{}, msgOrShow ...interface{}) error {
	if len(msgOrShow) == 0 {
		switch v := value.(type) {
		case string:
			return errors.New(v)
		case error:
			return v
		case fmt.Stringer:
			return errors.New(v.String())
		default:
			return errors.New(fmt.Sprintf("%v", v))
		}
	}
	// 如果有格式化参数，优先格式化输出
	if s, ok := value.(string); ok && (strings.Contains(s, "%") || strings.Contains(s, " ")) {
		message := fmt.Sprintf(s, msgOrShow...)
		return errors.New(message)
	} else {
		// 其它类型，先格式化主内容，再格式化参数
		var parts []string
		parts = append(parts, fmt.Sprintf("%v", value))
		for _, arg := range msgOrShow {
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
		return errors.New(strings.Join(parts, " "))
	}
}

func Echo(typeOrMsg interface{}, msgOrShow ...interface{}) {
	if len(msgOrShow) == 0 {
		switch v := typeOrMsg.(type) {
			case string:
				fmt.Println(v)
			case fmt.Stringer:
				fmt.Println(v.String())
			default:
				fmt.Printf("%#v\n", v)
		}
		return
	}
	// 如果有格式化参数，优先格式化输出
	if s, ok := typeOrMsg.(string); ok && (strings.Contains(s, "%") || strings.Contains(s, " ")) {
		message := fmt.Sprintf(s, msgOrShow...)
		fmt.Println(message)
	} else {
		// 其它类型，先输出主内容，再输出参数
		fmt.Printf("%#v ", typeOrMsg)
		for _, arg := range msgOrShow {
			fmt.Printf("%#v ", arg)
		}
		fmt.Println()
	}
}

func Logs(level string, typeOrMsg string, msgOrShow ...interface{}) {
	DateTime := Date.Now()
	if len(msgOrShow) == 0 {
		if Set.Logs {
			fmt.Printf("[%s] [%s] %s\n", DateTime, level, typeOrMsg)
		}
		if Set.LogsSave {
			Fso.WriterMulti(Set.LogsFile,fmt.Sprintf("[%s] [%s] %s\n", DateTime, level, typeOrMsg))
		}
		return
	}
	LogOpen := msgOrShow[len(msgOrShow)-1]
	if msgOrShow[len(msgOrShow)-1] == "N" {
		return // 如果最后一个参数是 "N"，则直接返回
	}
	if LogOpen == "Y" {
		msgOrShow = msgOrShow[:len(msgOrShow)-1] // 删除最后一个参数 "Y"
	}
	if strings.Contains(typeOrMsg, "%") || strings.Contains(typeOrMsg, " ") {
		message := fmt.Sprintf(typeOrMsg, msgOrShow[0:]...)
		if Set.Logs || LogOpen == "Y" {
			fmt.Printf("[%s] [%s] %s\n", DateTime, level, message)
		}
		if Set.LogsSave {
			Fso.WriterMulti(Set.LogsFile,fmt.Sprintf("[%s] [%s] %s\n", DateTime, level, message))
		}
		return
	} else {
		if len(msgOrShow) > 0 {
			message := msgOrShow[0].(string)
			message = fmt.Sprintf(message, msgOrShow[1:]...)
			if Set.Logs || LogOpen == "Y" {
				fmt.Printf("[%s] [%s] [%s] %s\n", DateTime, level, typeOrMsg, message)
			}
			if Set.LogsSave {
				Fso.WriterMulti(Set.LogsFile,fmt.Sprintf("[%s] [%s] [%s] %s\n", DateTime, level, typeOrMsg, message))
			}
		} else {
			if Set.Logs || LogOpen == "Y" {
				fmt.Printf("[%s] [%s] %s\n", DateTime, level, typeOrMsg)
			}
			if Set.LogsSave {
				Fso.WriterMulti(Set.LogsFile,fmt.Sprintf("[%s] [%s] %s\n", DateTime, level, typeOrMsg))
			}
		}
	}
}

//启动多个goroutines，即使某个goroutine报错也会继续执行其他goroutine
//即使所有goroutines都结束，程序也不会退出
func GoFunc(fns ...func()) {
	// 启动一个守护goroutine，定期唤醒，确保即使所有用户goroutines都结束，也至少有一个活跃的goroutine
	// 这样可以避免死锁检测
	done := make(chan struct{})
		go func() {
			// 守护goroutine定期唤醒，保持程序活跃，避免被判定为永久阻塞
			// 使用1小时间隔，资源占用极低（定时器在等待期间几乎不消耗资源）
			// 如需更低占用，可以改为1天(24*time.Hour)或更长
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				// 什么都不做，只是定期唤醒，保持goroutine活跃
				// 每1小时唤醒一次，资源占用极低（约2KB内存，几乎无CPU占用）
			}
		}()
		// 启动所有用户goroutines
		for i, fn := range fns {
			go func(index int, fn func()) {
				// 使用defer recover捕获panic，确保即使出错也不影响其他goroutine
				defer func() {
					if err := recover(); err != nil {
						// 记录错误但继续运行，不中断程序
						Logs("ERROR", "GoFunc", fmt.Sprintf("Goroutine [%d] 发生错误: %v", index, err))
					}
				}()
				// 执行函数，如果函数内部有panic会被recover捕获
				if fn != nil {
					fn()
				} else {
					Logs("WARN", "GoFunc", fmt.Sprintf("Goroutine [%d] 函数为空，跳过执行", index))
				}
			}(i, fn)
		}
		// 永久阻塞，等待永远不会发送的值，保持程序运行
		// 由于守护goroutine定期唤醒，不会触发死锁检测
	<-done
}