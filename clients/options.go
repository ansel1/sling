package clients

import (
	"net/http/cookiejar"
	"net/url"
	"net/http"
	"time"
	"crypto/tls"
	"github.com/ansel1/merry"
)

func NoRedirects() Option {
	return ClientOptionFunc(func(client *http.Client, _ *http.Transport) error {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return nil
	})
}

func MaxRedirects(max int) Option {
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

func Cookies(opts *cookiejar.Options) Option {
	return ClientOptionFunc(func(client *http.Client, _ *http.Transport) error {
		jar, err := cookiejar.New(opts)
		if err != nil {
			return merry.Wrap(err)
		}
		client.Jar = jar
		return nil
	})
}

func ProxyURL(proxyURL string) Option {
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

func ProxyFunc(f func(request *http.Request) (*url.URL, error)) Option {
	return ClientOptionFunc(func(_ *http.Client, transport *http.Transport) error {
		transport.Proxy = f
		return nil
	})
}

func Timeout(d time.Duration) Option {
	return ClientOptionFunc(func(client *http.Client, transport *http.Transport) error {
		client.Timeout = d
		return nil
	})
}

func SkipVerify() Option {
	return ClientOptionFunc(func(client *http.Client, transport *http.Transport) error {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
		return nil
	})
}

