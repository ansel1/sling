package clients

import (
	"net"
	"net/http"
	"time"
)

func NewClient(opts ...Option) (*http.Client, error) {
	// fyi: first iteration of this made a shallow copy
	// of http.DefaultTransport, but `go vet` complains that
	// we're making a copy of mutex lock in Transport (legit).
	// So we're just copying the init code.  Need to keep an eye
	// on this in future golang releases

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	c := &http.Client{}

	for _, opt := range opts {
		err := opt.Apply(c, t)
		if err != nil {
			return nil, err
		}
	}

	// if one of the options explicitly sets the transport, that
	// overrides our transport
	if c.Transport != nil {
		c.Transport = t
	}
	return c, nil
}

type Option interface {
	Apply(*http.Client, *http.Transport) error
}

type ClientOptionFunc func(*http.Client, *http.Transport) error

func (f ClientOptionFunc) Apply(c *http.Client, t *http.Transport) error {
	return f(c, t)
}
