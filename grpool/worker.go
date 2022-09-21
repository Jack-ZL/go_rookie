package grpool

import (
	grLog "github.com/Jack-ZL/go_rookie/log"
	"time"
)

type Worker struct {
	pool     *Pool
	task     chan func() // 任务管道队列
	lastTime time.Time   // 执行任务的最后时间

}

func (w *Worker) run() {
	w.pool.incRunning()
	go w.running()
}

/**
 * running
 * @Author：Jack-Z
 * @Description: 开始执行任务（for循环方式）
 * @receiver w
 */
func (w *Worker) running() {
	defer func() {
		w.pool.decRunning()
		w.pool.workerCache.Put(w)
		if err := recover(); err != nil {
			// 捕获任务发生时的panic
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				grLog.Default().Error(err)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			w.pool.workerCache.Put(w)
			return
		}

		f()
		// 任务运行完成，worker空闲了把它放还到pool池中
		w.pool.PutWorker(w)
	}
}
