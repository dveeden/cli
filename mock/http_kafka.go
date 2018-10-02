// Code generated by mocker; DO NOT EDIT
// github.com/travisjeffery/mocker
package mock

import (
	schedv1 "github.com/confluentinc/cc-structs/kafka/scheduler/v1"
	"net/http"
	"sync"
)

var (
	lockKafkaDescribe sync.RWMutex
	lockKafkaList     sync.RWMutex
)

// Kafka is a mock implementation of Kafka.
//
//     func TestSomethingThatUsesKafka(t *testing.T) {
//
//         // make and configure a mocked Kafka
//         mockedKafka := &Kafka{
//             DescribeFunc: func(cluster *schedv1.KafkaCluster) (*schedv1.KafkaCluster, *http.Response, error) {
// 	               panic("TODO: mock out the Describe method")
//             },
//             ListFunc: func(cluster *schedv1.KafkaCluster) ([]*schedv1.KafkaCluster, *http.Response, error) {
// 	               panic("TODO: mock out the List method")
//             },
//         }
//
//         // TODO: use mockedKafka in code that requires Kafka
//         //       and then make assertions.
//
//     }
type Kafka struct {
	// DescribeFunc mocks the Describe method.
	DescribeFunc func(cluster *schedv1.KafkaCluster) (*schedv1.KafkaCluster, *http.Response, error)

	// ListFunc mocks the List method.
	ListFunc func(cluster *schedv1.KafkaCluster) ([]*schedv1.KafkaCluster, *http.Response, error)

	// calls tracks calls to the methods.
	calls struct {
		// Describe holds details about calls to the Describe method.
		Describe []struct {
			// Cluster is the cluster argument value.
			Cluster *schedv1.KafkaCluster
		}
		// List holds details about calls to the List method.
		List []struct {
			// Cluster is the cluster argument value.
			Cluster *schedv1.KafkaCluster
		}
	}
}

// Reset resets the calls made to the mocked APIs.
func (mock *Kafka) Reset() {
	lockKafkaDescribe.Lock()
	mock.calls.Describe = nil
	lockKafkaDescribe.Unlock()
	lockKafkaList.Lock()
	mock.calls.List = nil
	lockKafkaList.Unlock()
}

// Describe calls DescribeFunc.
func (mock *Kafka) Describe(cluster *schedv1.KafkaCluster) (*schedv1.KafkaCluster, *http.Response, error) {
	if mock.DescribeFunc == nil {
		panic("moq: Kafka.DescribeFunc is nil but Kafka.Describe was just called")
	}
	callInfo := struct {
		Cluster *schedv1.KafkaCluster
	}{
		Cluster: cluster,
	}
	lockKafkaDescribe.Lock()
	mock.calls.Describe = append(mock.calls.Describe, callInfo)
	lockKafkaDescribe.Unlock()
	return mock.DescribeFunc(cluster)
}

// DescribeCalled returns true if at least one call was made to Describe.
func (mock *Kafka) DescribeCalled() bool {
	lockKafkaDescribe.RLock()
	defer lockKafkaDescribe.RUnlock()
	return len(mock.calls.Describe) > 0
}

// DescribeCalls gets all the calls that were made to Describe.
// Check the length with:
//     len(mockedKafka.DescribeCalls())
func (mock *Kafka) DescribeCalls() []struct {
	Cluster *schedv1.KafkaCluster
} {
	var calls []struct {
		Cluster *schedv1.KafkaCluster
	}
	lockKafkaDescribe.RLock()
	calls = mock.calls.Describe
	lockKafkaDescribe.RUnlock()
	return calls
}

// List calls ListFunc.
func (mock *Kafka) List(cluster *schedv1.KafkaCluster) ([]*schedv1.KafkaCluster, *http.Response, error) {
	if mock.ListFunc == nil {
		panic("moq: Kafka.ListFunc is nil but Kafka.List was just called")
	}
	callInfo := struct {
		Cluster *schedv1.KafkaCluster
	}{
		Cluster: cluster,
	}
	lockKafkaList.Lock()
	mock.calls.List = append(mock.calls.List, callInfo)
	lockKafkaList.Unlock()
	return mock.ListFunc(cluster)
}

// ListCalled returns true if at least one call was made to List.
func (mock *Kafka) ListCalled() bool {
	lockKafkaList.RLock()
	defer lockKafkaList.RUnlock()
	return len(mock.calls.List) > 0
}

// ListCalls gets all the calls that were made to List.
// Check the length with:
//     len(mockedKafka.ListCalls())
func (mock *Kafka) ListCalls() []struct {
	Cluster *schedv1.KafkaCluster
} {
	var calls []struct {
		Cluster *schedv1.KafkaCluster
	}
	lockKafkaList.RLock()
	calls = mock.calls.List
	lockKafkaList.RUnlock()
	return calls
}
