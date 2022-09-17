package grpool

import "time"

type Worker struct {
	pool     *Pool
	task     chan func() // 任务管道队列
	lastTime time.Time   // 执行任务的最后时间

}
