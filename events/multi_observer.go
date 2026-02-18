package events

import (
	"sync"

	"github.com/9seconds/mtg/v2/mtglib"
)

type multiObserver struct {
	observers []Observer
}

func (m multiObserver) EventStart(evt mtglib.EventStart) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventStart(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventConnectedToDC(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventDomainFronting(evt mtglib.EventDomainFronting) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventDomainFronting(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventTraffic(evt mtglib.EventTraffic) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventTraffic(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventFinish(evt mtglib.EventFinish) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventFinish(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventConcurrencyLimited(evt mtglib.EventConcurrencyLimited) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventConcurrencyLimited(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventIPBlocklisted(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventReplayAttack(evt mtglib.EventReplayAttack) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventReplayAttack(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) EventIPListSize(evt mtglib.EventIPListSize) {
	wg := &sync.WaitGroup{}

	for _, v := range m.observers {
		wg.Go(func() {
			v.EventIPListSize(evt)
		})
	}

	wg.Wait()
}

func (m multiObserver) Shutdown() {
	for _, v := range m.observers {
		v.Shutdown()
	}
}

func newMultiObserver(factories []ObserverFactory) Observer {
	observers := make([]Observer, len(factories))

	for i, v := range factories {
		observers[i] = v()
	}

	return multiObserver{
		observers: observers,
	}
}
