package render

import (
	"encoding/xml"
	"net/http"
)

type XML struct {
	Data any
}

/**
 * Render
 * @Author：Jack-Z
 * @Description: 字符串处理
 * @receiver s
 * @param w
 * @return error
 */
func (x *XML) Render(w http.ResponseWriter, code int) error {
	x.WriteContentType(w)
	w.WriteHeader(code)
	err := xml.NewEncoder(w).Encode(x.Data)
	return err
}

/**
 * WriteContentType
 * @Author：Jack-Z
 * @Description: 设置content-type
 * @receiver s
 * @param w
 */
func (x *XML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "application/xml; charset=utf-8")
}
