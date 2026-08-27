package Wsy

type ConfigMain struct {
	Key  string // 加密密钥
	Token string // 令牌密钥
	File string
	Logs bool
	LogsSave bool
	LogsFile string
	Version string
}

// 全局配置
var Set = ConfigMain{
	Version: "1.0.0",
	Key:  "123456",
	Token:"123456",
	File: "/opt/wsyos.ini",
	Logs: false,
	LogsSave: false,
	LogsFile: "",
}