package sling

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"context"
	"io/ioutil"
	"strings"
	"github.com/ansel1/merry"
)

const (
	contentType     = "Content-Type"
	jsonContentType = "application/json"
	xmlContentType  = "application/xml"
	formContentType = "application/x-www-form-urlencoded"
)



// Builder is an HTTP Request builder and sender.
type Builder struct {
	// http Client for doing requests
	Doer Doer

	Method           string
	URL              *url.URL
	Header           http.Header

	// advanced options, not typically used.  If not sure, leave them
	// blank
	GetBody          func() (io.ReadCloser, error)
	ContentLength    int64
	TransferEncoding []string
	Close            bool
	Host             string
	Trailer          http.Header

	// QueryParams are added to the request, in addition to any
	// query params already encoded in the URL
	QueryParams url.Values


	Marshaler   Marshaler
	Unmarshaler Unmarshaler

	Body interface{}
}

// New returns a new Builder with an http DefaultClient.
func New(options ...Option) (*Builder, error) {
	b := &Builder{}
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

func NewFromRequest(req *http.Request) *Builder {
	return &Builder{
		Method:           req.Method,
		Header:           cloneHeader(req.Header),
		GetBody:          req.GetBody,
		ContentLength:    req.ContentLength,
		TransferEncoding: req.TransferEncoding,
		Close:            req.Close,
		Host:             req.Host,
		Trailer:          cloneHeader(req.Trailer),
		URL:              cloneURL(req.URL),
	}
}

// Clone returns a copy of a Builder for creating a new Builder with properties
// from a parent Builder. For example,
//
// 	parentSling := sling.Clone().Client(client).Base("https://api.io/")
// 	fooSling := parentSling.Clone().Get("foo/")
// 	barSling := parentSling.Clone().Get("bar/")
//
// fooSling and barSling will both use the same client, but send requests to
// https://api.io/foo/ and https://api.io/bar/ respectively.
//
// Note that query and body values are copied so if pointer values are used,
// mutating the original value will mutate the value within the child Builder.
func (s *Builder) Clone() *Builder {
	s2 := *s
	s2.Header = cloneHeader(s.Header)
	s2.Trailer = cloneHeader(s.Trailer)
	s2.URL = cloneURL(s.URL)
	s2.QueryParams = cloneValues(s.QueryParams)
	return &s2
}

// Requests

// Request returns a new http.Request created with the Builder properties.
// Returns any errors parsing the base, encoding query structs, encoding
// the body, or creating the http.Request.
func (s *Builder) Request(ctx context.Context) (*http.Request, error) {
	// marshal body, if applicable
	bodyData, ct, err := s.getRequestBody()
	if err != nil {
		return nil, err
	}

	urlS := ""
	if s.URL != nil {
		urlS = s.URL.String()
	}

	req, err := http.NewRequest(s.Method, urlS, bodyData)
	if err != nil {
		return nil, err
	}

	// if we marshaled the body, use our content type
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}

	if s.ContentLength != 0 {
		req.ContentLength = s.ContentLength
	}

	if s.GetBody != nil {
		req.GetBody = s.GetBody
	}

	// copy the host
	if s.Host != "" {
		req.Host = s.Host
	}

	req.TransferEncoding = s.TransferEncoding
	req.Close = s.Close
	req.Trailer = s.Trailer

	// copy Headers pairs into new Header map
	for k, v := range s.Header {
		req.Header[k] = v
	}

	if len(s.QueryParams) > 0 {
		if req.URL.RawQuery != "" {
			req.URL.RawQuery += "&" + s.QueryParams.Encode()
		} else {
			req.URL.RawQuery = s.QueryParams.Encode()
		}
	}

	return req.WithContext(ctx), err
}

// getRequestBody returns the io.Reader which should be used as the body
// of new Requests.
func (s *Builder) getRequestBody() (body io.Reader, contentType string, err error) {
	switch v := s.Body.(type) {
	case nil:
		return nil, "", nil
	case io.Reader:
		return v, "", nil
	case string:
		return strings.NewReader(v), "", nil
	case []byte:
		return bytes.NewReader(v), "", nil
	default:
		marshaler := s.Marshaler
		if marshaler == nil {
			marshaler = DefaultMarshaler
		}
		b, ct, err := marshaler.Marshal(s.Body)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(b), ct, err
	}
}

// Sending

func (s *Builder) Do(ctx context.Context) (*http.Response, error) {
	req, err := s.Request(ctx)
	if err != nil {
		return nil, err
	}
	doer := s.Doer
	if doer == nil {
		doer = http.DefaultClient
	}
	return doer.Do(req)
}

// ReceiveSuccess creates a new HTTP request and returns the response. Success
// responses (2XX) are JSON decoded into the value pointed to by successV.
// Any error creating the request, sending it, or decoding a 2XX response
// is returned.
func (s *Builder) ReceiveSuccess(ctx context.Context, successV interface{}) (resp *http.Response, body []byte, err error) {
	return s.Receive(ctx, successV, nil)
}

// Receive creates a new HTTP request and returns the response. Success
// responses (2XX) are JSON decoded into the value pointed to by successV and
// other responses are JSON decoded into the value pointed to by failureV.
// Any error creating the request, sending it, or decoding the response is
// returned.
// Receive is shorthand for calling Request and Do.
func (s *Builder) Receive(ctx context.Context, successV, failureV interface{}) (resp *http.Response, body []byte, err error) {
	resp, err = s.Do(ctx)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var unmarshalInto interface{}
	if code := resp.StatusCode; 200 <= code && code <= 299 {
		unmarshalInto = successV
	} else {
		unmarshalInto = failureV
	}

	if unmarshalInto != nil {
		unmarshaler := s.Unmarshaler
		if unmarshaler == nil {
			unmarshaler = DefaultUnmarshaler
		}

		err = unmarshaler.Unmarshal(body, resp.Header.Get("Content-Type"), unmarshalInto)
	}
	return
}
