package Wsy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type WsyTool struct{}

// ToMac 生成/格式化 MAC 地址（参数顺序无关）。
//
// 支持的参数类型：
// - string:
//   - 分隔符控制："-" / ":" / "null"（"null" 表示不使用分隔符，输出 12 位 hex）
//   - 模式控制：
//     "total"：仅返回将生成的总数，不输出 MAC 列表
//     "last" ：仅返回最后一个 MAC（顺序/范围时为末尾地址，随机时为最后生成的一条）
//   - 或者：有效 MAC（"xx:xx:xx:xx:xx:xx" / "xx-xx-xx-xx-xx-xx" / "xxxxxxxxxxxx"）
//   - 其它任意字符串一律视为非法，直接返回空串。
//
// - int / int64: 数量 count（生成多少条）；count==0 返回空串。
// - bool: upper（是否输出大写 HEX）
// - last: last（是否返回最后一个）
// 行为：
// - 未提供 start MAC：生成随机 MAC（本地管理地址，unicast），默认 1 条；count<0 按 1 处理。
// - 仅提供 start MAC：顺序递增生成，默认 100 条；可用 count 覆盖。last 时返回 start。
// - 提供 start + end MAC：生成范围内所有（或用 count 截断）；start>end 返回空串。last 时返回 end。
func (a *WsyTool) RanMac(args ...any) string {
	type mac6 = [6]byte
	parseMAC := func(in string) (mac mac6, ok bool) {
		s := strings.ToLower(strings.TrimSpace(in))
		if s == "" || strings.Contains(s, "*") {
			return mac, false
		}
		s = strings.ReplaceAll(strings.ReplaceAll(s, "-", ":"), ".", ":")
		if strings.Contains(s, ":") {
			p := strings.Split(s, ":")
			if len(p) != 6 {
				return mac, false
			}
			for i := 0; i < 6; i++ {
				if len(p[i]) != 2 {
					return mac, false
				}
				b, err := hex.DecodeString(p[i])
				if err != nil || len(b) != 1 {
					return mac, false
				}
				mac[i] = b[0]
			}
			return mac, true
		}
		if len(s) != 12 {
			return mac, false
		}
		b, err := hex.DecodeString(s)
		if err != nil || len(b) != 6 {
			return mac, false
		}
		copy(mac[:], b)
		return mac, true
	}
	toU64 := func(m mac6) uint64 {
		return uint64(m[0])<<40 | uint64(m[1])<<32 | uint64(m[2])<<24 | uint64(m[3])<<16 | uint64(m[4])<<8 | uint64(m[5])
	}
	sep, upper := ":", false
	var start, end *mac6
	count := int64(-1) // -1 未指定
	totalOnly := false
	lastOnly := false
	for _, a0 := range args {
		switch v := a0.(type) {
		case bool:
			upper = v
		case int:
			count = int64(v)
		case int64:
			count = v
		case string:
			s := strings.TrimSpace(v)
			switch strings.ToLower(s) {
			case "-", ":", "null":
				if s == "null" {
					sep = ""
				} else {
					sep = s
				}
				continue
			case "total":
				totalOnly = true
				continue
			case "last":
				lastOnly = true
				continue
			}
			if m, ok := parseMAC(v); ok {
				if start == nil {
					tmp := m
					start = &tmp
				} else if end == nil {
					tmp := m
					end = &tmp
				}
				continue
			}
			return ""
		}
	}
	format := func(v uint64) string {
		b0, b1, b2 := byte(v>>40), byte(v>>32), byte(v>>24)
		b3, b4, b5 := byte(v>>16), byte(v>>8), byte(v)
		if sep == "" {
			if upper {
				return fmt.Sprintf("%02X%02X%02X%02X%02X%02X", b0, b1, b2, b3, b4, b5)
			}
			return fmt.Sprintf("%02x%02x%02x%02x%02x%02x", b0, b1, b2, b3, b4, b5)
		}
		if upper {
			return fmt.Sprintf("%02X%s%02X%s%02X%s%02X%s%02X%s%02X", b0, sep, b1, sep, b2, sep, b3, sep, b4, sep, b5)
		}
		return fmt.Sprintf("%02x%s%02x%s%02x%s%02x%s%02x%s%02x", b0, sep, b1, sep, b2, sep, b3, sep, b4, sep, b5)
	}
	// 默认数量：随机=1；顺序(start)=100；范围(start+end)=全量（count=-1）
	if count == -1 {
		if start == nil {
			count = 1
		} else if end == nil {
			count = 100
		}
	}
	if count == 0 {
		if totalOnly {
			return "0"
		}
		if lastOnly {
			return ""
		}
		return ""
	}
	emitSeq := func(n int64, base uint64) string {
		if lastOnly {
			if n <= 0 {
				return ""
			}
			return format(base + uint64(n-1))
		}
		var sb strings.Builder
		for i := int64(0); i < n; i++ {
			if i != 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(format(base + uint64(i)))
		}
		return sb.String()
	}
	emitRand := func(n int64) string {
		if lastOnly {
			var last string
			for i := int64(0); i < n; i++ {
				var b [6]byte
				if _, err := rand.Read(b[:]); err != nil {
					return ""
				}
				b[0] = (b[0] | 0x02) & 0xFE
				last = format(toU64(b))
			}
			return last
		}
		var sb strings.Builder
		for i := int64(0); i < n; i++ {
			var b [6]byte
			if _, err := rand.Read(b[:]); err != nil {
				return ""
			}
			b[0] = (b[0] | 0x02) & 0xFE
			if i != 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(format(toU64(b)))
		}
		return sb.String()
	}
	// 随机
	if start == nil {
		if count < 0 {
			count = 1
		}
		if totalOnly {
			return fmt.Sprintf("%d", count)
		}
		return emitRand(count)
	}
	// 顺序/范围
	sv := toU64(*start)
	const max48 = (uint64(1) << 48) - 1
	if end != nil {
		ev := toU64(*end)
		if sv > ev {
			return ""
		}
		total := int64(ev - sv + 1)
		n := total
		if count > 0 && count < n {
			n = count
		} else if count == -1 && n > 10000 {
			n = 10000
		}
		if totalOnly {
			return fmt.Sprintf("%d", n)
		}
		if n <= 0 || sv+uint64(n-1) > max48 {
			return ""
		}
		return emitSeq(n, sv)
	}
	// 仅 start
	if count < 0 {
		count = 100
	}
	if totalOnly {
		return fmt.Sprintf("%d", count)
	}
	if count <= 0 || sv+uint64(count-1) > max48 {
		return ""
	}
	return emitSeq(count, sv)
}

