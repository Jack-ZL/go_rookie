package render

import (
	"encoding/json"
	"net/http"
)

type JSON struct {
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
func (j *JSON) Render(w http.ResponseWriter, code int) error {
	j.WriteContentType(w)
	w.WriteHeader(code)
	jsonData, err := json.Marshal(j.Data)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonData)
	return err
}

/**
 * WriteContentType
 * @Author：Jack-Z
 * @Description: 设置content-type
 * @receiver s
 * @param w
 */
func (j *JSON) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, "application/json; charset=utf-8")
}
