package Wsy

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type WsySum struct{}

// Add 计算两个数的和
// 示例: result := Lin.Sum.Add("10.5", "20.3") // 返回 "30.80"
func (s WsySum) Add(a, b string) string {
	valA, errA := strconv.ParseFloat(a, 64)
	valB, errB := strconv.ParseFloat(b, 64)

	if errA != nil || errB != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", errA, errB)
		return "0.00"
	}

	result := valA + valB
	return fmt.Sprintf("%.2f", result)
}

// Subtract 计算两个数的差
// 示例: result := Lin.Sum.Subtract("30.5", "10.3") // 返回 "20.20"
func (s WsySum) Subtract(a, b string) string {
	valA, errA := strconv.ParseFloat(a, 64)
	valB, errB := strconv.ParseFloat(b, 64)

	if errA != nil || errB != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", errA, errB)
		return "0.00"
	}

	result := valA - valB
	return fmt.Sprintf("%.2f", result)
}

// Multiply 计算两个数的乘积
// 示例: result := Lin.Sum.Multiply("10.5", "2") // 返回 "21.00"
func (s WsySum) Multiply(a, b string) string {
	valA, errA := strconv.ParseFloat(a, 64)
	valB, errB := strconv.ParseFloat(b, 64)

	if errA != nil || errB != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", errA, errB)
		return "0.00"
	}

	result := valA * valB
	return fmt.Sprintf("%.2f", result)
}

// Divide 计算两个数的商
// 示例: result := Lin.Sum.Divide("21", "7") // 返回 "3.00"
func (s WsySum) Divide(a, b string) string {
	valA, errA := strconv.ParseFloat(a, 64)
	valB, errB := strconv.ParseFloat(b, 64)

	if errA != nil || errB != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", errA, errB)
		return "0.00"
	}

	if valB == 0 {
		Logs("ERROR", "SUM", "除数不能为零")
		return "0.00"
	}

	result := valA / valB
	return fmt.Sprintf("%.2f", result)
}

// Percentage 计算百分比
// 参数:
//   - total: 总金额
//   - percent: 百分比值（不带%符号）
//
// 示例: result := Lin.Sum.Percentage("100", "15") // 返回 "15.00" (100的15%)
func (s WsySum) Percentage(total, percent string) string {
	valTotal, errTotal := strconv.ParseFloat(total, 64)
	valPercent, errPercent := strconv.ParseFloat(percent, 64)

	if errTotal != nil || errPercent != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", errTotal, errPercent)
		return "0.00"
	}

	result := valTotal * valPercent / 100.0
	return fmt.Sprintf("%.2f", result)
}

// Round 四舍五入到指定小数位
// 参数:
//   - value: 要四舍五入的值
//   - places: 保留的小数位数
//
// 示例: result := Lin.Sum.Round("123.456", 2) // 返回 "123.46"
func (s WsySum) Round(value string, places int) string {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v", err)
		return "0.00"
	}

	shift := math.Pow(10, float64(places))
	result := math.Round(val*shift) / shift

	format := fmt.Sprintf("%%.%df", places)
	return fmt.Sprintf(format, result)
}

