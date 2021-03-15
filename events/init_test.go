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

func (o *ObserverMock) EventFinish(evt mtglib.EventStart) {
	o.Called(evt)
}

func (o *ObserverMock) Shutdown() {
	o.Called()
}
