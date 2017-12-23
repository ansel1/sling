package sling

import (
	"encoding/base64"
	"github.com/ansel1/merry"
	"github.com/ansel1/sling/clients"
	goquery "github.com/google/go-querystring/query"
	"net/http"
	"net/url"
)

const (
	HeaderAccept      = "Accept"
	HeaderContentType = "Content-Type"

	ContentTypeJSON = "application/json"
	ContentTypeXML  = "application/xml"
	ContentTypeForm = "application/x-www-form-urlencoded"
)

type Option interface {
	Apply(*Requests) error
}

type OptionFunc func(*Requests) error

func (f OptionFunc) Apply(r *Requests) error {
	return f(r)
}

func (r *Requests) With(opts ...Option) (*Requests, error) {
	r2 := r.Clone()
	err := r2.Apply(opts...)
	if err != nil {
		return nil, err
	}
	return r2, nil
}

func (r *Requests) Apply(opts ...Option) error {
	for _, o := range opts {
		err := o.Apply(r)
		if err != nil {
			return merry.Prepend(err, "applying options")
		}
	}
	return nil
}

func Method(m string, paths ...string) Option {
	return OptionFunc(func(r *Requests) error {
		r.Method = m
		for _, p := range paths {
			err := RelativeURL(p).Apply(r)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func Head(paths ...string) Option {
	return Method("HEAD", paths...)
}

func Get(paths ...string) Option {
	return Method("GET", paths...)
}

func Post(paths ...string) Option {
	return Method("POST", paths...)
}

func Put(paths ...string) Option {
	return Method("PUT", paths...)
}

func Patch(paths ...string) Option {
	return Method("PATCH", paths...)
}

func Delete(paths ...string) Option {
	return Method("DELETE", paths...)
}

func AddHeader(key, value string) Option {
	return OptionFunc(func(b *Requests) error {
		if b.Header == nil {
			b.Header = make(http.Header)
		}
		b.Header.Add(key, value)
		return nil
	})
}

func Header(key, value string) Option {
	return OptionFunc(func(b *Requests) error {
		if b.Header == nil {
			b.Header = make(http.Header)
		}
		b.Header.Set(key, value)
		return nil
	})
}

func DeleteHeader(key string) Option {
	return OptionFunc(func(b *Requests) error {
		b.Header.Del(key)
		return nil
	})
}

func BasicAuth(username, password string) Option {
	return Header("Authorization", "Basic "+basicAuth(username, password))
}

// basicAuth returns the base64 encoded username:password for basic auth copied
// from net/http.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func BearerAuth(token string) Option {
	if token == "" {
		return DeleteHeader("Authorization")
	}
	return Header("Authorization", "Bearer "+token)
}

func URL(p string) Option {
	return OptionFunc(func(b *Requests) error {
		u, err := url.Parse(p)
		if err != nil {
			return merry.Prepend(err, "invalid url")
		}
		b.URL = u
		return nil
	})
}

func RelativeURL(p string) Option {
	return OptionFunc(func(b *Requests) error {
		u, err := url.Parse(p)
		if err != nil {
			return merry.Prepend(err, "invalid url")
		}
		if b.URL == nil {
			b.URL = u
		} else {
			b.URL = b.URL.ResolveReference(u)
		}
		return nil
	})
}

func QueryParams(queryStructs ...interface{}) Option {
	return OptionFunc(func(s *Requests) error {
		if s.QueryParams == nil {
			s.QueryParams = url.Values{}
		}
		for _, queryStruct := range queryStructs {
			var values url.Values
			switch t := queryStruct.(type) {
			case nil:
			case map[string][]string:
				values = url.Values(t)
			case url.Values:
				values = t
			default:
				// encodes query structs into a url.Values map and merges maps
				var err error
				values, err = goquery.Values(queryStruct)
				if err != nil {
					return merry.Prepend(err, "invalid query struct")
				}
			}

			// merges new values into existing
			for key, values := range values {
				for _, value := range values {
					s.QueryParams.Add(key, value)
				}
			}
		}
		return nil
	})
}

func Body(body interface{}) Option {
	return OptionFunc(func(b *Requests) error {
		b.Body = body
		return nil
	})
}

func Marshaler(m BodyMarshaler) Option {
	return OptionFunc(func(b *Requests) error {
		b.Marshaler = m
		return nil
	})
}

func Unmarshaler(m BodyUnmarshaler) Option {
	return OptionFunc(func(b *Requests) error {
		b.Unmarshaler = m
		return nil
	})
}

func Accept(accept string) Option {
	return Header("Accept", accept)
}

func ContentType(contentType string) Option {
	return Header("Content-Type", contentType)
}

func Host(host string) Option {
	return OptionFunc(func(b *Requests) error {
		b.Host = host
		return nil
	})
}

func JSON(indent bool) Option {
	return Marshaler(&JSONMarshaler{Indent: indent})
}

func XML(indent bool) Option {
	return Marshaler(&XMLMarshaler{Indent: indent})
}

func Form() Option {
	return Marshaler(&FormMarshaler{})
}

func Client(opts ...clients.Option) Option {
	return OptionFunc(func(b *Requests) error {
		c, err := clients.NewClient(opts...)
		if err != nil {
			return err
		}
		b.Doer = c
		return nil
	})
}

func Use(m ...Middleware) Option {
	return OptionFunc(func(r *Requests) error {
		r.Middleware = append(r.Middleware, m...)
		return nil
	})
}

func WithDoer(d Doer) Option {
	return OptionFunc(func(r *Requests) error {
		r.Doer = d
		return nil
	})
}
