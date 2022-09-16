package grerror

type GrError struct {
	err     error
	ErrFunc ErrorFunc
}

func Default() *GrError {
	return &GrError{}
}

func (e *GrError) Error() string {
	return e.err.Error()
}

func (e *GrError) Put(err error) {
	e.Check(err)

}

func (e *GrError) Check(err error) {
	if err != nil {
		e.err = err
		panic(e) // 抛出错误
	}
}

type ErrorFunc func(grError *GrError)

func (e *GrError) Result(errFunc ErrorFunc) {
	e.ErrFunc = errFunc
}

// 暴漏方法，让用户自定义
func (e *GrError) ExecuteResult() {
	e.ErrFunc(e)
}
