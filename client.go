package sling

import (
	"crypto/tls"
	"github.com/ansel1/merry"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

func NewClient(opts ...ClientOption) (*http.Client, error) {
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

type ClientOption interface {
	Apply(*http.Client, *http.Transport) error
}

type ClientOptionFunc func(*http.Client, *http.Transport) error

func (f ClientOptionFunc) Apply(c *http.Client, t *http.Transport) error {
	return f(c, t)
}

func NoRedirects() ClientOption {
	return ClientOptionFunc(func(client *http.Client, _ *http.Transport) error {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return nil
	})
}

func MaxRedirects(max int) ClientOption {
	return ClientOptionFunc(func(client *http.Client, _ *http.Transport) error {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= max {
				return merry.Errorf("stopped after max %d requests", len(via))
			}
			return nil
		}
		return nil
	})
}

func Cookies(opts *cookiejar.Options) ClientOption {
	return ClientOptionFunc(func(client *http.Client, _ *http.Transport) error {
		jar, err := cookiejar.New(opts)
		if err != nil {
			return merry.Wrap(err)
		}
		client.Jar = jar
		return nil
	})
}

func ProxyURL(proxyURL string) ClientOption {
	return ClientOptionFunc(func(_ *http.Client, transport *http.Transport) error {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return merry.Wrap(err)
		}
		transport.Proxy = func(request *http.Request) (*url.URL, error) {
			return u, nil
		}
		return nil
	})
}

func ProxyFunc(f func(request *http.Request) (*url.URL, error)) ClientOption {
	return ClientOptionFunc(func(_ *http.Client, transport *http.Transport) error {
		transport.Proxy = f
		return nil
	})
}

func Timeout(d time.Duration) ClientOption {
	return ClientOptionFunc(func(client *http.Client, transport *http.Transport) error {
		client.Timeout = d
		return nil
	})
}

func SkipVerify() ClientOption {
	return ClientOptionFunc(func(client *http.Client, transport *http.Transport) error {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
		return nil
	})
}