// Ratio 金额计算
// 参数:
//   - total: 总金额
//   - ratio: 比例/百分比
//   - operator: 可选参数，运算符:
//   - "%": 减去比例部分（默认）
//   - "+": 加上比例部分
//   - "-": 只返回比例部分
//   - "*": 直接乘法计算
//   - "/": 直接除法计算
//   - precision: 可选参数，指定结果保留的小数位数，不指定时保持完整精度
//
// 示例：
//   - Ratio("100", "10") // 默认使用"%"运算符，返回 "90"
//   - Ratio("3.67", "5", "-") // 返回 "0.1835"
//   - Ratio("3.67", "5", "-", 2) // 返回 "0.18"
func (s WsySum) Ratio(total, ratio string, args ...interface{}) string {
	// 解析输入参数
	totalVal, errTotal := strconv.ParseFloat(total, 64)
	ratioVal, errRatio := strconv.ParseFloat(ratio, 64)

	if errTotal != nil || errRatio != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", errTotal, errRatio)
		return "0.00"
	}

	// 设置默认值
	op := "%"
	precision := -1 // 默认保持完整精度

	// 解析可选参数
	if len(args) > 0 {
		if opStr, ok := args[0].(string); ok && opStr != "" {
			op = opStr
		}
	}

	if len(args) > 1 {
		if precInt, ok := args[1].(int); ok {
			precision = precInt
		}
	}

	// 计算结果
	var result float64
	switch op {
	case "+": // 加上比例部分
		result = totalVal + (totalVal * ratioVal / 100.0)
	case "-": // 只返回比例部分
		result = totalVal * ratioVal / 100.0
	case "*": // 直接乘法
		result = totalVal * ratioVal
	case "/": // 直接除法
		if ratioVal == 0 {
			Logs("ERROR", "SUM", "除数不能为零")
			return "0.00"
		}
		result = totalVal / ratioVal
	default: // "%", 减去比例部分
		result = totalVal - (totalVal * ratioVal / 100.0)
	}

	// 格式化结果
	if precision < 0 {
		return fmt.Sprintf("%v", result)
	}

	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Sum 计算一组数值的总和
// 示例: result := Lin.Sum.Sum("10", "20", "30") // 返回 "60.00"
func (s WsySum) Sum(values ...string) string {
	var total float64 = 0

	for _, value := range values {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			Logs("WARN", "SUM", "跳过无效数值: %s", value)
			continue
		}
		total += val
	}

	return fmt.Sprintf("%.2f", total)
}

// Average 计算一组数值的平均值
// 示例: result := Lin.Sum.Average("10", "20", "30") // 返回 "20.00"
func (s WsySum) Average(values ...string) string {
	if len(values) == 0 {
		return "0.00"
	}

	var total float64 = 0
	validCount := 0

	for _, value := range values {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			Logs("WARN", "SUM", "跳过无效数值: %s", value)
			continue
		}
		total += val
		validCount++
	}

	if validCount == 0 {
		return "0.00"
	}

	result := total / float64(validCount)
	return fmt.Sprintf("%.2f", result)
}

// Min 返回一组数值中的最小值
// 示例: result := Lin.Sum.Min("30", "10", "20") // 返回 "10.00"
func (s WsySum) Min(values ...string) string {
	if len(values) == 0 {
		return "0.00"
	}

	var min *float64 = nil

	for _, value := range values {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			Logs("WARN", "SUM", "跳过无效数值: %s", value)
			continue
		}

		if min == nil || val < *min {
			min = &val
		}
	}

	if min == nil {
		return "0.00"
	}

	return fmt.Sprintf("%.2f", *min)
}

// Max 返回一组数值中的最大值
// 示例: result := Lin.Sum.Max("10", "30", "20") // 返回 "30.00"
func (s WsySum) Max(values ...string) string {
	if len(values) == 0 {
		return "0.00"
	}

	var max *float64 = nil

	for _, value := range values {
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			Logs("WARN", "SUM", "跳过无效数值: %s", value)
			continue
		}

		if max == nil || val > *max {
			max = &val
		}
	}

	if max == nil {
		return "0.00"
	}

	return fmt.Sprintf("%.2f", *max)
}

