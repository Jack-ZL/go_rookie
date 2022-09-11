package binding

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
	"sync"
)

type StructValidator interface {
	ValidateStruct(any) error // 结构体验证，返回对应的错误信息
	Engine() any              // 返回对应的验证器
}

var Validator StructValidator = &defaultValidator{}

type defaultValidator struct {
	one      sync.Once
	validate *validator.Validate
}

/**
 * 重写error，满足批量错误输出
 */
type SliceValidationError []error

func (err SliceValidationError) Error() string {
	n := len(err)
	switch n {
	case 0:
		return ""
	default:
		var b strings.Builder
		if err[0] != nil {
			fmt.Fprintf(&b, "[%d]: %s", 0, err[0].Error())
		}
		if n > 1 {
			for i := 1; i < n; i++ {
				if err[i] != nil {
					b.WriteString("\n")
					fmt.Fprintf(&b, "[%d]: %s", i, err[i].Error())
				}
			}
		}
		return b.String()
	}
}

/**
 * lazyInit
 * @Author：Jack-Z
 * @Description: 懒加载New验证器
 * 每次都需要使用`validator.New()`，会极大的浪费性能，可以使用单例来做优化
 * @receiver d
 */
func (d *defaultValidator) lazyInit() {
	d.one.Do(func() {
		d.validate = validator.New()
	})
}

func (d *defaultValidator) Engine() any {
	d.lazyInit()
	return d.validate
}

func (d *defaultValidator) validateStruct(data any) error {
	d.lazyInit()
	return d.validate.Struct(data)
}

/**
 * ValidateStruct
 * @Author：Jack-Z
 * @Description: 参数校验
 * @receiver d
 * @param data
 * @return error
 */
func (d *defaultValidator) ValidateStruct(data any) error {
	of := reflect.ValueOf(data)
	switch of.Kind() {
	case reflect.Pointer:
		return d.ValidateStruct(of.Elem().Interface())
	case reflect.Struct:
		return d.validateStruct(data)
	case reflect.Slice, reflect.Array:
		count := of.Len()
		sliceValidationError := make(SliceValidationError, 0)
		for i := 0; i < count; i++ {
			if err := d.validateStruct(of.Index(i).Interface()); err != nil {
				sliceValidationError = append(sliceValidationError, err)
			}
		}
		// 错误切片为空，说明没任何错误，直接返回nil
		if len(sliceValidationError) == 0 {
			return nil
		}
		return sliceValidationError
	}
	return nil
}
