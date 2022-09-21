/**
 * Package bytesconv
 * @Description: internal 目录下的包，不允许被其他项目中进行导入，这是在 Go 1.4 当中引入的 feature，会在编译时执行
 */
package bytesconv

import "unsafe"

/**
 * StringToBytes
 * @Author：Jack-Z
 * @Description:字符串转byte切片
 * @param s
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
