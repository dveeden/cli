// Code generated by mocker. DO NOT EDIT.
// github.com/travisjeffery/mocker
// Source: test_helper.go

package mock

import (
	sync "sync"

	github_com_segmentio_analytics_go "github.com/segmentio/analytics-go"
)

// SegmentClient is a mock of SegmentClient interface
type SegmentClient struct {
	lockEnqueue sync.Mutex
	EnqueueFunc func(m github_com_segmentio_analytics_go.Message) error

	lockClose sync.Mutex
	CloseFunc func() error

	calls struct {
		Enqueue []struct {
			M github_com_segmentio_analytics_go.Message
		}
		Close []struct {
		}
	}
}

// Enqueue mocks base method by wrapping the associated func.
func (m_2 *SegmentClient) Enqueue(m github_com_segmentio_analytics_go.Message) error {
	m_2.lockEnqueue.Lock()
	defer m_2.lockEnqueue.Unlock()

	if m_2.EnqueueFunc == nil {
		panic("mocker: SegmentClient.EnqueueFunc is nil but SegmentClient.Enqueue was called.")
	}

	call := struct {
		M github_com_segmentio_analytics_go.Message
	}{
		M: m,
	}

	m_2.calls.Enqueue = append(m_2.calls.Enqueue, call)

	return m_2.EnqueueFunc(m)
}

// EnqueueCalled returns true if Enqueue was called at least once.
func (m_2 *SegmentClient) EnqueueCalled() bool {
	m_2.lockEnqueue.Lock()
	defer m_2.lockEnqueue.Unlock()

	return len(m_2.calls.Enqueue) > 0
}

// EnqueueCalls returns the calls made to Enqueue.
func (m_2 *SegmentClient) EnqueueCalls() []struct {
	M github_com_segmentio_analytics_go.Message
} {
	m_2.lockEnqueue.Lock()
	defer m_2.lockEnqueue.Unlock()

	return m_2.calls.Enqueue
}

// Close mocks base method by wrapping the associated func.
func (m *SegmentClient) Close() error {
	m.lockClose.Lock()
	defer m.lockClose.Unlock()

	if m.CloseFunc == nil {
		panic("mocker: SegmentClient.CloseFunc is nil but SegmentClient.Close was called.")
	}

	call := struct {
	}{}

	m.calls.Close = append(m.calls.Close, call)

	return m.CloseFunc()
}

// CloseCalled returns true if Close was called at least once.
func (m *SegmentClient) CloseCalled() bool {
	m.lockClose.Lock()
	defer m.lockClose.Unlock()

	return len(m.calls.Close) > 0
}

// CloseCalls returns the calls made to Close.
func (m *SegmentClient) CloseCalls() []struct {
} {
	m.lockClose.Lock()
	defer m.lockClose.Unlock()

	return m.calls.Close
}

// Reset resets the calls made to the mocked methods.
func (m *SegmentClient) Reset() {
	m.lockEnqueue.Lock()
	m.calls.Enqueue = nil
	m.lockEnqueue.Unlock()
	m.lockClose.Lock()
	m.calls.Close = nil
	m.lockClose.Unlock()
}