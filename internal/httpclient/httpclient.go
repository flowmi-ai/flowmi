package httpclient

import (
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultBodyLimit = 100 << 20 // 100 MB
)

// New creates a resty client with default configuration.
func New() *resty.Client {
	return resty.New().
		SetTimeout(defaultTimeout).
		SetResponseBodyLimit(defaultBodyLimit)
}
