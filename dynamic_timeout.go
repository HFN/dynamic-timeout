package dynamic_timeout

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	defaultMinTimeout  = 100 * time.Millisecond
	defaultMaxTimeout  = 1 * time.Second
	defaultMaxHistory  = 100
	defaultTimeoutFunc = func(responseTimeHistory []time.Duration) time.Duration {
		sort.Slice(responseTimeHistory, func(i, j int) bool {
			return responseTimeHistory[i] < responseTimeHistory[j]
		})

		ninetyFifthPercentileIndex := 95 * len(responseTimeHistory) / 100
		return 3 * responseTimeHistory[ninetyFifthPercentileIndex]
	}
)

type (
	// DynamicTimeout represents a struct responsible for maintaining response time and calculating a dynamic timeout
	DynamicTimeout struct {
		minTimeout  time.Duration
		maxTimeout  time.Duration
		maxHistory  int
		timeoutFunc TimeOutFunc

		responseTimeHistory []time.Duration
		currentHistoryIndex int
		lock                *sync.Mutex
	}

	// TimeOutFunc represents a function which calculate a timeout based on a given history of response times
	// Given history array is not in the order of their observation necessarily
	// If there are not enough history items, Then the rest of the array is filled with maxTimeout
	TimeOutFunc func (responseTimeHistory []time.Duration) time.Duration

	// Option is the type of constructor options for New(...)
	Option func (dt *DynamicTimeout)
)

// New returns a new DynamicTimeout
func New(options ...Option) (*DynamicTimeout, error) {
	dt := &DynamicTimeout{
		minTimeout:  defaultMinTimeout,
		maxTimeout:  defaultMaxTimeout,
		maxHistory:  defaultMaxHistory,
		timeoutFunc: defaultTimeoutFunc,
	}

	for _, option := range options {
		option(dt)
	}

	if dt.maxHistory <= 0 {
		return nil, errors.New("maxHistory is not positive")
	}
	if dt.minTimeout <= 0 {
		return nil, errors.New("minTimeout is not positive")
	}
	if dt.maxTimeout <= 0 {
		return nil, errors.New("maxTimeout is not positive")
	}
	if dt.minTimeout > dt.maxTimeout {
		return nil, errors.New("minTimeout is greater than maxTimeout")
	}
	if dt.timeoutFunc == nil {
		return nil, errors.New("timeoutFunc is nil")
	}

	dt.responseTimeHistory = make([]time.Duration, dt.maxHistory, dt.maxHistory)
	for index := range dt.responseTimeHistory {
		dt.responseTimeHistory[index] = dt.maxTimeout
	}
	dt.currentHistoryIndex = 0
	dt.lock = &sync.Mutex{}

	return dt, nil
}

// WithMinTimeout sets minimum allowed timeout
func WithMinTimeout(minTimeout time.Duration) Option {
	return func(dt *DynamicTimeout) {
		dt.minTimeout = minTimeout
	}
}

// WithMaxTimeout sets maximum allowed timeout
func WithMaxTimeout(maxTimeout time.Duration) Option {
	return func(dt *DynamicTimeout) {
		dt.maxTimeout = maxTimeout
	}
}

// WithMaxHistory sets maximum number of history item that should be stored in memory
func WithMaxHistory(maxHistory int) Option {
	return func(dt *DynamicTimeout) {
		dt.maxHistory = maxHistory
	}
}

// WithTimeoutFunc sets a custom TimeOutFunc to calculate timeout
func WithTimeoutFunc(timeoutFunc TimeOutFunc) Option {
	return func(dt *DynamicTimeout) {
		dt.timeoutFunc = timeoutFunc
	}
}

// Observe observes the given responseTime and stores it inside history
// It is strongly recommended observing responseTime of all successful and timed out requests
func (dt *DynamicTimeout) Observe(responseTime time.Duration) {
	dt.lock.Lock()

	dt.responseTimeHistory[dt.currentHistoryIndex] = responseTime
	dt.currentHistoryIndex = (dt.currentHistoryIndex + 1) % dt.maxHistory

	dt.lock.Unlock()
}

// GetTimeout calls TimeOutFunc to return an appropriate timeout
func (dt *DynamicTimeout) GetTimeout() time.Duration {
	dt.lock.Lock()

	responseTimeHistory := make([]time.Duration, dt.maxHistory)
	copy(responseTimeHistory, dt.responseTimeHistory)

	dt.lock.Unlock()

	timeout := dt.timeoutFunc(responseTimeHistory)
	if timeout < dt.minTimeout {
		return dt.minTimeout
	}
	if timeout > dt.maxTimeout {
		return dt.maxTimeout
	}

	return timeout
}
