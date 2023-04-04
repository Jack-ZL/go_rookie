package breaker

import (
	"errors"
	"sync"
	"time"
)

// 熔断服务

type State int //状态标识

const (
	StateClosed   State = iota //关闭
	StateHalfOpen              //半开启
	StateOpen                  //开启
)

// 计数
type Counts struct {
	Requests             uint32 //请求数量
	TotalSuccesses       uint32 //总成功数
	TotalFailures        uint32 //总失败数
	ConsecutiveSuccesses uint32 //连续成功数量
	ConsecutiveFailures  uint32 //连续失败数量
}

func (c *Counts) OnRequest() {
	c.Requests++
}

func (c *Counts) OnSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) OnFail() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) Clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveFailures = 0
	c.ConsecutiveSuccesses = 0
}

/**
 * Settings
 *  @Description: 断路器各项设置
 */
type Settings struct {
	Name          string                                  //名字
	MaxRequests   uint32                                  //最大请求数
	Interval      time.Duration                           //间隔时间
	Timeout       time.Duration                           //超时时间
	ReadyToTrip   func(counts Counts) bool                //执行熔断
	OnStateChange func(name string, from State, to State) //状态变更
	IsSuccessful  func(err error) bool                    //是否成功
	Fallback      func(err error) (any, error)            //降级处理方法
}

// CircuitBreaker 断路器
type CircuitBreaker struct {
	name          string                                  //名字
	maxRequests   uint32                                  //最大请求数：当连续请求成功数大于此时 断路器关闭
	interval      time.Duration                           //间隔时间
	timeout       time.Duration                           //超时时间
	readyToTrip   func(counts Counts) bool                //是否执行熔断
	isSuccessful  func(err error) bool                    //是否成功
	onStateChange func(name string, from State, to State) //状态变更

	mutex      sync.Mutex
	state      State                        //状态
	generation uint64                       //新的一代：状态变更，new一个
	counts     Counts                       //数量
	expiry     time.Time                    //到期时间 检查是否从开到半开
	fallback   func(err error) (any, error) //降级处理方法
}

func (cb *CircuitBreaker) NewGeneration() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.generation++
	cb.counts.Clear()
	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = time.Now().Add(cb.interval)
		}

	case StateOpen:
		cb.expiry = time.Now().Add(cb.timeout)
	case StateHalfOpen:
		cb.expiry = zero
	}
}

func NewCircuitBreaker(st Settings) *CircuitBreaker {
	cb := new(CircuitBreaker)
	cb.name = st.Name
	cb.onStateChange = st.OnStateChange
	cb.fallback = st.Fallback
	if st.MaxRequests == 0 {
		cb.maxRequests = 1
	} else {
		cb.maxRequests = st.MaxRequests
	}

	if st.Interval == 0 {
		cb.interval = time.Duration(0) * time.Second
	} else {
		cb.interval = st.Interval
	}

	if st.Timeout == 0 {
		// 断路器 由开->半开
		cb.timeout = time.Duration(20) * time.Second
	} else {
		cb.timeout = st.Timeout
	}

	if st.ReadyToTrip == nil {
		cb.readyToTrip = func(counts Counts) bool {
			return counts.ConsecutiveFailures > 5
		}
	} else {
		cb.readyToTrip = st.ReadyToTrip
	}

	if st.IsSuccessful == nil {
		cb.isSuccessful = func(err error) bool {
			return err == nil
		}
	} else {
		cb.isSuccessful = st.IsSuccessful
	}
	cb.NewGeneration()

	return cb
}

func (cb *CircuitBreaker) Execute(req func() (any, error)) (any, error) {
	// 判读是否执行断路器
	err, generation := cb.beforeRequest()
	if err != nil {
		// 发生错误时，执行降级方法
		if cb.fallback != nil {
			cb.fallback(err)
		}
		return nil, err
	}
	result, err := req() //发起一个请求
	cb.counts.OnRequest()

	// 请求之后判断：当前状态是否需要更新
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

func (cb *CircuitBreaker) beforeRequest() (error, uint64) {
	// 判断当前状态：如果断路器是打开状态，直接返回err
	now := time.Now()
	state, generation := cb.currentState(now)
	if state == StateOpen {
		return errors.New("断路器是打开状态"), generation
	}

	if state == StateHalfOpen {
		if cb.counts.Requests > cb.maxRequests {
			return errors.New("请求数量过多"), generation
		}
	}

	return nil, generation
}

func (cb *CircuitBreaker) afterRequest(beforeGeneration uint64, isSuccessful bool) {
	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != beforeGeneration {
		return
	}
	if isSuccessful {
		cb.OnSuccess(state)
	} else {
		cb.OnFail(state)
	}
}

/**
 * currentState
 * @Author：Jack-Z
 * @Description: 当前状态
 * @receiver cb
 * @param now
 * @return State
 */
func (cb *CircuitBreaker) currentState(now time.Time) (State, uint64) {
	switch cb.state {
	case StateClosed:
		if cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.NewGeneration()
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.SetState(StateHalfOpen)
		}
	}
	return cb.state, cb.generation
}

/**
 * SetState
 * @Author：Jack-Z
 * @Description: 设置状态
 * @receiver cb
 * @param target
 */
func (cb *CircuitBreaker) SetState(target State) {
	if cb.state == target {
		return
	}
	before := cb.state
	cb.state = target
	// 状态变更之后，重新计数
	cb.NewGeneration()

	if cb.onStateChange == nil {
		cb.onStateChange(cb.name, before, target)
	}
}

func (cb *CircuitBreaker) OnSuccess(state State) {
	switch state {
	case StateClosed:
		cb.counts.OnSuccess()

	case StateHalfOpen:
		cb.counts.OnSuccess()
		if cb.counts.ConsecutiveSuccesses > cb.maxRequests {
			cb.SetState(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) OnFail(state State) {
	switch state {
	case StateClosed:
		cb.counts.OnFail()
		if cb.readyToTrip(cb.counts) {
			cb.SetState(StateOpen)
		}
	case StateHalfOpen:
		cb.counts.OnFail()
		if cb.readyToTrip(cb.counts) {
			cb.SetState(StateOpen)
		}
	}
}
