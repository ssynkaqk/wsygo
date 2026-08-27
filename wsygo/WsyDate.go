package Wsy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)
type WsyDate struct{}

// ToLoc 返回业务时区，默认东八区 Asia/Shanghai；加载失败则 CST+8。
// zone 可选，规则同 ToZoneTime（如 "Asia/Tokyo"、"+08:00"）。
func (d WsyDate) ToLoc(zone ...string) *time.Location {
	locName := "Asia/Shanghai"
	if len(zone) > 0 && zone[0] != "" {
		if strings.HasPrefix(zone[0], "+") || strings.HasPrefix(zone[0], "-") {
			locName = "UTC" + zone[0]
		} else {
			locName = zone[0]
		}
	}
	loc, err := time.LoadLocation(locName)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// Auto 智能日期时间处理函数，根据参数数量和内容自动判断处理方式
//
// 示例:
// Wsy.Date.Time()                                    // "2024-04-05 15:30:25"
// Wsy.Date.Time("ymd")                              // "20240405"
// Wsy.Date.Time("YMDHISV")                          // "20250425092920123"
// Wsy.Date.Time("2025-04-04")                       // "2025-04-04"
// Wsy.Date.Time("2025-04-04", "Y-M-D")              // "2025-04-04"
// Wsy.Date.Time("2025-04-04 12:12:52", "YMDHIS")    // "20250404121252"
// Wsy.Date.Time("2025-04-04 12:12:52", "H:I:S")     // "12:12:52"
// Wsy.Date.Time("20250425092920123", "Y-M-D H:I:S.V") // "2025-04-25 09:29:20.123"
func (d WsyDate) ToTime(value ...string) string {
	loc := d.ToLoc()
	if len(value) == 0 {
		return time.Now().In(loc).Format("2006-01-02 15:04:05")
	}

	// 解析时间和毫秒
	parseTime := func(str string) (t time.Time, msec string, err error) {
		if strings.Contains(str, ".") {
			parts := strings.Split(str, ".")
			str, msec = parts[0], parts[1][:3]
		}
		// 处理数字格式
		matched, _ := regexp.MatchString(`^\d+$`, str)
		if matched {
			switch len(str) {
			case 8:  // 20250425
				t, err = time.ParseInLocation("20060102", str, loc)
				return t, msec, err
			case 14: // 20250425092920
				t, err = time.ParseInLocation("20060102150405", str, loc)
				return t, msec, err
			case 17: // 20250425092920123
				t, err = time.ParseInLocation("20060102150405", str[:14], loc)
				msec = str[14:17]
				return t, msec, err
			}
		}

		// 带显式时区用 Parse；否则按 ToLoc 东八区 ParseInLocation（避免日期里的 - 误判）
		hasTZ := strings.ContainsAny(str, "Zz")
		if !hasTZ {
			hasTZ, _ = regexp.MatchString(`(?i)\b(UTC|GMT|MST|PST|EST|CST|EDT|PDT|MDT|CDT)\b`, str)
		}
		if !hasTZ {
			hasTZ, _ = regexp.MatchString(`[+-]\d{2}:?\d{2}(:\d{2})?$`, strings.TrimSpace(str))
		}
		if !hasTZ {
			hasTZ, _ = regexp.MatchString(`[+-]\d{4}$`, strings.TrimSpace(str))
		}

		// 处理标准格式
		for _, layout := range []string{
			// RFC 标准格式
			time.RFC3339,                    // "2006-01-02T15:04:05Z07:00"
			time.RFC3339Nano,                // "2006-01-02T15:04:05.999999999Z07:00"
			time.RFC1123,                    // "Mon, 02 Jan 2006 15:04:05 MST"
			time.RFC1123Z,                   // "Mon, 02 Jan 2006 15:04:05 -0700"
			time.RFC822,                     // "02 Jan 06 15:04 MST"
			time.RFC822Z,                    // "02 Jan 06 15:04 -0700"
			time.RFC850,                     // "Monday, 02-Jan-06 15:04:05 MST"
			time.ANSIC,                      // "Mon Jan _2 15:04:05 2006"
			time.UnixDate,                   // "Mon Jan _2 15:04:05 MST 2006"
			time.RubyDate,                   // "Mon Jan 02 15:04:05 -0700 2006"
			
			// ISO 8601 格式
			"2006-01-02T15:04:05Z",         // UTC时间
			"2006-01-02T15:04:05.000Z",     // UTC时间带毫秒
			"2006-01-02T15:04:05.000000Z",  // UTC时间带微秒
			"2006-01-02T15:04:05.000000000Z", // UTC时间带纳秒
			"2006-01-02T15:04:05-07:00",    // ISO 8601 带时区
			"2006-01-02T15:04:05+07:00",    // ISO 8601 带正时区
			"2006-01-02T15:04:05.000-07:00", // ISO 8601 毫秒带时区
			"2006-01-02T15:04:05.000+07:00", // ISO 8601 毫秒带正时区
			"2006-01-02T15:04:05.000000-07:00", // ISO 8601 微秒带时区
			"2006-01-02T15:04:05.000000+07:00", // ISO 8601 微秒带正时区
			"2006-01-02T15:04:05",          // ISO 8601 无时区
			
			// 空格分隔格式
			"2006-01-02 15:04:05-07:00",    // 空格分隔带时区
			"2006-01-02 15:04:05+07:00",    // 空格分隔带正时区
			"2006-01-02 15:04:05.000-07:00", // 空格分隔带毫秒和时区
			"2006-01-02 15:04:05.000+07:00", // 空格分隔带毫秒和正时区
			"2006-01-02 15:04:05.000000-07:00", // 空格分隔带微秒和时区
			"2006-01-02 15:04:05.000000+07:00", // 空格分隔带微秒和正时区
			
			// Go 默认格式
			"2006-01-02 15:04:05 +0700 MST", // Go默认时间格式
			"2006-01-02 15:04:05 -0700 MST", // Go默认时间格式（负时区）
			"2006-01-02 15:04:05 +0700",     // 简化时区格式
			"2006-01-02 15:04:05 -0700",     // 简化负时区格式
			"2006-01-02 15:04:05.000 +0700 MST", // Go默认格式带毫秒
			"2006-01-02 15:04:05.000 -0700 MST", // Go默认格式带毫秒（负时区）
			
			// 斜杠分隔格式
			"2006/01/02T15:04:05-07:00",    // 斜杠分隔带时区
			"2006/01/02T15:04:05+07:00",    // 斜杠分隔带正时区
			"2006/01/02T15:04:05.000-07:00", // 斜杠分隔带毫秒和时区
			"2006/01/02T15:04:05.000+07:00", // 斜杠分隔带毫秒和正时区
			"2006/01/02 15:04:05-07:00",    // 斜杠分隔空格带时区
			"2006/01/02 15:04:05+07:00",    // 斜杠分隔空格带正时区
			"2006/01/02 15:04:05.000-07:00", // 斜杠分隔空格带毫秒和时区
			"2006/01/02 15:04:05.000+07:00", // 斜杠分隔空格带毫秒和正时区
			
			// 点分隔格式（欧洲常见）
			"02.01.2006 15:04:05-07:00",    // 欧洲日期格式带时区
			"02.01.2006 15:04:05+07:00",    // 欧洲日期格式带正时区
			"02.01.2006 15:04:05.000-07:00", // 欧洲日期格式带毫秒和时区
			"02.01.2006 15:04:05.000+07:00", // 欧洲日期格式带毫秒和正时区
			
			// 标准格式（无时区）
			"2006-01-02 15:04:05",          // 标准格式
			"2006-01-02 15:04:05.000",      // 标准格式带毫秒
			"2006-01-02 15:04:05.000000",   // 标准格式带微秒
			"2006-01-02 15:04:05.000000000", // 标准格式带纳秒
			"2006/01/02 15:04:05",          // 斜杠分隔标准格式
			"2006/01/02 15:04:05.000",      // 斜杠分隔标准格式带毫秒
			"02.01.2006 15:04:05",          // 欧洲日期标准格式
			"02.01.2006 15:04:05.000",      // 欧洲日期标准格式带毫秒
			
			// 仅日期格式
			"2006-01-02",                   // 仅日期
			"2006/01/02",                   // 斜杠分隔仅日期
			"02.01.2006",                   // 欧洲日期格式仅日期
			"02-01-2006",                   // 欧洲日期格式仅日期（横线）
			
			// 特殊格式
			"Jan 02, 2006 15:04:05",        // 英文月份格式
			"Jan 02, 2006 15:04:05 -0700",  // 英文月份格式带时区
			"Jan 02, 2006 15:04:05 +0700",  // 英文月份格式带正时区
			"January 02, 2006 15:04:05",    // 英文全月份格式
			"January 02, 2006 15:04:05 -0700", // 英文全月份格式带时区
			"January 02, 2006 15:04:05 +0700", // 英文全月份格式带正时区
		} {
			if hasTZ {
				t, err = time.Parse(layout, str)
			} else {
				t, err = time.ParseInLocation(layout, str, loc)
			}
			if err == nil {
				return t, msec, nil
			}
		}
		return time.Time{}, msec, fmt.Errorf("无法解析日期格式")
	}

	// 格式化时间
	format := func(t time.Time, format string, msec string) string {
		// 检查是否为简单格式
		if !strings.ContainsAny(format, "-/.: ") {
			format = strings.ToUpper(format)
			result := strings.NewReplacer(
				"Y", t.Format("2006"),
				"M", t.Format("01"),
				"D", t.Format("02"),
				"H", t.Format("15"),
				"I", t.Format("04"),
				"S", t.Format("05"),
				"V", msec,
			).Replace(format)

			// 返回纯数字结果
			var nums strings.Builder
			for _, c := range result {
				if c >= '0' && c <= '9' {
					nums.WriteRune(c)
				}
			}
			return nums.String()
		}

		// 处理带分隔符的格式
		format = strings.ToUpper(format)
		return strings.NewReplacer(
			"Y", t.Format("2006"),
			"M", t.Format("01"),
			"D", t.Format("02"),
			"H", t.Format("15"),
			"I", t.Format("04"),
			"S", t.Format("05"),
			"V", msec,
		).Replace(format)
	}

	// 处理单个参数
	if len(value) == 1 {
		arg := value[0]
		now := time.Now().In(loc)

		// 检查是否为格式字符串
		if matched, _ := regexp.MatchString(`^[YMDHISVymdhisv\-\s\:\.\,\/]+$`, arg); matched {
			return format(now, arg, fmt.Sprintf("%03d", now.Nanosecond()/1e6))
		}

		// 解析日期时间
		if t, msec, err := parseTime(arg); err == nil {
			if strings.Contains(arg, ":") || len(arg) > 8 {
				if msec != "" {
					return t.Format("2006-01-02 15:04:05") + "." + msec
				}
				return t.Format("2006-01-02 15:04:05")
			}
			return t.Format("2006-01-02")
		}
		return arg
	}

	// 处理两个参数
	if t, msec, err := parseTime(value[0]); err == nil {
		return format(t, value[1], msec)
	}
	return value[0]
}
// FormatTime 格式化时间对象，内部辅助方法
func (d WsyDate) FormatTime(t time.Time, format string) string {
	format = strings.ToLower(format)
	simpleFormats := map[string]string{
		"y":      "2006",           // 年
		"m":      "01",             // 月
		"d":      "02",             // 日
		"h":      "15",             // 时 (24小时)
		"i":      "04",             // 分
		"s":      "05",             // 秒
		"ym":     "200601",         // 年月
		"ymd":    "20060102",       // 年月日
		"ymdh":   "2006010215",     // 年月日时
		"ymdhi":  "200601021504",   // 年月日时分
		"ymdhis": "20060102150405", // 年月日时分秒
		"his":    "150405",         // 时分秒
		"hi":     "1504",           // 时分
		"year":   "2006",           // 年（全称）
		"month":  "01",             // 月（全称）
		"day":    "02",             // 日（全称）
		"hour":   "15",             // 时（全称）
		"minute": "04",             // 分（全称）
		"second": "05",             // 秒（全称）
	}
	if goFormat, ok := simpleFormats[format]; ok {
		return t.Format(goFormat)
	}
	replacements := map[string]string{
		"Y": "2006", "y": "2006", // 统一为4位年份
		"m": "01",            // 2位月份
		"d": "02",            // 2位日期
		"H": "15", "h": "15", // 统一使用24小时制
		"i": "04", // 分钟
		"s": "05", // 秒钟
	}
	goFormat := format
	for short, long := range replacements {
		goFormat = strings.ReplaceAll(goFormat, short, long)
	}
	return t.Format(goFormat)
}

// Valid 判断字符串是否为常见的日期或时间格式
// 支持格式：yyyy-mm-dd、yyyy/mm/dd、yyyy-mm-dd hh:mm:ss、yyyy/mm/dd hh:mm:ss
func (d WsyDate) Valid(date string) bool {
	patterns := []string{
		`^\d{4}-\d{2}-\d{2}$`,
		`^\d{4}/\d{2}/\d{2}$`,
		`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`,
		`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}$`,
	}
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, date)
		if matched {
			return true
		}
	}
	return false
}

