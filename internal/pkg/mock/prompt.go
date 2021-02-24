// Code generated by mocker. DO NOT EDIT.
// github.com/travisjeffery/mocker
// Source: prompt.go

package mock

import (
	sync "sync"
)

// Prompt is a mock of Prompt interface
type Prompt struct {
	lockReadLine sync.Mutex
	ReadLineFunc func() (string, error)

	lockReadLineMasked sync.Mutex
	ReadLineMaskedFunc func() (string, error)

	lockIsPipe sync.Mutex
	IsPipeFunc func() (bool, error)

	calls struct {
		ReadLine []struct {
		}
		ReadLineMasked []struct {
		}
		IsPipe []struct {
		}
	}
}

// ReadLine mocks base method by wrapping the associated func.
func (m *Prompt) ReadLine() (string, error) {
	m.lockReadLine.Lock()
	defer m.lockReadLine.Unlock()

	if m.ReadLineFunc == nil {
		panic("mocker: Prompt.ReadLineFunc is nil but Prompt.ReadLine was called.")
	}

	call := struct {
	}{}

	m.calls.ReadLine = append(m.calls.ReadLine, call)

	return m.ReadLineFunc()
}

// ReadLineCalled returns true if ReadLine was called at least once.
func (m *Prompt) ReadLineCalled() bool {
	m.lockReadLine.Lock()
	defer m.lockReadLine.Unlock()

	return len(m.calls.ReadLine) > 0
}

// ReadLineCalls returns the calls made to ReadLine.
func (m *Prompt) ReadLineCalls() []struct {
} {
	m.lockReadLine.Lock()
	defer m.lockReadLine.Unlock()

	return m.calls.ReadLine
}

// ReadLineMasked mocks base method by wrapping the associated func.
func (m *Prompt) ReadLineMasked() (string, error) {
	m.lockReadLineMasked.Lock()
	defer m.lockReadLineMasked.Unlock()

	if m.ReadLineMaskedFunc == nil {
		panic("mocker: Prompt.ReadLineMaskedFunc is nil but Prompt.ReadLineMasked was called.")
	}

	call := struct {
	}{}

	m.calls.ReadLineMasked = append(m.calls.ReadLineMasked, call)

	return m.ReadLineMaskedFunc()
}

// ReadLineMaskedCalled returns true if ReadLineMasked was called at least once.
func (m *Prompt) ReadLineMaskedCalled() bool {
	m.lockReadLineMasked.Lock()
	defer m.lockReadLineMasked.Unlock()

	return len(m.calls.ReadLineMasked) > 0
}

// ReadLineMaskedCalls returns the calls made to ReadLineMasked.
func (m *Prompt) ReadLineMaskedCalls() []struct {
} {
	m.lockReadLineMasked.Lock()
	defer m.lockReadLineMasked.Unlock()

	return m.calls.ReadLineMasked
}

// IsPipe mocks base method by wrapping the associated func.
func (m *Prompt) IsPipe() (bool, error) {
	m.lockIsPipe.Lock()
	defer m.lockIsPipe.Unlock()

	if m.IsPipeFunc == nil {
		panic("mocker: Prompt.IsPipeFunc is nil but Prompt.IsPipe was called.")
	}

	call := struct {
	}{}

	m.calls.IsPipe = append(m.calls.IsPipe, call)

	return m.IsPipeFunc()
}

// IsPipeCalled returns true if IsPipe was called at least once.
func (m *Prompt) IsPipeCalled() bool {
	m.lockIsPipe.Lock()
	defer m.lockIsPipe.Unlock()

	return len(m.calls.IsPipe) > 0
}

// IsPipeCalls returns the calls made to IsPipe.
func (m *Prompt) IsPipeCalls() []struct {
} {
	m.lockIsPipe.Lock()
	defer m.lockIsPipe.Unlock()

	return m.calls.IsPipe
}

// Reset resets the calls made to the mocked methods.
func (m *Prompt) Reset() {
	m.lockReadLine.Lock()
	m.calls.ReadLine = nil
	m.lockReadLine.Unlock()
	m.lockReadLineMasked.Lock()
	m.calls.ReadLineMasked = nil
	m.lockReadLineMasked.Unlock()
	m.lockIsPipe.Lock()
	m.calls.IsPipe = nil
	m.lockIsPipe.Unlock()
}