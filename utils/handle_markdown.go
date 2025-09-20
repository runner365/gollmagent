package utils

import (
	"regexp"
)

// MarkdownToText 去除输入字符串中的常见 Markdown 格式
func MarkdownToText(input string) string {
	// 定义一系列正则表达式来匹配常见的 Markdown 语法
	replacements := []struct {
		pattern *regexp.Regexp
		repl    string
	}{
		// 移除标题
		{regexp.MustCompile(`^#\s+`), ""},   // # 标题
		{regexp.MustCompile(`^##\s+`), ""},  // ## 标题
		{regexp.MustCompile(`^###\s+`), ""}, // ### 标题
		// 可以继续添加更多标题级别

		// 移除粗体 ​**text**​ 或 __text__
		{regexp.MustCompile(`\*\*(.*?)\*\*`), "${1}"},
		{regexp.MustCompile(`__(.*?)__`), "${1}"},

		// 移除斜体 *text* 或 _text_
		{regexp.MustCompile(`\*(.*?)\*`), "${1}"},
		{regexp.MustCompile(`_(.*?)_`), "${1}"},

		// 移除删除线 ~~text~~
		{regexp.MustCompile(`~~(.*?)~~`), "${1}"},

		// 移除代码块 `code`
		{regexp.MustCompile("`([^`]+)`"), "${1}"},

		// 移除代码块 ```code block```
		// {regexp.MustCompile("```.*?\n(.*?)\n```", regexp.DotAll), "${1}"},

		// 移除链接 [text](url)
		{regexp.MustCompile(`$$(.*?)$$$.*?$`), "${1}"},

		// 移除图片 ![alt](url)
		{regexp.MustCompile(`!$$(.*?)$$$.*?$`), "${1}"},
	}

	// 对输入字符串依次应用所有正则替换
	result := input
	for _, r := range replacements {
		result = r.pattern.ReplaceAllString(result, r.repl)
	}

	return result
}
