package render

import (
	"fmt"
	"github.com/Jack-ZL/go_rookie/internal/bytesconv"
	"net/http"
)

type String struct {
	Format string
	Data   []any
}

/**
 * Render
 * @Author：Jack-Z
 * @Description: 字符串处理
 * @receiver s
 * @param w
 * @return error
 */
func (s *String) Render(w http.ResponseWriter) error {
	s.WriteContentType(w)
	if len(s.Data) > 0 {
		_, err := fmt.Fprintf(w, s.Format, s.Data...)
		return err
	}
	_, err := w.Write(bytesconv.StringToBytes(s.Format))
	return err
}

/**
 * WriteContentType
 * @Author：Jack-Z
 * @Description: 设置content-type
 * @receiver s
 * @param w
 */
func (s *String) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "text/plain; charset=utf-8")
}
