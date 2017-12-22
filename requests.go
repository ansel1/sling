package sling

import (
	"bytes"
	"context"
	"github.com/ansel1/merry"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// Requests is an HTTP Request builder and sender.
type Requests struct {
	// Doer holds the HTTP client for used to execute requests.
	// Defaults to http.DefaultClient.
	Doer Doer

	// Middleware wraps the Doer.  Middleware will be invoked in the order
	// it is in this slice.
	Middleware []Middleware

	// Method defaults to "GET".
	Method string
	URL    *url.URL

	// Header supplies the request headers.  If the Content-Type header
	// is explicitly set here, it will override the Content-Type header
	// supplied by the Marshaler.
	Header http.Header

	// advanced options, not typically used.  If not sure, leave them
	// blank.
	// Most of these settings are set automatically by the http package.
	// Setting them here will override the automatic values.
	GetBody          func() (io.ReadCloser, error)
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Host             string
	Trailer          http.Header

	// QueryParams are added to the request, in addition to any
	// query params already encoded in the URL
	QueryParams url.Values

	// Marshaler will be used to marshal the Body value into the body
	// of requests.  It is only used if the Body value is a struct value.
	// Defaults to the DefaultMarshaler, which marshals to JSON.
	//
	// If no Content-Type header has been explicitly set on Requests, the
	// Marshaler will supply an appropriate one.
	Marshaler BodyMarshaler

	// Unmarshaler will be used by the Receive methods to unmarshal
	// the response body.  Defaults to DefaultUnmarshaler, which unmarshals
	// multiple content types based on the Content-Type response header.
	Unmarshaler BodyUnmarshaler

	// Body can be set to a string, []byte, or io.Reader.  In these
	// cases, the value will be used as the body of the request.
	// Body can also be set to a struct.  In this case, the BodyMarshaler
	// will be used to marshal the value into the request body.
	Body interface{}
}

// New returns a new Requests.
func New(options ...Option) (*Requests, error) {
	b := &Requests{}
	err := b.Apply(options...)
	if err != nil {
		return nil, merry.Wrap(err)
	}
	return b, nil
}

func cloneURL(url *url.URL) *url.URL {
	if url == nil {
		return nil
	}
	urlCopy := *url
	return &urlCopy
}

func cloneValues(v url.Values) url.Values {
	if v == nil {
		return nil
	}
	v2 := make(url.Values, len(v))
	for key, value := range v {
		v2[key] = value
	}
	return v2
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	h2 := make(http.Header)
	for key, value := range h {
		h2[key] = value
	}
	return h2
}

// Clone returns a deep copy of a Requests.  Useful inheriting and adding settings from
// a parent Requests without modifying the parent.  For example,
//
// 	parent, _ := sling.New(Get("https://api.io/"))
// 	foo, _ := parent.Clone().Apply(Get("foo/"))
// 	bar, _ := parent.Clone().Apply(Post("bar/"))
//
// foo and bar will both use the same client, but send requests to
// https://api.io/foo/ and https://api.io/bar/ respectively.
func (r *Requests) Clone() *Requests {
	s2 := *r
	s2.Header = cloneHeader(r.Header)
	s2.Trailer = cloneHeader(r.Trailer)
	s2.URL = cloneURL(r.URL)
	s2.QueryParams = cloneValues(r.QueryParams)
	return &s2
}

// Requests

// Request returns a new http.Request.  If option arguments are passed,
// they will only by applied to this single request.
func (r *Requests) Request(opts ...Option) (*http.Request, error) {
	return r.RequestContext(context.Background())
}

// RequestContext does the same as Request, but requires a context.  Use this
// to set a request timeout:
//
//     req, err := r.RequestContext(context.WithTimeout(context.Background(), 10 * time.Seconds))
//
func (r *Requests) RequestContext(ctx context.Context, opts ...Option) (*http.Request, error) {
	reqs := r
	if len(opts) > 0 {
		var err error
		reqs, err = reqs.With(opts...)
		if err != nil {
			return nil, err
		}
	}
	// marshal body, if applicable
	bodyData, ct, err := reqs.getRequestBody()
	if err != nil {
		return nil, err
	}

	urlS := ""
	if reqs.URL != nil {
		urlS = reqs.URL.String()
	}

	req, err := http.NewRequest(reqs.Method, urlS, bodyData)
	if err != nil {
		return nil, err
	}

	// if we marshaled the body, use our content type
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	if reqs.ContentLength != 0 {
		req.ContentLength = reqs.ContentLength
	}

	if reqs.GetBody != nil {
		req.GetBody = reqs.GetBody
	}

	// copy the host
	if reqs.Host != "" {
		req.Host = reqs.Host
	}

	req.TransferEncoding = reqs.TransferEncoding
	req.Close = reqs.Close
	req.Trailer = reqs.Trailer

	// copy Headers pairs into new Header map
	for k, v := range reqs.Header {
		req.Header[k] = v
	}

	if len(reqs.QueryParams) > 0 {
		if req.URL.RawQuery != "" {
			req.URL.RawQuery += "&" + reqs.QueryParams.Encode()
		} else {
			req.URL.RawQuery = reqs.QueryParams.Encode()
		}
	}

	return req.WithContext(ctx), nil
}

// getRequestBody returns the io.Reader which should be used as the body
// of new Requests.
func (r *Requests) getRequestBody() (body io.Reader, contentType string, err error) {
	switch v := r.Body.(type) {
	case nil:
		return nil, "", nil
	case io.Reader:
		return v, "", nil
	case string:
		return strings.NewReader(v), "", nil
	case []byte:
		return bytes.NewReader(v), "", nil
	default:
		marshaler := r.Marshaler
		if marshaler == nil {
			marshaler = DefaultMarshaler
		}
		b, ct, err := marshaler.Marshal(r.Body)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(b), ct, err
	}
}

// DoContext executes a request with the Doer.  The response body is not closed:
// it is the callers responsibility to close the response body.
// If the caller prefers the body as a byte slice, or prefers the body
// unmarshaled into a struct, see the RecieveX methods below.
//
// Additional options arguments can be passed.  They will be applied to this request only.
func (r *Requests) DoContext(ctx context.Context, opts ...Option) (*http.Response, error) {
	// if there are request options, apply them now, rather than passing them
	// to RequestContext().  Options may modify the Middleware or the Doer, and
	// we want to honor those options as well as the ones which affect the request.
	reqs := r
	if len(opts) > 0 {
		var err error
		reqs, err = reqs.With(opts...)
		if err != nil {
			return nil, err
		}
	}
	req, err := reqs.RequestContext(ctx)
	if err != nil {
		return nil, err
	}
	doer := reqs.Doer
	if doer == nil {
		doer = http.DefaultClient
	}
	return Wrap(doer, reqs.Middleware...).Do(req)
}

// Do executes a request with the Doer.  The response body is not closed:
// it is the callers responsibility to close the response body.
// If the caller prefers the body as a byte slice, or prefers the body
// unmarshaled into a struct, see the Receive methods below.
//
// Additional options arguments can be passed.  They will be applied to this request only.
func (r *Requests) Do(opts ...Option) (*http.Response, error) {
	return r.DoContext(context.Background())
}

// ReceiveContext creates a new HTTP request and returns the response. Success
// responses (2XX) are unmarshaled into the value pointed to by successV.
// Any error creating the request, sending it, or decoding a 2XX response
// is returned.
//
// If option arguments are passed, they are applied to this single request only.
//
// The context argument can be used to set a request timeout.
func (r *Requests) ReceiveContext(ctx context.Context, successV interface{}, opts ...Option) (resp *http.Response, body []byte, err error) {
	return r.ReceiveFullContext(ctx, successV, nil)
}

// Receive is the same as ReceiveContext, but does not require a context.
func (r *Requests) Receive(successV interface{}, opts ...Option) (resp *http.Response, body []byte, err error) {
	return r.ReceiveFullContext(context.Background(), successV, nil, opts...)
}

// RecieveFull is the same as RecieveFullContext, but does not require a context.
func (r *Requests) ReceiveFull(successV, failureV interface{}, opts ...Option) (resp *http.Response, body []byte, err error) {
	return r.ReceiveFullContext(context.Background(), successV, failureV, opts...)

}

// ReceiveFullContext creates a new HTTP request and returns the response. Success
// responses (2XX) are decoded into the value pointed to by successV and
// other responses are decoded into the value pointed to by failureV.
// Any error creating the request, sending it, or decoding the response is
// returned.
// Receive is shorthand for calling RequestContext and DoContext.
func (r *Requests) ReceiveFullContext(ctx context.Context, successV, failureV interface{}, opts ...Option) (resp *http.Response, body []byte, err error) {
	resp, err = r.DoContext(ctx, opts...)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, body, err
	}

	var unmarshalInto interface{}
	if code := resp.StatusCode; 200 <= code && code <= 299 {
		unmarshalInto = successV
	} else {
		unmarshalInto = failureV
	}

	if unmarshalInto != nil {
		unmarshaler := r.Unmarshaler
		if unmarshaler == nil {
			unmarshaler = DefaultUnmarshaler
		}

		err = unmarshaler.Unmarshal(body, resp.Header.Get("Content-Type"), unmarshalInto)
	}
	return resp, body, err
}
