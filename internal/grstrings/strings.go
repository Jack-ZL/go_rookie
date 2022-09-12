package grstrings

import (
	"fmt"
	"reflect"
	"strings"
)

/**
 * JoinStrings
 * @Author：Jack-Z
 * @Description: 字符串拼接
 * @param data
 * @return string
 */
func JoinStrings(data ...any) string {
	var sb strings.Builder
	for _, v := range data {
		sb.WriteString(check(v))
	}
	return sb.String()
}

/**
 * check
 * @Author：Jack-Z
 * @Description: 变量类型转换为string
 * @param v
 * @return string
 */
func check(v any) string {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.String:
		return v.(string)
	default:
		return fmt.Sprintf("%v", v)
	}
}
