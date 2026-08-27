package Wsy

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"github.com/shirou/gopsutil/process"
)

type WsyShell struct{}

// Run 执行shell命令并返回标准输出和错误
// 参数：
//   - value: shell命令（支持管道和多个命令）
//
// 返回：
//   - string: 标准输出（去除末尾换行符）
//   - error: 错误信息。如果命令执行失败(err != nil)则返回该错误；如果命令成功但stderr有内容，则返回stderr作为错误
//
// 示例：
//
//	// 执行命令
//	stdout, err := Run("ls -l")
//	if err != nil {
//	    fmt.Printf("执行失败: %v\n", err)
//	} else {
//	    fmt.Printf("输出: %s\n", stdout)
//	}
func (f *WsyShell) Run(value string) (string, error) {
	if value == "" { return "", errors.New("命令不能为空") }
	cmd := exec.Command("sh", "-c", value)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	stdout := strings.TrimRight(stdoutBuf.String(), "\n")
	stderr := strings.TrimRight(stderrBuf.String(), "\n")
	if err != nil {
		if stderr != "" {
			return stdout, errors.New(stderr)
		}
		return stdout, err
	}
	if stderr != "" {
		return stdout, errors.New(stderr)
	}
	return stdout, nil
}
// Start 启动后台命令（不等待完成，立即返回）
// 参数：
//   - value: shell命令（支持管道和多个命令）
//
// 返回：
//   - *exec.Cmd: 命令对象，可用于后续管理（如 Kill、Wait 等）
//   - error: 启动错误，成功时为nil
//
// 示例：
//
//	// 启动后台命令
//	cmd, err := Start("minicap -P 1920x1080@1920x1080/0")
//	if err != nil {
//	    fmt.Printf("启动失败: %v\n", err)
//	} else {
//	    // 命令已在后台运行，可以继续其他操作
//	    // 如需停止：cmd.Process.Kill()
//	}
func (f *WsyShell) Start(value string) (*exec.Cmd, error) {
	if value == "" {
		return nil, errors.New("命令不能为空")
	}
	cmd := exec.Command("sh", "-c", value)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动命令失败: %w", err)
	}
	return cmd, nil
}

// 执行Shell脚本  //错误返回空值
// Shell 执行shell命令并返回结果
// 参数：
//   - value: shell命令（支持管道和多个命令）
//
// 返回：
//   - string: 命令输出（去除末尾换行符），错误时返回空字符串
//
// 示例：
//
//	// 执行单个命令
//	output := Shell("ls -l")
//
//	// 执行多个命令
//	output := Shell("cd /tmp && ls -l")
//
//	// 使用管道
//	output := Shell("ps -ef | grep nginx")
func (f *WsyShell) Shell(value ...string) string {
	var IsBack string = ""
	value = append([]string{"-c"}, value...)
	cmd := exec.Command("sh", value...)
	output, err := cmd.CombinedOutput() //CombinedOutput
	if err == nil {
		IsBack = string(output)
		IsBack = regexp.MustCompile(`\n$`).ReplaceAllString(IsBack, "")
	}
	return IsBack
}

// 执行Shell脚本 "ls","-al"   "ls -al"  //返回带错误的真实值,用于调试
// Command 执行shell命令并返回详细结果（包括错误信息）
// 参数：
//   - value: shell命令
//
// 返回：
//   - string: 命令输出或错误信息
//
// 示例：
//
//	// 执行命令
//	output := Command("ls -l")
//
//	// 执行不存在的命令
//	output := Command("invalid-cmd")  // 返回错误信息
//
//	// 执行多个命令
//	output := Command("cd /tmp && ls -l")
func (f *WsyShell) Command(value ...string) string {
	value = append([]string{"-c"}, value...)
	cmd := exec.Command("sh", value...)
	//创建获取命令输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "无法获取命令的stdout管道：" + err.Error()
	}
	//执行命令
	if err := cmd.Start(); err != nil {
		return "执行命令错误：" + err.Error()
	}
	//读取所有输出
	IsBack, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "标准输出错误:" + err.Error()
	}
	if err := cmd.Wait(); err != nil {
		return "不支持的命令：" + err.Error()
	}
	return string(IsBack)
}

