package grpool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type sig struct{}

const DefaultExpire = 3 // 默认的过期时间：3秒

var (
	ErrorInvalidCap    = errors.New("pool cap can't <= 0")
	ErrorInvalidExpire = errors.New("pool expire can't <= 0")
	ErrorHasClosed     = errors.New("pool has been released")
)

/**
 * Pool
 *  @Description: 定义一个pool协程池
 */
type Pool struct {
	cap          int32         // pool协程池的最大容量
	running      int32         // 正在运行的worker的数量
	workers      []*Worker     // 多个空闲的worker
	expire       time.Duration // 过期时间：空闲的worker超过这个时间就回收
	release      chan sig      // 释放资源的关闭信号，pool就不能使用
	lock         sync.Mutex    // 加锁，保证pool里面的资源的安全（即保护worker的资源）
	once         sync.Once     // 释放操作只能调用一次，不能多次调用
	workerCache  sync.Pool     // worker缓存
	cond         *sync.Cond    // 当有空闲的worker时通知阻塞进程
	PanicHandler func()        // 异常错误处理
}

/**
 * NewPool
 * @Author：Jack-Z
 * @Description: new的协程池
 * @param cap
 * @return *Pool
 * @return error
 */
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
	p.workerCache.New = func() any {
		return &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
	}
	p.cond = sync.NewCond(&p.lock)
	go p.expireWorker()
	return p, nil
}

/**
 * expireWorker
 * @Author：Jack-Z
 * @Description: 定时清理过期的空闲worker
 * @receiver p
 */
func (p *Pool) expireWorker() {
	// 使用定时器
	ticker := time.NewTicker(p.expire)
	for range ticker.C {
		if p.IsClosed() { // 释放资源了就不用执行清除了
			break
		}
		// 循环空闲的workers，如果当前时间和worker的最后运行任务的时间 间隔大于expire，则进行清理
		p.lock.Lock()
		idleWorkers := p.workers
		n := len(idleWorkers) - 1
		if n >= 0 {
			var cleanN = -1
			for i, w := range idleWorkers {
				if time.Now().Sub(w.lastTime) <= p.expire {
					break
				}
				cleanN = i
				w.task <- nil
				idleWorkers[i] = nil
			}
			if cleanN != -1 {
				if cleanN >= len(idleWorkers)-1 {
					p.workers = idleWorkers[:0]
				} else {
					p.workers = idleWorkers[cleanN+1:]
				}
			}
		}
		p.lock.Unlock()
	}
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
	if len(p.release) > 0 {
		return ErrorHasClosed
	}
	// 获取pool池里面的一个worker，然后执行任务即可
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
	// 1、获取Pool里面的worker，如果有空闲的worker，直接获取
	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n >= 0 {
		w := idleWorkers[n]         // 去除末尾的那个worker
		idleWorkers[n] = nil        // 被取走的位置置为nil
		p.workers = idleWorkers[:n] // pool池中数量-1
		p.lock.Unlock()
		return w
	}
	// 2、如果没有空闲的worker，则需要新建一个worker
	if p.running < p.cap {
		p.lock.Unlock()
		// 新建一个worker
		c := p.workerCache.Get()
		var w *Worker
		if c == nil {
			w = &Worker{
				pool: p,
				task: make(chan func(), 1),
			}
		} else {
			w = c.(*Worker)
		}

		w.run()
		return w
	}
	p.lock.Unlock()

	// 3、如果 运行中的worker >= pool的容量，则阻塞等待有worker释放
	return p.waitIdleWorker()
}

/**
 * waitIdleWorker
 * @Author：Jack-Z
 * @Description: 阻塞等待有worker释放
 * @receiver p
 * @return *Worker
 */
func (p *Pool) waitIdleWorker() *Worker {
	p.lock.Lock()
	p.cond.Wait()

	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n < 0 {
		p.lock.Unlock()

		if p.running < p.cap {
			// 新建一个worker
			c := p.workerCache.Get()
			var w *Worker
			if c == nil {
				w = &Worker{
					pool: p,
					task: make(chan func(), 1),
				}
			} else {
				w = c.(*Worker)
			}

			w.run()
			return w
		}

		return p.waitIdleWorker()
	}
	w := idleWorkers[n]         // 取出末尾的那个worker
	idleWorkers[n] = nil        // 被取走的位置置为nil
	p.workers = idleWorkers[:n] // pool池中数量-1
	p.lock.Unlock()
	return w
}

/**
 * incRunning
 * @Author：Jack-Z
 * @Description:（原子操作）增加一个运行中的worker
 * @receiver p
 */
func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) PutWorker(w *Worker) {
	w.lastTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.cond.Signal()
	p.lock.Unlock()
}

/**
 * decRunning
 * @Author：Jack-Z
 * @Description: （原子操作）减去一个运行中的worker
 * @receiver p
 */
func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}

/**
 * Release
 * @Author：Jack-Z
 * @Description: 释放资源
 * @receiver p
 */
func (p *Pool) Release() {
	p.once.Do(func() {
		// 只执行一次
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
		p.release <- sig{}
	})
}

/**
 * IsClosed
 * @Author：Jack-Z
 * @Description: 判断是否释放
 * @receiver p
 * @return bool
 */
func (p *Pool) IsClosed() bool {
	return len(p.release) > 0
}

/**
 * Restart
 * @Author：Jack-Z
 * @Description:重启资源使用
 * @receiver p
 * @return bool
 */
func (p *Pool) Restart() bool {
	if len(p.release) <= 0 {
		return true
	}
	_ = <-p.release
	go p.expireWorker()
	return true
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

func (p *Pool) Free() int {
	return int(p.cap - p.running)
}