//生成激活码-不重复
//al->数字+字母；bool true 时字母全大写（数字不变）
//mx->数字+字母；bool true 时字母全大写（与 al 字符集一致）
//zm->字母；bool true 时全大写
//sz->数字
//类型名大小写不敏感：AL、Mx、mX、ZM、SZ 等均可
//Random() 默认 al，6 位小写
//前缀含 % 时为占位符，每个 % 替换为随机体；无 % 则为 前缀+随机体
//  dev     -> dev123456   %-dev -> 123456-dev   %dev -> 123456dev   dev% -> dev123456   dev-% -> dev-123456
//Random("al",80,"12","dev",true) 示例：类型、条数、长度、前缀、大写开关

func (a *WsyTool) Random(args ...any) string {
	ranCodeTab := map[string][2]string{
		"al": {"0123456789abcdefghijklmnopqrstuvwxyz", "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		"mx": {"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		"zm": {"abcdefghijklmnopqrstuvwxyz", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		"sz": {"0123456789", "0123456789"},
	}
	mode, count, length, maxT, upper := "al", 1, 6, 0, false
	var digits, prefs []string
	for _, arg := range args {
		switch v := arg.(type) {
		case bool:
			upper = upper || v
		case int:
			count = v
		case int64:
			count = int(v)
		case string:
			s := strings.TrimSpace(v)
			if s == "" { continue }
			k := strings.ToLower(s)
			if _, ok := ranCodeTab[k]; ok { mode = k; continue }
			allDig := true
			for i := 0; i < len(s); i++ {
				if s[i] < '0' || s[i] > '9' { allDig = false; break }
			}
			if allDig { digits = append(digits, s) } else { prefs = append(prefs, s) }
		}
	}
	for i, d := range digits {
		if n, err := strconv.Atoi(d); err == nil {
			if i == 0 { length = n } else if i == 1 && n > 0 { maxT = n }
		}
	}
	prefix := ""
	if len(prefs) > 0 { prefix = prefs[len(prefs)-1] }
	if upper { prefix = strings.ToUpper(prefix) }
	if count < 0 { count = 1 }
	if count == 0 || length <= 0 { return "" }
	pair, ok := ranCodeTab[mode]
	if !ok { return "" }
	cs := pair[0]
	if upper { cs = pair[1] }
	if maxT <= 0 { maxT = count * 10000; if maxT < 10000 { maxT = 10000 } }
	randBody := func(n int) string {
		cl := len(cs)
		if n <= 0 || cl == 0 { return "" }
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil { return "" }
		var b strings.Builder
		b.Grow(n)
		for i := 0; i < n; i++ { b.WriteByte(cs[int(buf[i])%cl]) }
		return b.String()
	}
	var seen map[string]struct{}
	if count > 1 { seen = make(map[string]struct{}, count) }
	out := make([]string, 0, count)
	for t := 0; len(out) < count && t < maxT; t++ {
		body := randBody(length)
		if body == "" { return "" }
		code := prefix + body
		if strings.Contains(prefix, "%") { code = strings.ReplaceAll(prefix, "%", body) }
		if seen != nil {
			if _, dup := seen[code]; dup { continue }
			seen[code] = struct{}{}
		}
		out = append(out, code)
	}
	if len(out) != count { return "" }
	if count == 1 { return out[0] }
	return strings.Join(out, "\n")
}