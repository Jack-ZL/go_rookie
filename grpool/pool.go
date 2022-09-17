package grpool

import (
	"errors"
	"sync"
	"time"
)

type sig struct{}

const DefaultExpire = 3 // 默认的过期时间：3秒

var (
	ErrorInvalidCap    = errors.New("pool cap can't <= 0")
	ErrorInvalidExpire = errors.New("pool expire can't <= 0")
)

type Pool struct {
	cap     int32         // pool的最大容量
	running int32         // 正在运行的worker的数量
	workers []*Worker     // 多个空闲的worker
	expire  time.Duration // 过期时间：空闲的worker超过这个时间就回收
	release chan sig      // 释放资源的关闭信号，pool就不能使用
	lock    sync.Mutex    // 加锁，保证pool里面的资源的安全（即保护worker的资源）
	once    sync.Once     // 释放操作只能调用一次，不能多次调用
}

func NewPool(cap int) (*Pool, error) {
	return NewTimePool(cap, DefaultExpire)
}

func NewTimePool(c int, expire int) (*Pool, error) {
	if c <= 0 {
		return nil, ErrorInvalidCap
	}

	if expire <= 0 {
		return nil, ErrorInvalidExpire
	}

	p := &Pool{
		cap:     int32(c),
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	return p, nil
}

/**
 * Submit
 * @Author：Jack-Z
 * @Description: 提交任务：获取pool池里面的一个worker，然后执行任务即可！
 * @receiver p
 * @param task
 * @return error
 */
func (p *Pool) Submit(task func()) error {
	w := p.GetWorker()
	w.task <- task
	return nil
}

/**
 * GetWorker
 * @Author：Jack-Z
 * @Description: 获取一个worker
 * @receiver p
 * @return *Worker
 */
func (p *Pool) GetWorker() *Worker {
	return nil
}
