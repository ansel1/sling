package sling

import (
	"context"
	"net/http"
)

func Request(opts ...Option) (*http.Request, error) {
	return RequestContext(context.Background(), opts...)
}

func RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {
	r, err := New(opts...)
	if err != nil {
		return nil, err
	}
	return r.RequestContext(ctx)
}

func Do(opts ...Option) (*http.Response, error) {
	return DoContext(context.Background(), opts...)
}

func DoContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	r, err := New(opts...)
	if err != nil {
		return nil, err
	}
	return r.DoContext(ctx, opts...)
}

func ReceiveContext(ctx context.Context, successV interface{}, opts ...Option) (*http.Response, string, error) {
	r, err := New(opts...)
	if err != nil {
		return nil, "", err
	}
	return r.ReceiveContext(ctx, successV, opts...)
}

func Receive(successV interface{}, opts ...Option) (*http.Response, string, error) {
	return ReceiveContext(context.Background(), successV, opts...)
}

func ReceiveFull(successV, failureV interface{}, opts ...Option) (*http.Response, string, error) {
	return ReceiveFullContext(context.Background(), successV, failureV, opts...)
}

func ReceiveFullContext(ctx context.Context, successV, failureV interface{}, opts ...Option) (*http.Response, string, error) {
	r, err := New(opts...)
	if err != nil {
		return nil, "", err
	}
	return r.ReceiveFullContext(ctx, successV, failureV, opts...)
}