// ToAbout 获取相对时间并以指定格式输出，支持毫秒精度
//
// 示例:
//	Lin.Date.ToAbout("-1", "date", "YMD")      // 输出: 昨天日期，如"20240509"
//	Lin.Date.ToAbout("0", "date", "Y-M-D")     // 输出: 今天日期，如"2024-05-10"
//	Lin.Date.ToAbout("-3600", "time", "YMDHIS") // 输出: 1小时前时间，如"20240510143025"
//	Lin.Date.ToAbout("86400", "time", "Y-M-D H:I:S.V") // 输出: 1天后时间，如"2024-05-11 15:30:25.123"
func (d WsyDate) ToAbout(offset string, mode ...string) string {
	var timeMode, format string = "date", "Y-M-D"
	if len(mode) > 0 && mode[0] != "" {
		timeMode = strings.ToLower(mode[0])
	}
	if len(mode) > 1 && mode[1] != "" {
		format = mode[1]
	}
	offsetValue, err := strconv.Atoi(offset)
	if err != nil {
		return d.ToTime(format)
	}
	now := time.Now().In(d.ToLoc())
	var targetTime time.Time
	if timeMode == "time" {
		targetTime = now.Add(time.Duration(offsetValue) * time.Second)
	} else {
		targetTime = now.AddDate(0, 0, offsetValue)
	}
	needsMilliseconds := strings.Contains(strings.ToUpper(format), "V")
	if needsMilliseconds {
		dateStr := targetTime.Format("2006-01-02 15:04:05") + fmt.Sprintf(".%03d", targetTime.Nanosecond()/1e6)
		return d.ToTime(dateStr, format)
	}
	return d.ToTime(targetTime.Format("2006-01-02 15:04:05"), format)
}

