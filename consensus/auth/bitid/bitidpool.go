package bitid

import (
	"sync"
)

type BitIDPoolInterface interface {
	Done(NewBitIDInterface)
	Get() NewBitIDInterface
}

func NewBitIDPool(idFunc NewBitIDFunc, allowConcurrency bool) BitIDPoolInterface {
	if false {
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
	mutex     sync.Mutex
}

func (bp *BitIDPool) Done(bid NewBitIDInterface) {
	bid.Done()
	bp.mutex.Lock()
	bp.objs = append(bp.objs, bid)
	bp.mutex.Unlock()
}
func (bp *BitIDPool) Get() NewBitIDInterface {
	bp.mutex.Lock()
	if len(bp.objs) == 0 {
		bp.objs = append(bp.objs, bp.newFunc())
		bp.allocated++
	}
	ret := bp.objs[0]
	bp.objs[0] = bp.objs[len(bp.objs)-1]
	bp.objs = bp.objs[:len(bp.objs)-1]
	bp.mutex.Unlock()
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
