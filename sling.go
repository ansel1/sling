package sling

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	goquery "github.com/google/go-querystring/query"
	"context"
	"errors"
	"io/ioutil"
)

const (
	contentType     = "Content-Type"
	jsonContentType = "application/json"
	xmlContentType  = "application/xml"
	formContentType = "application/x-www-form-urlencoded"
)

// Doer executes http requests.  It is implemented by *http.Client.  You can
// wrap *http.Client with layers of Doers to form a stack of client-side
// middleware.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Marshaler interface {
	Marshal(v interface{}) (data []byte, contentType string, err error)
}

type Unmarshaler interface {
	Unmarshal(data []byte, contentType string, v interface{}) error
}

type MarshalFunc func(v interface{}) ([]byte, string, error)

func (f MarshalFunc) Marshal(v interface{}) ([]byte, string, error) {
	return f(v)
}

type UnmarshalFunc func(data []byte, contentType string, v interface{}) error

func (f UnmarshalFunc) Unmarshal(data []byte, contentType string, v interface{}) error {
	return f(data, contentType, v)
}

type GetBodyFunc func() (io.ReadCloser, error)


// Sling is an HTTP Request builder and sender.
type Sling struct {
	// http Client for doing requests
	httpClient Doer
	Template http.Request
	// Additional query params appended to request
	QueryParams url.Values
	Marshaler   Marshaler
	Unmarshaler Unmarshaler

	bodyValue interface{}

	Error error
}

// New returns a new Sling with an http DefaultClient.
func New() *Sling {
	return &Sling{}
}

// Clone returns a copy of a Sling for creating a new Sling with properties
// from a parent Sling. For example,
//
// 	parentSling := sling.Clone().Client(client).Base("https://api.io/")
// 	fooSling := parentSling.Clone().Get("foo/")
// 	barSling := parentSling.Clone().Get("bar/")
//
// fooSling and barSling will both use the same client, but send requests to
// https://api.io/foo/ and https://api.io/bar/ respectively.
//
// Note that query and body values are copied so if pointer values are used,
// mutating the original value will mutate the value within the child Sling.
func (s *Sling) Clone() *Sling {

	s2 := *s
	s2.Template = s.cloneRequest()
	if s.QueryParams != nil {
		s2.QueryParams = url.Values{}
		for k, v := range s.QueryParams {
			s2.QueryParams[k] = v
		}
	}
	return &s2
}

func (s *Sling) cloneRequest() http.Request {
	req := s.Template
	if s.Template.Header != nil {
		// copy Headers pairs into new Header map
		headerCopy := make(http.Header)
		for k, v := range s.Template.Header {
			headerCopy[k] = v
		}
		req.Header = headerCopy
	}
	if s.Template.URL != nil {
		u2 := *s.Template.URL
		req.URL = &u2
	}
	return req
}

// Doer sets the custom Doer implementation used to do requests.
// If a nil client is given, the http.DefaultClient will be used.
func (s *Sling) Doer(doer Doer) *Sling {
	if doer == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = doer
	}
	return s
}

// Method

// Head sets the Sling method to HEAD and sets the given pathURL.
func (s *Sling) Head(pathURL string) *Sling {
	s.Template.Method = "HEAD"
	return s.Path(pathURL)
}

// Get sets the Sling method to GET and sets the given pathURL.
func (s *Sling) Get(pathURL string) *Sling {
	s.Template.Method = "GET"
	return s.Path(pathURL)
}

// Post sets the Sling method to POST and sets the given pathURL.
func (s *Sling) Post(pathURL string) *Sling {
	s.Template.Method = "POST"
	return s.Path(pathURL)
}

// Put sets the Sling method to PUT and sets the given pathURL.
func (s *Sling) Put(pathURL string) *Sling {
	s.Template.Method = "PUT"
	return s.Path(pathURL)
}

// Patch sets the Sling method to PATCH and sets the given pathURL.
func (s *Sling) Patch(pathURL string) *Sling {
	s.Template.Method = "PATCH"
	return s.Path(pathURL)
}