// ToZoneTime 示例：
//   t := Wsy.Date.ToZoneTime(1718000000)
//   // 输出: "2024-06-10 10:13:20"（默认 Asia/Shanghai）
//   t2 := Wsy.Date.ToZoneTime("1718000000", "Asia/Tokyo")
//   // 输出: "2024-06-10 11:13:20"（东京时区）
//   t3 := Wsy.Date.ToZoneTime(1718000000, "+08:00")
//   // 输出: "2024-06-10 10:13:20"（UTC+8）
//   t4 := Wsy.Date.ToZoneTime(1718000000, "-05:00")
//   // 输出: "2024-06-09 21:13:20"（UTC-5）
//
func (d WsyDate) ToZoneTime(bootTime interface{}, zone ...string) string {
    var sec int64
    switch v := bootTime.(type) {
    case int64:
        sec = v
    case uint64:
        sec = int64(v)
    case float64:
        sec = int64(v)
    case string:
        if n, err := strconv.ParseInt(v, 10, 64); err == nil {
            sec = n
        } else {
            return ""
        }
    default:
        return ""
    }
    return time.Unix(sec, 0).In(d.ToLoc(zone...)).Format("2006-01-02 15:04:05")
}

// ToUnixTime 示例：
//   ts := Wsy.Date.ToUnixTime("2024-06-10 10:13:20")
//   // 输出: 1717985600 （默认 Asia/Shanghai 东八区）
//
//   ts2 := Wsy.Date.ToUnixTime("2024-06-10 10:13:20", "Asia/Tokyo")
//   // 输出: 1717977600 （东京时区对应的时间戳）
//
//   ts3 := Wsy.Date.ToUnixTime("2024-06-10 10:13:20", "+08:00")
//   // 输出: 1717985600 （UTC+8 时区）
//
//   ts4 := Wsy.Date.ToUnixTime("2024-06-10 10:13:20", "-05:00")
//   // 输出: 1718008800 （UTC-5 时区）
//
//   ts5 := Wsy.Date.ToUnixTime("2024-06-10 10:13:20", "America/New_York")
//   // 输出: 1718008800 （纽约时区对应的时间戳）
//

