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
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventStart(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventConnectedToDC(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventDomainFronting(evt mtglib.EventDomainFronting) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventDomainFronting(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventTraffic(evt mtglib.EventTraffic) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventTraffic(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventFinish(evt mtglib.EventFinish) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventFinish(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventConcurrencyLimited(evt mtglib.EventConcurrencyLimited) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventConcurrencyLimited(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventIPBlocklisted(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventReplayAttack(evt mtglib.EventReplayAttack) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventReplayAttack(evt)
		}(v)
	}

	wg.Wait()
}

func (m multiObserver) EventIPListSize(evt mtglib.EventIPListSize) {
	wg := &sync.WaitGroup{}
	wg.Add(len(m.observers))

	for _, v := range m.observers {
		go func(obs Observer) {
			defer wg.Done()

			obs.EventIPListSize(evt)
		}(v)
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