// Delete sets the Sling method to DELETE and sets the given pathURL.
func (s *Sling) Delete(pathURL string) *Sling {
	s.Template.Method = "DELETE"
	return s.Path(pathURL)
}

// Header

// Add adds the key, value pair in Headers, appending values for existing keys
// to the key's values. Header keys are canonicalized.
func (s *Sling) Add(key, value string) *Sling {
	if s.Template.Header == nil {
		s.Template.Header = http.Header{}
	}
	s.Template.Header.Add(key, value)
	return s
}

// Set sets the key, value pair in Headers, replacing existing values
// associated with key. Header keys are canonicalized.
func (s *Sling) Set(key, value string) *Sling {
	if s.Template.Header == nil {
		s.Template.Header = http.Header{}
	}
	s.Template.Header.Set(key, value)
	return s
}

// SetBasicAuth sets the Authorization header to use HTTP Basic Authentication
// with the provided username and password. With HTTP Basic Authentication
// the provided username and password are not encrypted.
func (s *Sling) SetBasicAuth(username, password string) *Sling {
	return s.Set("Authorization", "Basic "+basicAuth(username, password))
}

// basicAuth returns the base64 encoded username:password for basic auth copied
// from net/http.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// Url

// Base sets the rawURL. If you intend to extend the url with Path,
// baseUrl should be specified with a trailing slash.
func (s *Sling) Base(rawURL string) *Sling {
	u, err := url.Parse(rawURL)
	if err != nil {
		s.Error = err
		return s
	}
	s.Template.URL = u
	return s
}

// Path extends the rawURL with the given path by resolving the reference to
// an absolute URL. If parsing errors occur, the rawURL is left unmodified.
func (s *Sling) Path(path string) *Sling {
	pathURL, err := url.Parse(path)
	if err != nil {
		s.Error = err
		return s
	}
	if s.Template.URL == nil {
		s.Template.URL = pathURL
	} else {
		s.Template.URL = s.Template.URL.ResolveReference(pathURL)
	}
	return s
}

// QueryStruct appends the queryStruct to the Sling's queryStructs. The value
// pointed to by each queryStruct will be encoded as url query parameters on
// new requests (see Request()).
// The queryStruct argument should be a pointer to a url tagged struct. See
// https://godoc.org/github.com/google/go-querystring/query for details.
func (s *Sling) AddQueryParams(queryStruct interface{}) *Sling {
	if s.QueryParams == nil {
		s.QueryParams = url.Values{}
	}
	var values url.Values
	switch t := queryStruct.(type) {
	case nil:
	case url.Values:
		values = t
	default:
		// encodes query structs into a url.Values map and merges maps
		var err error
		values, err = goquery.Values(queryStruct)
		if err != nil {
			s.Error = err
			return s
		}
	}

	// merges new values into existing
	for key, values := range values {
		for _, value := range values {
			s.QueryParams.Add(key, value)
		}
	}
	return s
}

// Requests

// Request returns a new http.Request created with the Sling properties.
// Returns any errors parsing the rawURL, encoding query structs, encoding
// the body, or creating the http.Request.
func (s *Sling) Request(ctx context.Context) (*http.Request, error) {
	if s.Error != nil {
		return nil, s.Error
	}

	// marshal body, if applicable
	bodyData, ct, err := s.getRequestBody()
	if err != nil {
		return nil, err
	}
	if bodyData != nil {

	}
	http.NewRequest(s.Template.Method, "", bodyData)

	req := s.cloneRequest()
	if req.URL == nil {
		return nil, errors.New("request URL cannot be nil")
	}
	if req.Method == "" {
		req.Method = "GET"
	}
	if len(s.QueryParams) > 0 {
		if req.URL.RawQuery != "" {
			req.URL.RawQuery += "&" + s.QueryParams.Encode()
		} else {
			req.URL.RawQuery = s.QueryParams.Encode()
		}
	}


	// marshal body, if applicable
	bodyData, ct, err := s.getRequestBody()
	if err != nil {
		return nil, err
	}
	if bodyData != nil {
		req.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(bodyData)), nil
		}
		req.Body, _ = req.GetBody()
		req.ContentLength = (int64)(len(bodyData))
		if ct != "" && req.Header.Get(contentType) == "" {
			if req.Header == nil {
				req.Header = http.Header{}
			}
			req.Header.Set(contentType, ct)
		}
	}
	return &req, err
}