// FormatMoney 格式化金额为货币格式
// 参数:
//   - amount: 金额
//   - currency: 货币符号 (可选，默认为空)
//   - useThousandsSeparator: 是否使用千位分隔符 (可选，默认为true)
//
// 示例:
//   - FormatMoney("1234.56") // 返回 "1,234.56"
//   - FormatMoney("1234.56", "￥") // 返回 "￥1,234.56"
//   - FormatMoney("1234.56", "￥", false) // 返回 "￥1234.56"
func (s WsySum) FormatMoney(amount string, options ...interface{}) string {
	// 解析金额
	val, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		Logs("ERROR", "SUM", "金额解析失败: %v", err)
		return amount
	}

	// 解析可选参数
	currency := ""
	useThousandsSeparator := true

	if len(options) > 0 {
		if cur, ok := options[0].(string); ok {
			currency = cur
		}
	}

	if len(options) > 1 {
		if sep, ok := options[1].(bool); ok {
			useThousandsSeparator = sep
		}
	}

	// 格式化金额为两位小数
	amountStr := fmt.Sprintf("%.2f", val)

	// 分离整数部分和小数部分
	parts := strings.Split(amountStr, ".")
	intPart := parts[0]
	decPart := ""
	if len(parts) > 1 {
		decPart = parts[1]
	}

	// 添加千位分隔符
	if useThousandsSeparator {
		intLen := len(intPart)
		if intLen > 3 {
			var sb strings.Builder

			// 处理负数前缀
			startIdx := 0
			if intPart[0] == '-' {
				sb.WriteByte('-')
				startIdx = 1
			}

			// 添加千位分隔符
			remainder := (intLen - startIdx) % 3
			if remainder > 0 {
				sb.WriteString(intPart[startIdx : startIdx+remainder])
				if intLen-startIdx > 3 {
					sb.WriteByte(',')
				}
			}

			for i := startIdx + remainder; i < intLen; i += 3 {
				end := i + 3
				if end > intLen {
					end = intLen
				}

				sb.WriteString(intPart[i:end])
				if end < intLen {
					sb.WriteByte(',')
				}
			}

			intPart = sb.String()
		}
	}

	// 重组金额字符串
	result := intPart
	if decPart != "" {
		result += "." + decPart
	}

	// 添加货币符号
	if currency != "" {
		result = currency + result
	}

	return result
}

// ParseJSON 解析JSON字符串，执行计算并返回结果
// 参数:
//   - jsonStr: 包含计算描述的JSON字符串
//   - 必须包含 "operation"(操作类型) 和 "values"(数值数组) 字段
//   - 可选的 "options" 字段用于提供额外参数
//
// 示例:
//
//	json1 := `{"operation":"sum","values":["10","20","30"]}`
//	result1 := Lin.Sum.ParseJSON(json1) // 返回 "60.00"
//
//	json2 := `{"operation":"percentage","values":["100","15"]}`
//	result2 := Lin.Sum.ParseJSON(json2) // 返回 "15.00"
func (s WsySum) ParseJSON(jsonStr string) string {
	var data map[string]interface{}

	// 解析JSON
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		Logs("ERROR", "SUM", "JSON解析失败: %v", err)
		return "0.00"
	}

	// 获取操作类型
	operation, ok := data["operation"].(string)
	if !ok {
		Logs("ERROR", "SUM", "未指定operation字段或类型错误")
		return "0.00"
	}

	// 获取值数组
	valuesRaw, ok := data["values"].([]interface{})
	if !ok {
		Logs("ERROR", "SUM", "未指定values字段或类型错误")
		return "0.00"
	}

	// 转换值数组为字符串数组
	values := make([]string, len(valuesRaw))
	for i, v := range valuesRaw {
		values[i] = fmt.Sprintf("%v", v)
	}

	// 获取可选参数
	var options []interface{}
	if optionsRaw, ok := data["options"].([]interface{}); ok {
		options = optionsRaw
	}

	// 执行对应的计算
	switch strings.ToLower(operation) {
	case "add":
		if len(values) >= 2 {
			return s.Add(values[0], values[1])
		}
	case "subtract":
		if len(values) >= 2 {
			return s.Subtract(values[0], values[1])
		}
	case "multiply":
		if len(values) >= 2 {
			return s.Multiply(values[0], values[1])
		}
	case "divide":
		if len(values) >= 2 {
			return s.Divide(values[0], values[1])
		}
	case "percentage":
		if len(values) >= 2 {
			return s.Percentage(values[0], values[1])
		}
	case "round":
		if len(values) >= 1 && len(options) >= 1 {
			places, ok := options[0].(float64)
			if !ok {
				places = 2 // 默认两位小数
			}
			return s.Round(values[0], int(places))
		}
	case "Ratio":
		if len(values) >= 2 {
			op := "%"
			precision := -1 // 默认保持完整精度

			if len(options) >= 1 {
				if opStr, ok := options[0].(string); ok {
					op = opStr
				}
			}

			if len(options) >= 2 {
				if prec, ok := options[1].(float64); ok {
					precision = int(prec)
				}
			}

			return s.Ratio(values[0], values[1], op, precision)
		}
	case "sum":
		return s.Sum(values...)
	case "average":
		return s.Average(values...)
	case "min":
		return s.Min(values...)
	case "max":
		return s.Max(values...)
	case "formatmoney":
		if len(values) >= 1 {
			return s.FormatMoney(values[0], options...)
		}
	default:
		Logs("ERROR", "SUM", "不支持的操作类型: %s", operation)
	}

	return "0.00"
}