// GetRoot 获取root权限
// 参数：
//   - value[0]: 操作类型（"open"表示打开，"close"表示关闭）
//   - value[1]: 操作的目录（多个目录用","分隔，如"system,vendor"）
//
// 返回：
//   - "ok": 操作成功
//   - "err": 操作失败
//   - "0": 参数错误
func (f *WsyShell) UnlockSystem(value ...string) string {
	if OS() != "android" {
		return ""
	}
	if len(value) < 2 {
		return "0"
	}
	operation := value[0]
	targets := value[1]
	mountOption := "ro"

	if operation == "open" {
		mountOption = "rw"
	}

	if targets == "all" {
		targets = "system,vendor"
	}
	for _, path := range strings.Split(targets, ",") {
		f.Shell(fmt.Sprintf("mount -o %s,remount /%s", mountOption, path))
		testOutput := f.Shell(fmt.Sprintf("touch /%s/test_write && rm /%s/test_write", path, path))
		if testOutput == "" {
			Logs("INFO", "UNLOCK", "%s 验证可写: 成功", path)
		} else {
			Logs("WARN", "UNLOCK", "%s 验证可写失败: %s", path, testOutput)
		}
	}
	return "ok"
}
// Ps 模糊查找进程列表，返回详细信息
// 参数：
//   - value: 进程名（模糊匹配，为空则返回所有进程）
//
// 返回：
//   - []map[string]interface{}: 进程信息列表，每个map包含：
//     - "pid": 进程ID (int32)
//     - "name": 进程名称 (string)
//     - "user": 用户名 (string)
//     - "cmdline": 完整命令行 (string)
//     - "exe": 可执行文件路径 (string)
//     - "status": 进程状态 (string)
//     - "cpu": CPU使用率 (float64)
//     - "mem": 内存使用率 (float64)
//     - "createTime": 创建时间 (int64)
//
// 示例：
//
//	// 查找所有包含 "devos" 的进程
//	processes := Shell.Ps("devos")
//	for _, proc := range processes {
//	    fmt.Printf("PID: %v, Name: %v, User: %v\n", proc["pid"], proc["name"], proc["user"])
//	}
//
//	// 获取所有进程
//	allProcesses := Shell.Ps("")
func (f *WsyShell) PS(value string) []map[string]interface{} {
	procs, err := process.Processes()
	if err != nil {
		return []map[string]interface{}{}
	}
	curr := os.Getpid()
	var result []map[string]interface{}
	searchValue := strings.ToLower(strings.TrimSpace(value))
	
	for _, p := range procs {
		if int(p.Pid) == curr {
			continue
		}
		name, err := p.Name()
		if err != nil {
			continue
		}
		if searchValue != "" {
			if !strings.Contains(strings.ToLower(name), searchValue) {
				cmdline, _ := p.Cmdline()
				if !strings.Contains(strings.ToLower(cmdline), searchValue) {
					continue
			}
		}
	}
		procInfo := map[string]interface{}{
			"pid": p.Pid,
			"name": name,
		}
		if username, err := p.Username(); err == nil {
			procInfo["user"] = username
		} else {
			procInfo["user"] = ""
		}
		if cmdline, err := p.Cmdline(); err == nil {
			procInfo["cmdline"] = cmdline
		} else {
			procInfo["cmdline"] = ""
		}
		if exe, err := p.Exe(); err == nil {
			procInfo["exe"] = exe
		} else {
			procInfo["exe"] = ""
	}
		if status, err := p.Status(); err == nil && len(status) > 0 {
			procInfo["status"] = string(status[0])
		} else {
			procInfo["status"] = ""
		}
		if cpu, err := p.CPUPercent(); err == nil {
			procInfo["cpu"] = cpu
		} else {
			procInfo["cpu"] = 0.0
		}
		if mem, err := p.MemoryPercent(); err == nil {
			procInfo["mem"] = mem
		} else {
			procInfo["mem"] = 0.0
		}
		if createTime, err := p.CreateTime(); err == nil {
			procInfo["createTime"] = createTime
		} else {
			procInfo["createTime"] = int64(0)
		}
		result = append(result, procInfo)
	}
	return result
}
// Pid 精准获取进程PID（类似 pidof），按可执行名匹配，忽略前缀 "./"
// 可选：支持按命令行参数进一步过滤（contains 匹配）
// 示例：Pid("devos") 可匹配以下进程的 PID：
//   devos --uid 25101 ...
//   ./devos send
//   devos:aa --uid ...
//   ./devos:aa send
// 示例：Pid("devos") 全部 devos
// 示例：Pid("devos","send") 仅匹配命令行包含 send 的进程
// 示例：Pid("devos","--uid 25101","--boxid dev") 同时包含多个条件才匹配
// 返回：
//   - "0" 未找到
//   - "pid1,pid2" 找到多个
func (f *WsyShell) Pid(value string, args ...string) string {
	if Str.IsNull(value) { return "0" }
	procName := filepath.Base(value)
	procName = strings.TrimPrefix(procName, "./")
	procs, err := process.Processes()
	if err != nil {
		return "0"
	}
	curr := os.Getpid()
	var pids []string
	wantArgs := make([]string, 0, len(args))
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a != "" {
			wantArgs = append(wantArgs, a)
		}
	}
	for _, p := range procs {
		if int(p.Pid) == curr {
			continue
		}
		cmd, err := p.CmdlineSlice()
		if err != nil || len(cmd) == 0 {
			continue
		}
		// 处理通过 /bin/sh 启动的脚本
		var exe string
		var cmdArgs string
		var cmdArgsSlice []string
		firstCmd := filepath.Base(cmd[0])
		firstCmd = strings.TrimPrefix(firstCmd, "./")
		if firstCmd == "sh" || firstCmd == "bash" {
			// 如果第一个命令是 shell，从后续参数中查找实际可执行文件
			if len(cmd) > 1 {
				// 检查第二个参数（脚本路径）
				scriptPath := cmd[1]
				exe = filepath.Base(scriptPath)
				exe = strings.TrimPrefix(exe, "./")
				if len(cmd) > 2 {
					cmdArgsSlice = cmd[2:]
					cmdArgs = strings.Join(cmdArgsSlice, " ")
				}
			}
			// 如果没找到，尝试从完整命令行中查找
			if exe == "" || exe == firstCmd {
				cmdline, _ := p.Cmdline()
				fields := strings.Fields(cmdline)
				for _, field := range fields {
					if strings.Contains(field, procName) {
						exe = filepath.Base(field)
						exe = strings.TrimPrefix(exe, "./")
						break
					}
				}
				if cmdArgs == "" {
					cmdArgs = cmdline
					cmdArgsSlice = fields
				}
			}
		} else {
			exe = firstCmd
			if len(cmd) > 1 {
				cmdArgsSlice = cmd[1:]
				cmdArgs = strings.Join(cmdArgsSlice, " ")
			}
		}
		if exe != procName {
			continue
		}
		if len(wantArgs) > 0 {
			matched := true
			for _, w := range wantArgs {
				// w 不包含空格：按 token 精准匹配，避免 fortest 命中 fortest2
				// w 包含空格：按 contains（兼容 "--uid 25101" 这种传法）
				if strings.Contains(w, " ") {
					if !strings.Contains(cmdArgs, w) {
						matched = false
						break
					}
					continue
				}
				found := false
				for _, tok := range cmdArgsSlice {
					if tok == w {
						found = true
						break
					}
				}
				if !found {
					matched = false
					break
				}
			}
			if !matched {
			continue
			}
		}
		pids = append(pids, strconv.Itoa(int(p.Pid)))
	}
	if len(pids) == 0 {
		return "0"
	}
 	return strings.Join(pids, ",")
}
// EndPid 结束指定PID的进程（支持多个PID，逗号分隔）
// 参数：
//   - Pid: 进程PID字符串，支持单个或多个（如 "12345" 或 "123456,123,456,1234564,125"）
//
// 示例：
//
//	// 结束单个PID
//	Shell.EndPid("12345")
//
//	// 结束多个PID
//	Shell.EndPid("123456,123,456,1234564,125")
//
//	// 使用 GetArgs 获取PID后结束进程
//	pid := Shell.GetArgs("/data/devdata", "xterm")
//	if pid != "0" {
//	    Shell.EndPid(pid)
//	}
func (f *WsyShell) EndPid(Pid string) {
	if Pid == "" || Pid == "0" { return }
	currentPID := os.Getpid()
	pidList := strings.Split(Pid, ",")
	for _, pidStr := range pidList {
		pidStr = strings.TrimSpace(pidStr)
		if pidStr == "" {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			Logs("ERROR", "SHELL", "无效的PID: %s", pidStr)
			continue
		}
		if pid == 0 {
			continue
		}
		// 不能结束自己的进程
		if pid == currentPID {
			Logs("WARNING", "SHELL", "不能结束当前进程 PID: %d", pid)
			continue
		}
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			Logs("ERROR", "SHELL", "无法找到进程 PID: %d, 错误: %v", pid, err)
			continue
		}
		err = proc.Kill()
		if err != nil {
			Logs("ERROR", "SHELL", "结束进程失败 PID: %d, 错误: %v", pid, err)
			continue
		}
		Logs("INFO", "SHELL", "成功结束进程 PID: %d", pid)
	}
}

