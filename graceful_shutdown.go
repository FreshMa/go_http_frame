package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
)

var ErrHookTimeout = errors.New("hook timeout")

type GracefulShutdown struct {
	reqCnt  int64
	closing uint32

	zeroReqCh chan struct{}
}

func NewGracefulShutdown() *GracefulShutdown {
	return &GracefulShutdown{
		reqCnt:    0,
		closing:   0,
		zeroReqCh: make(chan struct{}, 2),
	}
}

func (g *GracefulShutdown) RejectRequestMiddleware() HandleFunc {
	return func(c *Context) {
		cl := atomic.LoadUint32(&g.closing)
		if cl == 1 {
			log.Printf("server shutdown ing, request rejected\n")
			c.W.WriteHeader(http.StatusServiceUnavailable)
			c.W.Write([]byte("server shutdown ing..."))
			c.Abort()
			return
		}

		// 这里处理的是请求未处理完成之前服务关闭的情况
		atomic.AddInt64(&g.reqCnt, 1)
		c.Next()
		n := atomic.AddInt64(&g.reqCnt, -1)

		// 这里必须重新取一次
		cl = atomic.LoadUint32(&g.closing)
		if cl == 1 && n == 0 {
			g.zeroReqCh <- struct{}{}
		}

	}
}

// 开始拒绝请求，并且等到当前请求全部完成
// 这里原来有个bug，如果当前没有请求的时候，就不会走到middleware那里，触发不了zeroReqCh的条件
// 所以这里加了个补偿逻辑，如果在执行hook的时候发现请求数已经为0了，塞一个进去
func (g *GracefulShutdown) RejectRequestAndWaiting(ctx context.Context) error {
	atomic.StoreUint32(&g.closing, 1)

	// 处理服务关闭时没有请求的情况
	n := atomic.LoadInt64(&g.reqCnt)
	if n == 0 {
		g.zeroReqCh <- struct{}{}
	}

	select {
	case <-g.zeroReqCh:
		return nil
	case <-ctx.Done():
		return ErrHookTimeout
	}
}

func (g *GracefulShutdown) WaitServerShutdown(servers ...Server) Hook {
	return func(ctx context.Context) error {
		var wg sync.WaitGroup
		wg.Add(len(servers))
		for _, svr := range servers {
			go func(svr Server) {
				svr.Shutdown(ctx)
				wg.Done()
			}(svr)
		}

		// 需要一个通知svr shutdown完成的chan
		doneCh := make(chan struct{})
		go func() {
			wg.Wait()
			doneCh <- struct{}{}
		}()

		select {
		case <-doneCh:
			log.Printf("all svr shutdown gracefully\n")
			return nil
		case <-ctx.Done():
			log.Printf("svr shutdown timeout\n")
			return ErrHookTimeout
		}
	}
}