// Sin 计算正弦值
// 参数:
//   - angle: 角度值（以弧度为单位）
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Sin("1.5708") // π/2 弧度，返回 "1.00"
//   - result := Lin.Sum.Sin("0.5236", 4) // π/6 弧度，返回 "0.5000"
func (s WsySum) Sin(angle string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	val, err := strconv.ParseFloat(angle, 64)
	if err != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v", err)
		return "0.00"
	}

	result := math.Sin(val)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Cos 计算余弦值
// 参数:
//   - angle: 角度值（以弧度为单位）
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Cos("0") // 返回 "1.00"
//   - result := Lin.Sum.Cos("3.1416", 4) // π 弧度，返回 "-1.0000"
func (s WsySum) Cos(angle string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	val, err := strconv.ParseFloat(angle, 64)
	if err != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v", err)
		return "0.00"
	}

	result := math.Cos(val)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Tan 计算正切值
// 参数:
//   - angle: 角度值（以弧度为单位）
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Tan("0.7854") // π/4 弧度，返回 "1.00"
//   - result := Lin.Sum.Tan("0", 4) // 返回 "0.0000"
func (s WsySum) Tan(angle string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	val, err := strconv.ParseFloat(angle, 64)
	if err != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v", err)
		return "0.00"
	}

	// 检查是否在临界点（如π/2, 3π/2等）
	if math.Cos(val) == 0 {
		Logs("ERROR", "SUM", "正切在该角度下无定义")
		return "0.00"
	}

	result := math.Tan(val)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Log 计算自然对数
// 参数:
//   - value: 要计算对数的数值
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Log("2.7183") // 约等于e，返回 "1.00"
//   - result := Lin.Sum.Log("10", 4) // 返回 "2.3026"
func (s WsySum) Log(value string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v", err)
		return "0.00"
	}

	if val <= 0 {
		Logs("ERROR", "SUM", "无法计算非正数的对数")
		return "0.00"
	}

	result := math.Log(val)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Log10 计算以10为底的对数
// 参数:
//   - value: 要计算对数的数值
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Log10("100") // 返回 "2.00"
//   - result := Lin.Sum.Log10("2", 4) // 返回 "0.3010"
func (s WsySum) Log10(value string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v", err)
		return "0.00"
	}

	if val <= 0 {
		Logs("ERROR", "SUM", "无法计算非正数的对数")
		return "0.00"
	}

	result := math.Log10(val)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Mod 计算两个数相除的余数
// 参数:
//   - dividend: 被除数
//   - divisor: 除数
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Mod("10", "3") // 返回 "1.00"
//   - result := Lin.Sum.Mod("10.5", "3.2", 1) // 返回 "0.9"
func (s WsySum) Mod(dividend, divisor string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	dividendVal, err1 := strconv.ParseFloat(dividend, 64)
	divisorVal, err2 := strconv.ParseFloat(divisor, 64)

	if err1 != nil || err2 != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", err1, err2)
		return "0.00"
	}

	if divisorVal == 0 {
		Logs("ERROR", "SUM", "除数不能为零")
		return "0.00"
	}

	result := math.Mod(dividendVal, divisorVal)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Pow 计算幂运算