func (d WsyDate) ToUnixTime(dateTime string, zone ...string) int64 {
    loc := d.ToLoc(zone...)
    t, err := time.ParseInLocation("2006-01-02 15:04:05", d.ToTime(dateTime), loc)
    if err != nil {
        return 0
    }
    return t.Unix()
}



func (d WsyDate) Now() string {
	return d.ToTime()
}

// ToStdTime 转 time.Time；空参返回当前时间。可选第二参 mode（不区分大小写）：
// auto 默认 | full/date/datetime 不要仅时分秒回退 | time/clock 仅 HH:MM[:SS] 拼当天
func (d WsyDate) ToStdTime(value ...string) time.Time {
	loc := d.ToLoc()
	if len(value) == 0 || strings.TrimSpace(value[0]) == "" {
		return time.Now().In(loc)
	}
	s := strings.TrimSpace(value[0])
	u, f, clk := true, true, true // unix、ToTime 日期、仅时分秒
	if len(value) > 1 {
		switch strings.ToLower(strings.TrimSpace(value[1])) {
			case "full", "date", "datetime":
				clk = false
			case "time", "clock":
				u, f = false, false
		}
	}
	if u {
		if ok, _ := regexp.MatchString(`^\d+$`, s); ok {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				if n > 1_000_000_000_000 {
					return time.UnixMilli(n)
				}
				if n > 0 {
					return time.Unix(n, 0)
				}
			}
		}
	}
	if f {
		if ts := d.ToTime(s); ts != "" {
			for _, lay := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
				if tm, err := time.ParseInLocation(lay, ts, loc); err == nil {
					return tm
				}
			}
		}
	}
	if clk {
		if ok, _ := regexp.MatchString(`^\d{1,2}:\d{2}(:\d{2})?$`, s); ok {
			now := time.Now().In(loc)
			for _, lay := range []string{"15:04:05", "15:04"} {
				if tt, err := time.ParseInLocation(lay, s, loc); err == nil {
					return time.Date(now.Year(), now.Month(), now.Day(), tt.Hour(), tt.Minute(), tt.Second(), 0, loc)
				}
			}
		}
	}
	return time.Time{}
}

