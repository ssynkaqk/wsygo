# wsygo

Wsy 是一个面向 Go 的实用工具框架，封装常用能力：字符串、日期、HTTP、数据库、Redis、文件、上传、Shell、Gin 等。

## 安装

```bash
go get github.com/ssynkaqk/wsygo@latest
```

## 使用

```go
import Wsy "github.com/ssynkaqk/wsygo/wsygo"

func main() {
    // 示例：字符串、日期等通过全局命名空间调用
    _ = Wsy.Str
    _ = Wsy.Date
    _ = Wsy.Http
}
```

## 模块一览

| 模块 | 说明 |
|------|------|
| `Str` / `Date` / `Json` / `Map` / `Sum` | 字符串、日期、JSON、Map、数值工具 |
| `Http` / `Dns` / `TLS` / `Down` | HTTP、DNS、WebSocket、下载 |
| `DB` / `Redis` / `Cache` | 数据库、Redis、缓存 |
| `Fso` / `Upload` / `Shell` / `APP` | 文件、上传、进程、APK |
| `Gin` / `Token` / `Key` / `Role` | Web、Token、加解密、权限 |
| `Timer` / `Tool` / `Args` / `Html` / `Fanyi` | 定时器与其它辅助能力 |

## 要求

- Go 1.24+
