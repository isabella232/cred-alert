// This file was generated by counterfeiter
package queuefakes

import (
	"cred-alert/queue"
	"sync"

	"cloud.google.com/go/pubsub"
)

type FakeAcker struct {
	AckStub        func(*pubsub.Message, bool)
	ackMutex       sync.RWMutex
	ackArgsForCall []struct {
		arg1 *pubsub.Message
		arg2 bool
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeAcker) Ack(arg1 *pubsub.Message, arg2 bool) {
	fake.ackMutex.Lock()
	fake.ackArgsForCall = append(fake.ackArgsForCall, struct {
		arg1 *pubsub.Message
		arg2 bool
	}{arg1, arg2})
	fake.recordInvocation("Ack", []interface{}{arg1, arg2})
	fake.ackMutex.Unlock()
	if fake.AckStub != nil {
		fake.AckStub(arg1, arg2)
	}
}

func (fake *FakeAcker) AckCallCount() int {
	fake.ackMutex.RLock()
	defer fake.ackMutex.RUnlock()
	return len(fake.ackArgsForCall)
}

func (fake *FakeAcker) AckArgsForCall(i int) (*pubsub.Message, bool) {
	fake.ackMutex.RLock()
	defer fake.ackMutex.RUnlock()
	return fake.ackArgsForCall[i].arg1, fake.ackArgsForCall[i].arg2
}

func (fake *FakeAcker) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.ackMutex.RLock()
	defer fake.ackMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeAcker) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ queue.Acker = new(FakeAcker)