// GetTimeStamp 获取当前时间戳（东八区）
// 示例：
// Lin.Date.GetTimeStamp() // 输出当前时间戳
func (d WsyDate) GetTimeStamp() int64 {
	return time.Now().Unix()
}

// ToDay 计算天数（string 自动识别）
//
//	Wsy.Date.ToDay()                      // 当月天数
//	Wsy.Date.ToDay("12")                  // 当年 12 月天数
//	Wsy.Date.ToDay("2024")                // 2024 年全年天数（366）
//	Wsy.Date.ToDay("2024")                // 2024 年全年天数（365）
//	Wsy.Date.ToDay("2024-12")             // 2024 年 12 月天数
//	Wsy.Date.ToDay("2024-12-12")          // 2024 年 12 月天数（日可为 1-2 位）
//	Wsy.Date.ToDay("2024-12-12 12:52:11") // 2024 年 12 月天数
func (d WsyDate) ToDay(value ...string) int {
	now := time.Now().In(d.ToLoc())
	y, m, yearOnly := now.Year(), int(now.Month()), false

	s := ""
	if len(value) > 0 {
		s = strings.TrimSpace(value[0])
	}
	if s != "" {
		y, m, yearOnly = 0, 0, false
		if i := strings.IndexByte(s, ' '); i > 0 {
			s = strings.TrimSpace(s[:i])
		}
		s = strings.ReplaceAll(s, "/", "-")

		if ok, _ := regexp.MatchString(`^\d+$`, s); ok {
			n, _ := strconv.Atoi(s)
			switch len(s) {
			case 1, 2:
				if n < 1 || n > 12 {
					return 0
				}
				y, m = now.Year(), n
			case 4:
				y, yearOnly = n, true
			default:
				return 0
			}
		} else if ok, _ := regexp.MatchString(`^\d{4}-\d{1,2}(-\d{1,2})?$`, s); ok {
			p := strings.Split(s, "-")
			y, _ = strconv.Atoi(p[0])
			m, _ = strconv.Atoi(p[1])
		} else if tm := d.ToStdTime(s); !tm.IsZero() {
			y, m = tm.Year(), int(tm.Month())
		} else {
			return 0
		}
	}

	if y <= 0 {
		return 0
	}
	if yearOnly {
		return time.Date(y, 12, 31, 0, 0, 0, 0, time.UTC).YearDay()
	}
	if m < 1 || m > 12 {
		return 0
	}
	return time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC).Day()
}

