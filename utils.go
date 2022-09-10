package go_rookie

import (
	"strings"
	"unicode"
)

/**
 * SubStringLast
 * @Author：Jack-Z
 * @Description: 字符串截取
 * @param str
 * @param substr
 * @return string
 */
func SubStringLast(str string, substr string) string {
	index := strings.Index(str, substr)
	if index < 0 {
		return ""
	}
	return str[index+len(substr):]
}

/**
 * isASCII
 * @Author：Jack-Z
 * @Description: 判断是否为ASC字符
 * @param s
 * @return bool
 */
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
