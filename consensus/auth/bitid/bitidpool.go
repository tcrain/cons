package bitid

import (
	"sync"
)

type BitIDPoolInterface interface {
	Done(NewBitIDInterface)
	Get() NewBitIDInterface
}

func NewBitIDPool(idFunc NewBitIDFunc, allowConcurrency bool) BitIDPoolInterface {
	if allowConcurrency {
		ret := &ConcurrentBitIDPool{}
		ret.pool.New = func() interface{} { return idFunc() }
		return ret
	} else {
		ret := &BitIDPool{newFunc: idFunc}
		return ret
	}
}

type BitIDPool struct {
	objs      []NewBitIDInterface
	newFunc   NewBitIDFunc
	allocated int
}

func (bp *BitIDPool) Done(bid NewBitIDInterface) {
	bid.Done()
	bp.objs = append(bp.objs, bid)
}
func (bp *BitIDPool) Get() NewBitIDInterface {
	if len(bp.objs) == 0 {
		bp.objs = append(bp.objs, bp.newFunc())
		bp.allocated++
	}
	ret := bp.objs[0]
	bp.objs[0] = bp.objs[len(bp.objs)-1]
	bp.objs = bp.objs[:len(bp.objs)-1]
	return ret
}

type ConcurrentBitIDPool struct {
	pool sync.Pool
}

func (bp *ConcurrentBitIDPool) Done(bid NewBitIDInterface) {
	bid.Done()
	bp.pool.Put(bid)
}

func (bp *ConcurrentBitIDPool) Get() NewBitIDInterface {
	return bp.pool.Get().(NewBitIDInterface)
}