// Online 检查是否在线（基于时间差）
// 示例：
// Lin.Date.Online("2024-03-14 15:45:00") // 如果当前时间与给定时间相差不超过15分钟，返回true
// Lin.Date.Online("2024-03-14 15:45:00", 30) // 如果当前时间与给定时间相差不超过30分钟，返回true
func (d WsyDate) Online(startTime string, value ...int) bool {
	if startTime == "" {
		return false
	}
	minutes := 15
	if len(value) > 0 {
		minutes = value[0]
	}
	startTimestamp := d.ToUnixTime(startTime)
	if startTimestamp == 0 {
		return false
	}
	currentTimestamp := d.GetTimeStamp()
	diffSeconds := currentTimestamp - startTimestamp
	thresholdSeconds := int64(minutes * 60)
	return diffSeconds <= thresholdSeconds && diffSeconds >= -thresholdSeconds
}
// ToBothTime 比较两个时间，第一个时间小于等于第二个时间返回true
// 示例：
// Lin.Date.ToBothTime("2024-03-14 10:00:00", "2024-03-15 10:00:00") // 输出: true
// Lin.Date.ToBothTime("2024-03-16 10:00:00", "2024-03-15 10:00:00") // 输出: false
func (d WsyDate) ToBothTime(time1 string, time2 string) bool {
	if d.Valid(time1) == false || d.Valid(time2) == false {
		return false
	}
	return d.ToUnixTime(time1) <= d.ToUnixTime(time2)
}
// ToWithTime 判断时间是否在指定时间范围内
// 示例：
// Lin.Date.ToWithTime(
//
//	"2024-03-10 10:00:00",  // 开始时间
//	"2024-03-17 14:00:00",  // 结束时间
//	"2024-03-14 12:00:00",  // 当前时间（可选，默认为当前时间）
//
// ) // 输出: true，因为当前时间在开始和结束时间之间
func (d WsyDate) ToWithTime(startTime string, endTime string, currentTime ...string) bool {
	nowTimeStr := ""
	if d.Valid(startTime) == false || d.Valid(endTime) == false {
		return false
	}
	if len(currentTime) > 0 {
		nowTimeStr = currentTime[0]
		if d.Valid(nowTimeStr) == false {
			return false
		}
	} else {
		nowTimeStr = d.Now()
	}
	nowTimestamp := d.ToUnixTime(nowTimeStr)
	startTimestamp := d.ToUnixTime(startTime)
	endTimestamp := d.ToUnixTime(endTime)
	return nowTimestamp >= startTimestamp && nowTimestamp <= endTimestamp
}

// OnlineText 将秒数或时间转换为可读的天-小时-分钟-秒格式
// 示例：
// OnlineText("86400");      // 输出: "1天" 
// OnlineText("90300");      // 输出: "1天1小时5分钟" 
// OnlineText("90306");      // 输出: "1天1小时5分钟6秒" 
// OnlineText("2025-05-05 12:12:12");      // 输出: "1天1小时5分钟6秒" 当前时间和传入计算
func (d WsyDate) OnlineText(value string) string {
	var totalSeconds int64
	if num, err := strconv.ParseInt(value, 10, 64); err == nil {
		totalSeconds = num
	} else {
		if d.Valid(value) {
			inputTimestamp := d.ToUnixTime(value)
			if inputTimestamp > 0 {
				currentTimestamp := d.GetTimeStamp()
				totalSeconds = currentTimestamp - inputTimestamp
				if totalSeconds < 0 {
					totalSeconds = -totalSeconds
				}
			}
		} else {
			return ""
		}
	}
	secondsInDay := int64(86400)
	secondsInHour := int64(3600)
	secondsInMinute := int64(60)
	days := totalSeconds / secondsInDay
	hours := (totalSeconds % secondsInDay) / secondsInHour
	minutes := (totalSeconds % secondsInHour) / secondsInMinute
	remainingSeconds := totalSeconds % secondsInMinute
	var result strings.Builder
	if days > 0 {
		result.WriteString(fmt.Sprintf("%d天", days))
	}
	if hours > 0 {
		result.WriteString(fmt.Sprintf("%d小时", hours))
	}
	if minutes > 0 {
		result.WriteString(fmt.Sprintf("%d分钟", minutes))
	}
	if remainingSeconds > 0 {
		result.WriteString(fmt.Sprintf("%d秒", remainingSeconds))
	}
	return result.String()
}
//---------------------------------------------------------------------------------时间在线位图------------------------------------


