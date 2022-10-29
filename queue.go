package queue

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

const (
	DEFAULT_QUEUE_CAP = 1024
)

var (
	POOL_EMIPTY = errors.New("pool is empty")
	POOL_FULL   = errors.New("pool is full")
)

type Queue struct {
	q       chan *[]byte
	bufpool sync.Pool
	cap     int
	isInit  int32
}

func (q *Queue) Read(b []byte) (n int, err error) {
	var _b *[]byte
	if q.isInit == 0 {
		if atomic.CompareAndSwapInt32(&q.isInit, 0, 1) {
			_b = <-q.q
		} else {
			select {
			case _b = <-q.q:
			default:
				err = POOL_EMIPTY
				return
			}
		}
	} else {
		select {
		case _b = <-q.q:
		default:
			err = POOL_EMIPTY
			return
		}
	}
	buf := *_b
	defer func() {
		buf = buf[0:cap(buf)]
		q.bufpool.Put(_b)
	}()
	n = copy(b, buf)
	return
}

func (q *Queue) Write(b []byte) (n int, err error) {
	buf := *q.bufpool.Get().(*[]byte)
	n = copy(buf, b)
	buf = buf[0:n]
	select {
	case q.q <- &buf:
	default:
		err = POOL_FULL
	}
	return
}

func New(cap ...int) io.ReadWriter {
	q := &Queue{
		bufpool: sync.Pool{
			New: func() any {
				b := make([]byte, 4096)
				return &b
			},
		},
	}
	if len(cap) > 0 {
		q.cap = cap[0]
	} else {
		q.cap = DEFAULT_QUEUE_CAP
	}
	q.q = make(chan *[]byte, q.cap)
	return q
}

func NewQueueSize(size int, cap ...int) io.ReadWriter {
	q := &Queue{
		bufpool: sync.Pool{
			New: func() any {
				b := make([]byte, size)
				return &b
			},
		},
	}
	if len(cap) > 0 {
		q.cap = cap[0]
	} else {
		q.cap = DEFAULT_QUEUE_CAP
	}
	q.q = make(chan *[]byte, q.cap)
	return q
}
