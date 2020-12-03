package common

import "time"

type (
	FlowMessage struct {
		Rule   string
		Fields FlowMessagePayload
	}

	FlowMessagePayload map[string]string

	PoolItem struct {
		Size int
		Last time.Time
		Items []FlowMessagePayload
	}

	PoolBump struct {
		Reason string
		Rule string
	}
)