// addQueryStructs parses url tagged query structs using go-querystring to
// encode them to url.Values and format them onto the url.RawQuery. Any
// query parsing or encoding errors are returned.
func addQueryStructs(reqURL *url.URL, queryStructs []interface{}) error {
	urlValues, err := url.ParseQuery(reqURL.RawQuery)
	if err != nil {
		return err
	}
	// encodes query structs into a url.Values map and merges maps
	for _, queryStruct := range queryStructs {
		queryValues, err := goquery.Values(queryStruct)
		if err != nil {
			return err
		}
		for key, values := range queryValues {
			for _, value := range values {
				urlValues.Add(key, value)
			}
		}
	}
	// url.Values format to a sorted "url encoded" string, e.g. "key=val&foo=bar"
	reqURL.RawQuery = urlValues.Encode()
	return nil
}

var DefaultMarshaler Marshaler = MarshalFunc(MarshalJSON)
var DefaultUnmarshaler Unmarshaler = UnmarshalFunc(UnmarshalMulti)

// getRequestBody returns the io.Reader which should be used as the body
// of new Requests.
func (s *Sling) getRequestBody() (data []byte, contentType string, err error) {
	if s.bodyValue == nil {
		return nil, "", nil
	}
	marshaler := s.Marshaler
	if marshaler == nil {
		marshaler = DefaultMarshaler
	}

	return marshaler.Marshal(s.BodyValue)
}

// Sending

// ReceiveSuccess creates a new HTTP request and returns the response. Success
// responses (2XX) are JSON decoded into the value pointed to by successV.
// Any error creating the request, sending it, or decoding a 2XX response
// is returned.
func (s *Sling) ReceiveSuccess(successV interface{}) (*http.Response, error) {
	return s.Receive(successV, nil)
}

// Receive creates a new HTTP request and returns the response. Success
// responses (2XX) are JSON decoded into the value pointed to by successV and
// other responses are JSON decoded into the value pointed to by failureV.
// Any error creating the request, sending it, or decoding the response is
// returned.
// Receive is shorthand for calling Request and Do.
func (s *Sling) Receive(successV, failureV interface{}) (*http.Response, error) {
	req, err := s.Request()
	if err != nil {
		return nil, err
	}
	return s.Do(req, successV, failureV)
}

// Do sends an HTTP request and returns the response. Success responses (2XX)
// are JSON decoded into the value pointed to by successV and other responses
// are JSON decoded into the value pointed to by failureV.
// Any error sending the request or decoding the response is returned.
func (s *Sling) Do(req *http.Request, successV, failureV interface{}) (*http.Response, error) {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return resp, err
	}
	// when err is nil, resp contains a non-nil resp.Body which must be closed
	defer resp.Body.Close()

	// Don't try to decode on 204s
	if resp.StatusCode == 204 {
		return resp, nil
	}

	if successV == nil && failureV == nil {
		return resp, nil
	}

	if strings.Contains(resp.Header.Get(contentType), jsonContentType) {
		err = decodeResponseJSON(resp, successV, failureV)
	}
	return resp, err
}

// decodeResponse decodes response Body into the value pointed to by successV
// if the response is a success (2XX) or into the value pointed to by failureV
// otherwise. If the successV or failureV argument to decode into is nil,
// decoding is skipped.
// Caller is responsible for closing the resp.Body.
func decodeResponseJSON(resp *http.Response, successV, failureV interface{}) error {
	if code := resp.StatusCode; 200 <= code && code <= 299 {
		if successV != nil {
			return decodeResponseBodyJSON(resp, successV)
		}
	} else {
		if failureV != nil {
			return decodeResponseBodyJSON(resp, failureV)
		}
	}
	return nil
}

// decodeResponseBodyJSON JSON decodes a Response Body into the value pointed
// to by v.
// Caller must provide a non-nil v and close the resp.Body.
func decodeResponseBodyJSON(resp *http.Response, v interface{}) error {
	return json.NewDecoder(resp.Body).Decode(v)
}