// 参数:
//   - base: 底数
//   - exponent: 指数
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Pow("2", "3") // 返回 "8.00"
//   - result := Lin.Sum.Pow("2", "0.5", 4) // 返回 "1.4142"
func (s WsySum) Pow(base, exponent string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	baseVal, err1 := strconv.ParseFloat(base, 64)
	exponentVal, err2 := strconv.ParseFloat(exponent, 64)

	if err1 != nil || err2 != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v, %v", err1, err2)
		return "0.00"
	}

	// 处理特殊情况
	if baseVal < 0 && math.Floor(exponentVal) != exponentVal {
		Logs("ERROR", "SUM", "负数的非整数次幂无实数解")
		return "0.00"
	}

	result := math.Pow(baseVal, exponentVal)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// Sqrt 计算平方根
// 参数:
//   - value: 要计算平方根的数值
//   - precision: 可选参数，指定结果保留的小数位数，默认为2，-1表示保留完整精度
//
// 示例:
//   - result := Lin.Sum.Sqrt("9") // 返回 "3.00"
//   - result := Lin.Sum.Sqrt("2", 4) // 返回 "1.4142"
func (s WsySum) Sqrt(value string, args ...int) string {
	precision := 2 // 默认保留2位小数
	if len(args) > 0 {
		precision = args[0]
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		Logs("ERROR", "SUM", "数值解析失败: %v", err)
		return "0.00"
	}

	// 检查输入是否为负数
	if val < 0 {
		Logs("ERROR", "SUM", "无法计算负数的平方根")
		return "0.00"
	}

	result := math.Sqrt(val)

	// 使用指定精度格式化结果，不进行四舍五入
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}

// FormatFloat 将浮点数格式化为指定小数位的字符串
// 参数:
//   - value: 要格式化的浮点数值
//   - precision: 小数位数，默认为2
//
// 示例:
//   - Lin.Sum.FormatFloat(123.456, 2) // 返回 "123.46"
//   - Lin.Sum.FormatFloat(123.456, 0) // 返回 "123"
func (s WsySum) FormatFloat(value float64, precision int) string {
	// 定义输出格式
	format := fmt.Sprintf("%%.%df", precision)

	// 格式化浮点数
	return fmt.Sprintf(format, value)
}

// ConvertMoney 货币单位转换（分与元之间的转换）
// 参数:
//   - amount: 要转换的金额
//   - toYuan: 可选参数，true表示分转元，false表示元转分，默认为true
//   - precision: 可选参数，指定结果保留的小数位数，默认为2
//
// 示例:
//   - result := Lin.Sum.ConvertMoney("10000") // 分转元，返回 "100.00"
//   - result := Lin.Sum.ConvertMoney("100.50", false) // 元转分，返回 "10050"
//   - result := Lin.Sum.ConvertMoney("10000", true, 3) // 分转元，返回 "100.000"
func (s WsySum) ConvertMoney(amount string, args ...interface{}) string {
	var result float64
	toYuan := true // 默认分转元
	precision := 2 // 默认保留2位小数
	// 解析金额
	val, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		Logs("ERROR", "SUM", "金额解析失败: %v", err)
		return amount
	}
	if len(args) > 0 {
		if toYuanArg, ok := args[0].(bool); ok {
			toYuan = toYuanArg
		}
	}
	if len(args) > 1 {
		if prec, ok := args[1].(int); ok {
			precision = prec
		}
	}
	if toYuan {
		result = val / 100.0 // 分转元
	} else {
		result = val * 100.0 // 元转分
	}
	// 格式化结果
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, result)
}
