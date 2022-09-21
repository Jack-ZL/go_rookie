package binding

import (
	"encoding/xml"
	"net/http"
)

type xmlBinding struct {
}

func (xmlBinding) Name() string {
	return "xml"
}

func (x xmlBinding) Bind(r *http.Request, data any) error {
	if r.Body == nil {
		return nil
	}

	decoder := xml.NewDecoder(r.Body)
	err := decoder.Decode(data)
	if err != nil {
		return err
	}
	return validate(data)
}
