package testlib

import "github.com/stretchr/testify/mock"

type MtglibAntiReplayCacheMock struct {
	mock.Mock
}

func (m *MtglibAntiReplayCacheMock) SeenBefore(data []byte) bool {
	return m.Called(data).Bool(0)
}
