package dynamic_timeout

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestObserve(t *testing.T) {
	dt, _ := New(WithMaxHistory(3))
	assert.EqualValues(t, []time.Duration{defaultMaxTimeout, defaultMaxTimeout, defaultMaxTimeout}, dt.responseTimeHistory)

	dt.Observe(17)
	assert.EqualValues(t, []time.Duration{17, defaultMaxTimeout, defaultMaxTimeout}, dt.responseTimeHistory)

	dt.Observe(51)
	assert.EqualValues(t, []time.Duration{17, 51, defaultMaxTimeout}, dt.responseTimeHistory)

	dt.Observe(43)
	assert.EqualValues(t, []time.Duration{17, 51, 43}, dt.responseTimeHistory)

	dt.Observe(29)
	assert.EqualValues(t, []time.Duration{29, 51, 43}, dt.responseTimeHistory)

	dt.Observe(22)
	assert.EqualValues(t, []time.Duration{29, 22, 43}, dt.responseTimeHistory)
}

func TestGetTimeout(t *testing.T) {
	dt, _ := New(
		WithMinTimeout(10),
		WithMaxTimeout(100),
		WithMaxHistory(3),
		WithTimeoutFunc(func(responseTimeHistory []time.Duration) time.Duration {
			maximum := responseTimeHistory[0]
			for _, responseTime := range responseTimeHistory {
				if maximum < responseTime {
					maximum = responseTime
				}
			}
			return maximum
		}),
	)

	dt.Observe(1)
	dt.Observe(2)
	dt.Observe(3)
	assert.EqualValues(t, 10, dt.GetTimeout())

	dt.Observe(12)
	assert.EqualValues(t, 12, dt.GetTimeout())

	dt.Observe(123)
	assert.EqualValues(t, 100, dt.GetTimeout())
}

func TestDefaultTimeoutFunc(t *testing.T) {
	timeout := defaultTimeoutFunc([]time.Duration{
		0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100,
	})

	assert.EqualValues(t, 285, timeout)
}
