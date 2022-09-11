/**
 * Package binding
 * @Description: json绑定器，实现参数校验
 */
package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

type jsonBinding struct {
	DisallowUnknownFields bool
	IsValidate            bool
}

func (jsonBinding) Name() string {
	return "json"
}

func (j jsonBinding) Bind(r *http.Request, data any) error {
	body := r.Body
	if body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	if j.DisallowUnknownFields {
		// 参数中有的属性，但是对应的结构体没有，报错，也就是检查结构体是否有效
		decoder.DisallowUnknownFields()
	}

	if j.IsValidate {
		// 传参中没有，而接收的结构体中有的参数时
		err := validateParam(data, decoder)
		if err != nil {
			return err
		}
	} else {
		err := decoder.Decode(data)
		if err != nil {
			return err
		}
	}
	return validate(data)
}
func validate(data any) error {
	return Validator.ValidateStruct(data)
}

/**
 * validateParam
 * @Author：Jack-Z
 * @Description: json-利用反射-参数校验
 * @param data
 * @param decoder
 * @return error
 */
func validateParam(data any, decoder *json.Decoder) error {
	// 解析map，然后根据map中的key进行比对
	if data == nil {
		return nil
	}
	valueOf := reflect.ValueOf(data)
	if valueOf.Kind() != reflect.Pointer {
		// 不是指针类型
		return errors.New("not pointer type")
	}

	t := valueOf.Elem().Interface()
	of := reflect.ValueOf(t)
	switch of.Kind() {
	case reflect.Struct: // 结构体检验
		return checkParamStruct(of, data, decoder)

	case reflect.Slice, reflect.Array: // 切片（数组）校验
		elem := of.Type().Elem()
		if elem.Kind() == reflect.Struct {
			return checkParamSlice(elem, data, decoder)
		}

	default:
		_ = decoder.Decode(data)
	}

	return nil
}

/**
 * checkParamSlice
 * @Author：Jack-Z
 * @Description: 参数切片（多层嵌套的参数，如对象数组结构的）
 * @param of
 * @param data
 * @param decoder
 * @return error
 */
func checkParamSlice(of reflect.Type, data any, decoder *json.Decoder) error {
	mapData := make([]map[string]interface{}, 0)
	_ = decoder.Decode(&mapData)
	for i := 0; i < of.NumField(); i++ {
		field := of.Field(i)
		required := field.Tag.Get("restrict")
		tag := field.Tag.Get("json")

		for _, v := range mapData {
			value := v[tag]
			if value == nil && required == "required" {
				return errors.New(fmt.Sprintf("filed [%s] is required", tag))
			}
		}
	}
	marshal, _ := json.Marshal(mapData)
	_ = json.Unmarshal(marshal, data)
	return nil
}

/**
 * checkParamStruct
 * @Author：Jack-Z
 * @Description: json数据检验（一维对象）
 * @param of
 * @param data
 * @param decoder
 * @return error
 */
func checkParamStruct(of reflect.Value, data any, decoder *json.Decoder) error {
	mapData := make(map[string]interface{})
	_ = decoder.Decode(&mapData)
	for i := 0; i < of.NumField(); i++ {
		field := of.Type().Field(i)
		required := field.Tag.Get("restrict")
		tag := field.Tag.Get("json")
		value := mapData[tag]
		if value == nil && required == "required" {
			return errors.New(fmt.Sprintf("filed [%s] is required", tag))
		}
	}
	marshal, _ := json.Marshal(mapData)
	_ = json.Unmarshal(marshal, data)
	return nil
}