// RunTimeToInit 按间隔（分钟）返回 Len（槽位数）与 Sec（每槽秒数），默认 10
// interval 为正整数且能整除 1440（整日等分）则采用；否则（≤0、>1440、有余数）退回 10，避免除零与半天槽
//
//	Len, Sec := Wsy.Date.RunTimeToInit()   // 144, 600
//	Len, Sec := Wsy.Date.RunTimeToInit(12) // 120, 720（12 分钟一档）
func (d WsyDate) RunTimeToInit(interval ...int) (Len int, Sec int) {
	runTimeDayMin := 24 * 60 // 一日总分钟数
	min := 10
	if len(interval) > 0 {
		v := interval[0]
		if v > 0 && v <= runTimeDayMin && runTimeDayMin%v == 0 {
			min = v
		}
	}
	return runTimeDayMin / min, min * 60
}
// RunTimeToNum 规范为 0/1 位图，非法字符转 0；interval 可选，默认 10
func (d WsyDate) RunTimeToNum(key string, interval ...int) string {
	bootLen, _ := d.RunTimeToInit(interval...)
	s := strings.Map(func(r rune) rune {
		if r == '1' {
			return '1'
		}
		return '0'
	}, strings.TrimSpace(key))
	n := len(s)
	if n > bootLen {
		return s[:bootLen]
	}
	return s + strings.Repeat("0", bootLen-n)
}

// RunTimeToNew 将在线位图从 fromMin 一档转为 toMin 一档（二者均须能整除一日 1440 分钟，否则返回 toMin 档全 0）。
// 先按 fromMin 把每位铺到下属分钟，再按 toMin 聚合成新位：任一分钟为 1 则该 toMin 槽为 1（细→粗为 OR；粗→细为复制铺展）。
// 例：5 分钟 288 位 → 10 分钟 144 位，每连续 2 个 5 分槽合并为 1 个 10 分槽。
func (d WsyDate) RunTimeToNew(key string, fromMin, toMin int) string {
	const dayMin = 24 * 60
	ok := func(v int) bool { return v > 0 && v <= dayMin && dayMin%v == 0 }
	toLen, _ := d.RunTimeToInit(toMin)
	if !ok(fromMin) || !ok(toMin) {
		return strings.Repeat("0", toLen)
	}
	src := d.RunTimeToNum(key, fromMin)
	minute := make([]byte, dayMin)
	for m := 0; m < dayMin; m++ {
		si := m / fromMin
		if si < len(src) && src[si] == '1' {
			minute[m] = '1'
		} else {
			minute[m] = '0'
		}
	}
	out := make([]byte, toLen)
	for j := 0; j < toLen; j++ {
		one := false
		for k := 0; k < toMin; k++ {
			if minute[j*toMin+k] == '1' {
				one = true
				break
			}
		}
		if one {
			out[j] = '1'
		} else {
			out[j] = '0'
		}
	}
	return string(out)
}

