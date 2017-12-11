package sling

import (
	"net/http"
	"net/url"
	"github.com/ansel1/merry"
	goquery "github.com/google/go-querystring/query"
)

type Option interface {
	Apply(b *Builder) error
}

type OptionFunc func(*Builder) error

func (f OptionFunc) Apply(b *Builder) error {
	return f(b)
}

func (s *Builder) With(opts ...Option) (*Builder, error) {
	s2 := s.Clone()
	err := s2.Apply(opts...)
	if err != nil {
		return nil, err
	}
	return s2, nil
}

func (s *Builder) Apply(opts ...Option) error {
	for _, o := range opts {
		err := o.Apply(s)
		if err != nil {
			return merry.Prepend(err, "applying options")
		}
	}
	return nil
}

func Method(m string, paths ...string) Option {
	return OptionFunc(func(b *Builder) error {
		b.Method = m
		for _, p := range paths {
			err := RelativeURLString(p).Apply(b)
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

func Header(h http.Header) Option {
	return OptionFunc(func(b *Builder) error {
		b.Header = h
		return nil
	})
}

func AddHeader(key, value string) Option {
	return OptionFunc(func(b *Builder) error {
		if b.Header == nil {
			b.Header = make(http.Header)
		}
		b.Header.Add(key, value)
		return nil
	})
}

func SetHeader(key, value string) Option {
	return OptionFunc(func(b *Builder) error {
		if b.Header == nil {
			b.Header = make(http.Header)
		}
		b.Header.Set(key, value)
		return nil
	})
}

func DeleteHeader(key string) Option {
	return OptionFunc(func(b *Builder) error {
		b.Header.Del(key)
		return nil
	})
}

func BasicAuth(username, password string) Option {
	return SetHeader("Authorization", "Basic " + basicAuth(username, password))
}

func BearerAuth(token string) Option {
	if token == "" {
		return DeleteHeader("Authorization")
	}
	return SetHeader("Authorization", "Bearer " + token)
}

func URL(u *url.URL) Option {
	return OptionFunc(func(b *Builder) error {
		b.URL = u
		return nil
	})
}

func RelativeURL(u *url.URL) Option {
	return OptionFunc(func(b *Builder) error {
		switch {
		case b.URL == nil:
			b.URL = u
		case u == nil:
		default:
			b.URL = b.URL.ResolveReference(u)
		}
		return nil
	})
}

func URLString(p string) Option {
	return OptionFunc(func(b *Builder) error {
		u, err := url.Parse(p)
		if err != nil {
			return merry.Prepend(err, "invalid url")
		}
		b.URL = u
		return nil
	})
}

func RelativeURLString(p string) Option {
	return OptionFunc(func(b *Builder) error {
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
	return OptionFunc(func(s *Builder) error {
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
					return merry.Prepend(err,"invalid query struct")
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
	return OptionFunc(func(b *Builder) error {
		b.Body = body
		return nil
	})
}

func WithMarshaler(m Marshaler) Option {
	return OptionFunc(func(b *Builder) error {
		b.Marshaler = m
		return nil
	})
}

func WithUnmarshaler(m Unmarshaler) Option {
	return OptionFunc(func(b *Builder) error {
		b.Unmarshaler = m
		return nil
	})
}

func JSON(indent bool) Option {
	return WithMarshaler(&JSONMarshaler{Indent:indent})
}

func XML(indent bool) Option {
	return WithMarshaler(&XMLMarshaler{Indent:indent})
}

func Form() Option {
	return WithMarshaler(&FormMarshaler{})
}
