package go_rookie

import (
	"strings"
	"unicode"
	"unsafe"
)

/**
 * SubStringLast
 * @Author：Jack-Z
 * @Description: 字符串截取
 * @param str  给定的字符串
 * @param substr   子串
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
 * @Description: 该方法用于判断传入的字符串s是否全部由ASCII字符组成。如果字符串中有任意一个字符的ASCII码值大于unicode.MaxASCII（即127），则返回false，否则返回true
 * @param s 字符串
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

/**
 * StringToBytes
 * @Author：Jack-Z
 * @Description:字符串转byte切片
 * @param s 字符串
 * @return []byte
 */
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}
