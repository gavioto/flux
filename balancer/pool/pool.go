package pool

import (
	"container/heap"
	"math/rand"
	"sync"
	"time"

	"github.com/weaveworks/flux/balancer/model"
	"github.com/weaveworks/flux/common/netutil"
)

const retry_interval_base = 1 * time.Second

type PooledInstance struct {
	Instance  model.Instance
	index     int
	failures  uint
	retryTime time.Time
}

type InstancePool struct {
	lock  sync.Mutex
	rng   *rand.Rand
	now   func() time.Time
	timer interface {
		Reset(d time.Duration) bool
		Stop() bool
	}
	stopped chan struct{}

	// Instances that are ready for connections
	ready []*PooledInstance

	retryQueue
}

type retryQueue struct {
	retry []*PooledInstance
}

func NewInstancePool() *InstancePool {
	return &InstancePool{
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		now:     time.Now,
		stopped: make(chan struct{}),
	}
}

func (p *InstancePool) Stop() {
	close(p.stopped)

	if p.timer != nil {
		p.timer.Stop()
	}
}

// Pick an instance from the pool; ideally, from amongst the active
// instances, but failing that, from those waiting to be retried.
func (p *InstancePool) PickInstance() *PooledInstance {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Normal case: Pick a random ready instance.
	if len(p.ready) != 0 {
		inst := p.ready[p.rng.Intn(len(p.ready))]
		if inst.failures != 0 {
			// Retrying a suspect instance, so presume its
			// failure in order to prevent other threads
			// from using it.  A subsequent Failed() call
			// will be idempotent, a Succeeded() call will
			// reset its state.
			p.fail(inst)
		}

		return inst
	}

	// No ready instances, so try one from the retry queue
	if len(p.retry) != 0 {
		// We don't want to disturb the retry schedule, but we
		// want some kind of fairness when resorting to failed
		// instances.  So pick one at random, and don't
		// reschedule it.
		return p.retry[p.rng.Intn(len(p.retry))]
	}

	// None available
	return nil
}

func (p *InstancePool) Succeeded(inst *PooledInstance) {
	p.lock.Lock()
	defer p.lock.Unlock()

	inst.failures = 0

	if !inst.retryTime.IsZero() {
		heap.Remove(&p.retryQueue, inst.index)
		p.resetTimer(p.now())
		inst.retryTime = time.Time{}
		inst.index = len(p.ready)
		p.ready = append(p.ready, inst)
	}
}

func (p *InstancePool) Failed(inst *PooledInstance) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if inst.retryTime.IsZero() {
		p.fail(inst)
	}
}

func (p *InstancePool) fail(inst *PooledInstance) {
	// inst must already be ready, i.e. inst.retryTime.isZero()
	p.ready[inst.index] = p.ready[len(p.ready)-1]
	p.ready = p.ready[:len(p.ready)-1]
	p.reschedule(inst)
	heap.Push(&p.retryQueue, inst)
	p.resetTimer(p.now())
}

func (p *InstancePool) reschedule(inst *PooledInstance) {
	delay := (1 << inst.failures) * retry_interval_base
	inst.failures++
	inst.retryTime = p.now().Add(delay)
}

func (p *InstancePool) UpdateInstances(instances []model.Instance) {
	p.lock.Lock()
	defer p.lock.Unlock()

	wantInsts := make(map[string]netutil.IPPort)
	for _, inst := range instances {
		wantInsts[inst.Name] = inst.Address
	}

	// Copy any common instances across
	var ready, retry []*PooledInstance
	keepInsts := func(insts []*PooledInstance) {
		for _, inst := range insts {
			if _, found := wantInsts[inst.Instance.Name]; found {
				delete(wantInsts, inst.Instance.Name)
				if inst.retryTime.IsZero() {
					inst.index = len(ready)
					ready = append(ready, inst)
				} else {
					inst.index = len(retry)
					retry = append(retry, inst)
				}
			}
		}
	}

	keepInsts(p.ready)
	keepInsts(p.retry)

	// Add new instances
	for name, addr := range wantInsts {
		ready = append(ready, &PooledInstance{
			Instance: model.Instance{Name: name, Address: addr},
			index:    len(ready),
		})
	}

	p.ready = ready
	p.retry = retry
	heap.Init(&p.retryQueue)
	p.resetTimer(p.now())
}

func (p *InstancePool) resetTimer(now time.Time) {
	if len(p.retry) == 0 {
		if p.timer != nil {
			// Can't pause a go Timer
			p.timer.Reset(24 * time.Hour)
		}

		return
	}

	delay := p.retry[0].retryTime.Sub(now)
	if p.timer == nil {
		timer := time.NewTimer(delay)
		p.timer = timer
		go func() {
			for {
				select {
				case now := <-timer.C:
					p.processRetries(now)
				case <-p.stopped:
					return
				}
			}
		}()
	} else {
		p.timer.Reset(delay)
	}
}

// Make any instances that are due for a retry available again
func (p *InstancePool) processRetries(now time.Time) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for len(p.retry) != 0 && !now.Before(p.retry[0].retryTime) {
		inst := p.retry[0]
		heap.Pop(&p.retryQueue)
		inst.retryTime = time.Time{}
		inst.index = len(p.ready)
		p.ready = append(p.ready, inst)
	}

	p.resetTimer(now)
}

func (q *retryQueue) Len() int {
	return len(q.retry)
}

func (q *retryQueue) Less(i, j int) bool {
	return q.retry[i].retryTime.Before(q.retry[j].retryTime)
}

func (q *retryQueue) Swap(i, j int) {
	a, b := q.retry[i], q.retry[j]
	q.retry[i], q.retry[j] = b, a
	a.index = j
	b.index = i
}

func (q *retryQueue) Push(x interface{}) {
	inst := x.(*PooledInstance)
	inst.index = len(q.retry)
	q.retry = append(q.retry, inst)
}

func (q *retryQueue) Pop() interface{} {
	inst := q.retry[len(q.retry)-1]
	q.retry = q.retry[:len(q.retry)-1]
	return inst
}