// RunBin 直接启动可执行文件（不使用shell命令，后台运行）并返回PID
// 参数：
//   - programPath: 可执行文件路径（必填）
//   - args: 命令行参数（可选，多个参数）
//
// 返回：
//   - string: 进程PID（启动失败返回 "0"）
//
// 说明：
//   - 程序在后台启动，不等待完成
//   - 输出重定向到 /dev/null，不会阻塞当前进程
//
// 示例：
//
//	// 启动程序（无参数）
//	pid := Shell.RunBin("/usr/local/bin/pdldata")
//
//	// 启动程序（带参数）
//	pid := Shell.RunBin("/usr/local/bin/pdldata","--uid 25101","--boxid dev","--path /opt/xxdev")
func (f *WsyShell) RunBin(exeFile string, args ...string) string {
	if !Fso.IsFile(exeFile) { return "0" }
	cmd := exec.Command(exeFile, args...)
	cmd.Args[0] = filepath.Base(exeFile)
	// 设置进程属性，让进程独立运行（脱离父子关系）
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新的会话，让进程独立运行
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil { return "0" }
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		devNull.Close()
		return "0"
	}
	if cmd.Process == nil {
		devNull.Close()
		return "0"
	}
	go func() {
		cmd.Wait()
		devNull.Close()
	}()
	return strconv.Itoa(cmd.Process.Pid)
}
// RunSh 使用sh命令启动文件（后台运行）并返回PID
// 参数：
//   - scriptFile: 脚本文件路径（必填）
//   - args: 命令行参数（可选，多个参数）
//
// 返回：
//   - string: 进程PID（启动失败返回 "0"）
//
// 说明：
//   - 使用 sh 命令执行文件，适用于脚本文件
//   - 程序在后台启动，不等待完成
//   - 输出重定向到 /dev/null，不会阻塞当前进程
//
// 示例：
//
//	// 启动脚本（无参数）
//	pid := Shell.RunSh("/opt/.xxdev/devsh")
//
//	// 启动脚本（带参数）
//	pid := Shell.RunSh("/opt/.xxdev/devsh", "send", "task")
func (f *WsyShell) RunSh(scriptFile string, args ...string) string {
	if !Fso.IsFile(Fso.AbsPath(scriptFile)) { return "0" }
	scriptDir := Fso.GetPath(scriptFile, "path")
	scriptArg := Fso.GetPath(scriptFile, "name")
	cmdArgs := append([]string{scriptArg}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	if scriptDir != "" && scriptDir != "." {
		cmd.Dir = scriptDir
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil { return "0" }
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		devNull.Close()
		return "0"
	}
	if cmd.Process == nil {
		devNull.Close()
		return "0"
	}
	go func() {
		cmd.Wait()
		devNull.Close()
	}()
	return strconv.Itoa(cmd.Process.Pid)
}
// RunText 在内存中通过 sh 执行脚本文本（不落地文件）
// 参数：
//   - code: 明文脚本或加密脚本 执行
// 返回：
//   - string: 脚本标准输出（去除末尾换行），执行失败时返回空字符串
// 说明：
//   - 仅在内存中处理脚本内容，不创建临时脚本文件
func (f *WsyShell) RunText(code string) string {
	if Str.IsNull(code) { 
		return ""
	}else{
		codeKey := Key.DeCode(code)
		if !Str.IsNull(codeKey) { 
			code = codeKey
		}
	}
	cmd := exec.Command("sh")
	cmd.Stdin = strings.NewReader(code)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	out := regexp.MustCompile(`\n$`).ReplaceAllString(string(output), "") //过滤最后一行
	return out
}
// Nohup 使用 nohup 命令执行脚本文件（后台运行）并返回PID
// 参数：
//   - scriptFile: 脚本文件路径（必填）
//   - args: 命令行参数（可选，多个参数）
//
// 返回：
//   - string: 进程PID（启动失败返回 "0"）
//
// 说明：
//   - 使用 cd 切换到脚本目录，然后用 nohup ./文件名 执行
//   - 进程名会显示为脚本文件名（如 dev01），而不是 sh dev01
//   - 程序在后台启动，不等待完成
//   - 输出重定向到 /dev/null，不会阻塞当前进程
//
// 示例：
//
//	// 启动脚本（无参数）
//	pid := Shell.Nohup("/opt/.xxdev/devsh")
//
//	// 启动脚本（带参数）
//	pid := Shell.Nohup("/opt/.xxdev/devsh", "send", "task")
func (f *WsyShell) Nohup(scriptFile string, args ...string) string {
	if !Fso.IsFile(Fso.AbsPath(scriptFile)) { return "0" }
	scriptDir := Fso.GetPath(scriptFile, "path")
	scriptArg := Fso.GetPath(scriptFile, "name")
	cmdArgs := append([]string{"./" + scriptArg}, args...)
	cmd := exec.Command("nohup", cmdArgs...)
	if scriptDir != "" && scriptDir != "." {
		cmd.Dir = scriptDir
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil { return "0" }
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		devNull.Close()
		return "0"
	}
	if cmd.Process == nil {
		devNull.Close()
		return "0"
	}
	go func() {
		cmd.Wait()
		devNull.Close()
	}()
	return strconv.Itoa(cmd.Process.Pid)
}
//找到并干掉进程 
func (f *WsyShell) Kill(value string, args ...string) string { 
	pids := f.Pid(value, args...)
	if pids != "0" {
		Logs("INFO", "SHELL", "找到进程: %s, PID: %s", value, pids)
		f.EndPid(pids)
	}else{
		Logs("INFO", "SHELL", "未找到进程: %s", value)
	}
	return pids
}
// 检测命令参数是否在运行，在运行则退出程序
func (f *WsyShell) Exit(value ...string) {
	pids := f.Pid(Fso.GetPath(),value...)
	if pids != "0" {
		Logs("INFO", "SHELL", "程序已在运行 (其他PID: %s, 当前PID: %d)", pids, os.Getpid(), "Y")
		os.Exit(0)
	}
}
// 检测命令参数是否在运行，提示并返回true
func (f *WsyShell) CheckArgsRuning(args ...string) bool{
	pids := f.Pid(Fso.GetPath("name"),args...)
	if pids != "0" {
		Logs("INFO", "SHELL", "程序已在运行 (其他PID: %s, 当前PID: %d)", pids, os.Getpid())
		return true
	}
	return false
}
// RunOnce 检测进程是否运行，未运行则自动启动
// 参数：
//   - value: 可执行文件路径
//   - args: 命令行参数（可选）
//
// 返回：
//   - string: 进程PID（"0"表示启动失败）
//   - error: 错误信息
//
// 示例：
//
//	// 检测并启动（无参数）
//	pid, err := Shell.RunOnce("/opt/.xxdev/devos")
//
//	// 检测并启动（带参数）
//	pid, err := Shell.RunOnce("/opt/.xxdev/devos", "send")
func (f *WsyShell) RunOnce(value string, args ...string) string {
	if Str.IsNull(value) { return "0" }
	pids := f.Pid(value, args...)
	if pids != "0" {
		return pids
	}
	return f.RunBin(value, args...)
}
func (f *WsyShell) RunOnceSh(value string, args ...string) string {
	if Str.IsNull(value) { return "0" }
	pids := f.Pid(value, args...)
	if pids != "0" {
		return pids
	}
	return f.RunSh(value, args...)
}
func (f *WsyShell) NohupOnce(value string, args ...string) string {
	if Str.IsNull(value) { return "0" }
	pids := f.Pid(value, args...)
	if pids != "0" {
		return pids
	}
	return f.Nohup(value, args...)
}
//重启进程
func (f *WsyShell) Restart(value string,args ...string) string {
	if Str.IsNull(value) { return "0" }
	f.Kill(value, args...)
	pid := f.RunBin(value, args...)
	if pid == "0" {
		return "0"
	}
	return pid
}