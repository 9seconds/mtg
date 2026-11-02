package events_test

import (
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/mock"
)

type ObserverMock struct {
	mock.Mock
}

func (o *ObserverMock) EventStart(evt mtglib.EventStart) {
	o.Called(evt)
}

func (o *ObserverMock) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	o.Called(evt)
}

func (o *ObserverMock) EventDomainFronting(evt mtglib.EventDomainFronting) {
	o.Called(evt)
}

func (o *ObserverMock) EventTraffic(evt mtglib.EventTraffic) {
	o.Called(evt)
}

func (o *ObserverMock) EventFinish(evt mtglib.EventFinish) {
	o.Called(evt)
}

func (o *ObserverMock) EventConcurrencyLimited(evt mtglib.EventConcurrencyLimited) {
	o.Called(evt)
}

func (o *ObserverMock) EventPendingHandshakeLimit(evt mtglib.EventPendingHandshakeLimit) {
	o.Called(evt)
}

func (o *ObserverMock) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	o.Called(evt)
}

func (o *ObserverMock) EventReplayAttack(evt mtglib.EventReplayAttack) {
	o.Called(evt)
}

func (o *ObserverMock) EventIPListSize(evt mtglib.EventIPListSize) {
	o.Called(evt)
}

func (o *ObserverMock) Shutdown() {
	o.Called()
}

// LegacyObserverMock implements only the required events.Observer methods,
// like an external observer written before the optional
// events.PendingHandshakeLimitObserver interface was introduced.
type LegacyObserverMock struct {
	mock.Mock
}

func (o *LegacyObserverMock) EventStart(evt mtglib.EventStart) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventConnectedToDC(evt mtglib.EventConnectedToDC) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventDomainFronting(evt mtglib.EventDomainFronting) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventTraffic(evt mtglib.EventTraffic) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventFinish(evt mtglib.EventFinish) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventConcurrencyLimited(evt mtglib.EventConcurrencyLimited) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventIPBlocklisted(evt mtglib.EventIPBlocklisted) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventReplayAttack(evt mtglib.EventReplayAttack) {
	o.Called(evt)
}

func (o *LegacyObserverMock) EventIPListSize(evt mtglib.EventIPListSize) {
	o.Called(evt)
}

func (o *LegacyObserverMock) Shutdown() {
	o.Called()
}
