package Wsy

import (
	"html"
)

type WsyHtml struct{}

// html -> Escape
func (s WsyHtml) Escape(value string) string {
	return html.EscapeString(value)
}
// Escape -> html 将HTML实体还原为原始字符串
func (s WsyHtml) UnEscape(value string) string {
	return html.UnescapeString(value)
}