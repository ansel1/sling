package sling

import (
	"context"
	"net/http"
)

func DoContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	r, err := New(opts...)
	if err != nil {
		return nil, err
	}
	return r.DoContext(nil, opts...)
}

func Do(opts ...Option) (*http.Response, error) {
	r, err := New(opts...)
	if err != nil {
		return nil, err
	}
	return r.Do(opts...)
}