// RunTimeToIndex 通过时间计算出槽位序号，如 "2026-05-15 17:06:46"  得位置103，因为17*60+6=1026，1026/10=102，102+1=103
func (d WsyDate) RunTimeToIndex(at string, interval ...int) int {
	at = strings.TrimSpace(at)
	if at == "" {
		return 0
	}
	t := d.ToStdTime(at)
	if t.IsZero() {
		return 0
	}
	bootLen, bootSec := d.RunTimeToInit(interval...)
	slot := (t.Hour()*60 + t.Minute()) / (bootSec / 60)
	if slot < 0 || slot >= bootLen {
		return 0
	}
	return slot + 1
}
// RunTimeToSet 将槽位设为 value（0 或 1）；pos 为 int 槽位，或 string（纯数字槽位 / 标准时间）
func (d WsyDate) RunTimeToSet(key string, pos interface{}, value int, interval ...int) string {
	key = d.RunTimeToNum(key, interval...)
	var idx int
	switch v := pos.(type) {
		case int:
			idx = v
		case string:
			s := strings.TrimSpace(v)
			if s == "" {
				return key
			}
			L, _ := d.RunTimeToInit(interval...)
			if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= L && !strings.ContainsAny(s, "-:/") {
				idx = n
			} else {
				idx = d.RunTimeToIndex(s, interval...)
			}
		default:
			return key
	}
	if idx < 1 || idx > len(key) || (value != 0 && value != 1) {
		return key
	}
	b := []byte(key)
	b[idx-1] = byte('0' + value)
	return string(b)
}
// RunTimeToMap 位图转 map（时间键 → 0/1）。opt 顺序任意：string 基准日；int/整值 float64 为间隔分钟；bool==false 则键仅 "15:04:05"
func (d WsyDate) RunTimeToMap(key string, opt ...any) map[string]int {
	var date string
	var iv []int
	timeOnly := false
	for _, a := range opt {
		switch v := a.(type) {
			case bool:
				timeOnly = !v
			case int:
				iv = []int{v}
			case float64:
				if v == float64(int(v)) {
					iv = []int{int(v)}
				}
			case string:
				if s := strings.TrimSpace(v); s != "" {
					date = s
				}
		}
	}
	key = d.RunTimeToNum(key, iv...)
	_, sec := d.RunTimeToInit(iv...)
	step := sec / 60
	loc := d.ToLoc()
	n := time.Now().In(loc)
	day0 := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	if ds := strings.TrimSpace(date); ds != "" {
		if t, err := time.ParseInLocation("2006-01-02", d.ToTime(ds, "Y-M-D"), loc); err == nil {
			day0 = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		}
	}
	layout := "2006-01-02 15:04:05"
	if timeOnly {
		layout = "15:04:05"
	}
	m := make(map[string]int, len(key))
	for i := range key {
		ts := day0.Add(time.Duration(i*step) * time.Minute)
		m[ts.Format(layout)] = int(key[i] - '0')
	}
	return m
}
// RunTimeToJson 调用 RunTimeToMap 再 Json.Encode，返回 JSON 字符串
func (d WsyDate) RunTimeToJson(key string, opt ...any) string {
	return Json.Encode(d.RunTimeToMap(key, opt...))
}
// RunTime 将位图中指定时刻或槽位设为 1；除 key 外参数可选，顺序任意：string 为时刻/槽位序号，int（及整值 float64）为间隔分钟，同 RunTimeToInit
func (d WsyDate) SetRunTime(key string, opt ...any) string {
	var at string
	var iv []int
	for _, a := range opt {
		switch v := a.(type) {
			case int:
				iv = []int{v}
			case string:
				if s := strings.TrimSpace(v); s != "" {
					at = s
				}
		}
	}
	return d.RunTimeToSet(key, strings.TrimSpace(at), 1, iv...)
}

// RunTimeToFix 修正在线位图 val2（右）相对 val1（左）：左 0 且右 1 时改为 0；左 1 且右 0 时不改；其余保留 val2。
// interval 可选，同 RunTimeToInit；val1 为空时返回该间隔下的标准全 0 位图。
func (d WsyDate) RunTimeToFix(val1, val2 string, interval ...int) string {
	if val1 == "" {
		return d.RunTimeToNum("", interval...)
	}
	val1 = d.RunTimeToNum(val1, interval...)
	val2 = d.RunTimeToNum(val2, interval...)
	result := make([]byte, len(val1))
	for i := 0; i < len(val1); i++ {
		if val1[i] == '0' && val2[i] == '1' {
			result[i] = '0'
		} else if val2[i] == '0' || val2[i] == '1' {
			result[i] = val2[i]
		} else {
			result[i] = '0'
		}
	}
	return string(result)
}