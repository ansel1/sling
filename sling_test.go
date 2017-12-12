package sling

import (
	"net/url"
	"testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"net/http"
	"context"
	"strings"
	"bytes"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"errors"
	"fmt"
)

type FakeParams struct {
	KindName string `url:"kind_name"`
	Count    int    `url:"count"`
}

// Url-tagged query struct
var paramsA = struct {
	Limit int `url:"limit"`
}{
	30,
}
var paramsB = FakeParams{KindName: "recent", Count: 25}

// Json-tagged model struct
type FakeModel struct {
	Text          string  `json:"text,omitempty"`
	FavoriteCount int64   `json:"favorite_count,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
}

var modelA = FakeModel{Text: "note", FavoriteCount: 12}

func TestNew(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	require.NotNil(t, b)
}

func TestBuilder_Clone(t *testing.T) {
	cases := [][]Option{
		{Get(), URLString("http: //example.com")},
		{URLString("http://example.com")},
		{QueryParams(url.Values{})},
		{QueryParams(paramsA)},
		{QueryParams(paramsA, paramsB)},
		{Body(&FakeModel{Text: "a"})},
		{Body(FakeModel{Text: "a"})},
		{AddHeader("Content-Type", "application/json")},
		{AddHeader("A", "B"), AddHeader("a", "c")},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c...)
			require.NoError(t, err)

			child := b.Clone()
			require.Equal(t, b.Doer, child.Doer)
			require.Equal(t, b.Method, child.Method)
			require.Equal(t, b.URL, child.URL)
			// Header should be a copy of parent Builder header. For example, calling
			// baseSling.AddHeader("k","v") should not mutate previously created child Slings
			assert.EqualValues(t, b.Header, child.Header)
			if b.Header != nil {
				// struct literal cases don't init Header in usual way, skip header check
				assert.EqualValues(t, b.Header, child.Header)
				b.Header.Add("K", "V")
				assert.Empty(t, child.Header.Get("K"), "child.header was a reference to original map, should be copy")
			} else {
				assert.Nil(t, child.Header)
			}
			// queryStruct slice should be a new slice with a copy of the contents
			assert.EqualValues(t, b.QueryParams, child.QueryParams)
			if len(b.QueryParams) > 0 {
				// mutating one slice should not mutate the other
				child.QueryParams.Set("color", "red")
				assert.Empty(t, b.QueryParams.Get("color"), "child.QueryParams should be a copy")
			}
			// bodyJSON should be copied
			assert.Equal(t, b.Body, child.Body)
		})
	}
}

func TestBuilder_With(t *testing.T) {
	b, err := New(Method("red"))
	require.NoError(t, err)
	b2, err := b.With(Method("green"))
	require.NoError(t, err)
	// should clone first, then apply
	require.Equal(t, "green", b2.Method)
	require.Equal(t, "red", b.Method)

	t.Run("errors", func(t *testing.T) {
		b, err := New(Method("green"))
		require.NoError(t, err)
		b2, err := b.With(Method("red"), RelativeURLString("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, b2)
		require.Equal(t, "green", b.Method)
	})
}

func TestBuilder_Apply(t *testing.T) {
	b, err := New(Method("red"))
	require.NoError(t, err)
	err = b.Apply(Method("green"))
	require.NoError(t, err)
	// applies in place
	require.Equal(t, "green", b.Method)

	t.Run("errors", func(t *testing.T) {
		err := b.Apply(URLString("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, b.URL)
	})
}

func TestBuilder_Request_URLAndMethod(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedMethod string
		expectedURL    string
	}{
		{[]Option{URLString("http://a.io")}, "GET", "http://a.io"},
		{[]Option{RelativeURLString("http://a.io")}, "GET", "http://a.io"},
		{[]Option{Get("http://a.io")}, "GET", "http://a.io"},
		{[]Option{Put("http://a.io")}, "PUT", "http://a.io"},
		{[]Option{URLString("http://a.io/"), RelativeURLString("foo")}, "GET", "http://a.io/foo"},
		{[]Option{URLString("http://a.io/"), Post("foo")}, "POST", "http://a.io/foo"},
		// if relative relPath is an absolute url, base is ignored
		{[]Option{URLString("http://a.io"), RelativeURLString("http://b.io")}, "GET", "http://b.io"},
		{[]Option{RelativeURLString("http://a.io"), RelativeURLString("http://b.io")}, "GET", "http://b.io"},
		// last method setter takes priority
		{[]Option{Get("http://b.io"), Post("http://a.io")}, "POST", "http://a.io"},
		{[]Option{Post("http://a.io/"), Put("foo/"), Delete("bar")}, "DELETE", "http://a.io/foo/bar"},
		// last Base setter takes priority
		{[]Option{URLString("http://a.io"), URLString("http://b.io")}, "GET", "http://b.io"},
		// URLString setters are additive
		{[]Option{URLString("http://a.io/"), RelativeURLString("foo/"), RelativeURLString("bar")}, "GET", "http://a.io/foo/bar"},
		{[]Option{RelativeURLString("http://a.io/"), RelativeURLString("foo/"), RelativeURLString("bar")}, "GET", "http://a.io/foo/bar"},
		// removes extra '/' between base and ref url
		{[]Option{URLString("http://a.io/"), Get("/foo")}, "GET", "http://a.io/foo"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			req, err := b.Request(context.Background())
			require.NoError(t, err)
			assert.Equal(t, c.expectedURL, req.URL.String())
			assert.Equal(t, c.expectedMethod, req.Method)
		})
	}

	t.Run("invalidmethod", func(t *testing.T) {
		b, err := New(Method("@"))
		require.NoError(t, err)
		req, err := b.Request(context.Background())
		require.Error(t, err)
		require.Nil(t, req)
	})

}

func TestBuilder_Request_QueryStructs(t *testing.T) {
	cases := []struct {
		options     []Option
		expectedURL string
	}{
		{[]Option{URLString("http://a.io"), QueryParams(paramsA)}, "http://a.io?limit=30"},
		{[]Option{URLString("http://a.io/?color=red"), QueryParams(paramsA)}, "http://a.io/?color=red&limit=30"},
		{[]Option{URLString("http://a.io"), QueryParams(paramsA), QueryParams(paramsB)}, "http://a.io?count=25&kind_name=recent&limit=30"},
		{[]Option{URLString("http://a.io/"), RelativeURLString("foo?relPath=yes"), QueryParams(paramsA)}, "http://a.io/foo?relPath=yes&limit=30"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			req, _ := b.Request(context.Background())
			require.Equal(t, c.expectedURL, req.URL.String())
		})
	}
}

func TestBuilder_Request_Body(t *testing.T) {
	cases := []struct {
		options             []Option
		expectedBody        string // expected Body io.Reader as a string
		expectedContentType string
	}{
		// Body (json)
		{[]Option{Body(modelA)}, `{"text":"note","favorite_count":12}`, jsonContentType},
		{[]Option{Body(&modelA)}, `{"text":"note","favorite_count":12}`, jsonContentType},
		{[]Option{Body(&FakeModel{})}, `{}`, jsonContentType},
		{[]Option{Body(FakeModel{})}, `{}`, jsonContentType},
		// BodyForm
		//{[]Option{Body(paramsA)}, "limit=30", formContentType},
		//{[]Option{Body(paramsB)}, "count=25&kind_name=recent", formContentType},
		//{[]Option{Body(&paramsB)}, "count=25&kind_name=recent", formContentType},
		// Raw bodies, skips marshaler
		{[]Option{Body(strings.NewReader("this-is-a-test"))}, "this-is-a-test", ""},
		{[]Option{Body("this-is-a-test")}, "this-is-a-test", ""},
		{[]Option{Body([]byte("this-is-a-test"))}, "this-is-a-test", ""},
		// no body
		{nil, "", ""},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			req, err := b.Request(context.Background())
			require.NoError(t, err)
			if b.Body != nil {
				buf := new(bytes.Buffer)
				buf.ReadFrom(req.Body)
				// req.Body should have contained the expectedBody string
				assert.Equal(t, c.expectedBody, buf.String())
				// Header Content-Type should be expectedContentType ("" means no contentType expected)

			} else {
				assert.Nil(t, req.Body)
			}
			assert.Equal(t, c.expectedContentType, req.Header.Get(contentType))
		})
	}
}

func TestBuilder_Request_Marshaler(t *testing.T) {
	var capturedV interface{}
	b := Builder{
		Body: []string{"blue"},
		Marshaler: MarshalFunc(func(v interface{}) ([]byte, string, error) {
			capturedV = v
			return []byte("red"), "orange", nil
		}),
	}

	req, err := b.Request(context.Background())
	require.NoError(t, err)

	require.Equal(t, []string{"blue"}, capturedV)
	by, err := ioutil.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, "red", string(by))
	require.Equal(t, "orange", req.Header.Get("Content-Type"))

	t.Run("errors", func(t *testing.T) {
		b.Marshaler = MarshalFunc(func(v interface{}) ([]byte, string, error) {
			return nil, "", errors.New("boom")
		})
		_, err := b.Request(context.Background())
		require.Error(t, err, "boom")
	})
}

func TestBuilder_Request_ContentLength(t *testing.T) {
	b, err := New(Body("1234"))
	require.NoError(t, err)
	req, err := b.Request(context.Background())
	require.NoError(t, err)
	// content length should be set automatically
	require.EqualValues(t, 4, req.ContentLength)

	// I should be able to override it
	b.ContentLength = 10
	req, err = b.Request(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 10, req.ContentLength)
}

func TestBuilder_Request_GetBody(t *testing.T) {
	b, err := New(Body("1234"))
	require.NoError(t, err)
	req, err := b.Request(context.Background())
	require.NoError(t, err)
	// GetBody should be populated automatically
	rdr, err := req.GetBody()
	require.NoError(t, err)
	bts, err := ioutil.ReadAll(rdr)
	require.NoError(t, err)
	require.Equal(t, "1234", string(bts))

	// I should be able to override it
	b.GetBody = func() (io.ReadCloser, error) {
		return ioutil.NopCloser(strings.NewReader("5678")), nil
	}
	req, err = b.Request(context.Background())
	require.NoError(t, err)
	rdr, err = req.GetBody()
	require.NoError(t, err)
	bts, err = ioutil.ReadAll(rdr)
	require.NoError(t, err)
	require.Equal(t, "5678", string(bts))
}

func TestBuilder_Request_Host(t *testing.T) {
	b, err := New(URLString("http://test.com/red"))
	require.NoError(t, err)
	req, err := b.Request(context.Background())
	require.NoError(t, err)
	// Host should be set automatically
	require.Equal(t, "test.com", req.Host)

	// but I can override it
	b.Host = "test2.com"
	req, err = b.Request(context.Background())
	require.NoError(t, err)
	require.Equal(t, "test2.com", req.Host)
}

func TestBuilder_Request_TransferEncoding(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	req, err := b.Request(context.Background())
	require.NoError(t, err)
	// should be empty by default
	require.Nil(t, req.TransferEncoding)

	// but I can set it
	b.TransferEncoding = []string{"red"}
	req, err = b.Request(context.Background())
	require.NoError(t, err)
	require.Equal(t, b.TransferEncoding, req.TransferEncoding)
}

func TestBuilder_Request_Close(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	req, err := b.Request(context.Background())
	require.NoError(t, err)
	// should be false by default
	require.False(t, req.Close)

	// but I can set it
	b.Close = true
	req, err = b.Request(context.Background())
	require.NoError(t, err)
	require.True(t, req.Close)
}

func TestBuilder_Request_Trailer(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	req, err := b.Request(context.Background())
	require.NoError(t, err)
	// should be empty by default
	require.Nil(t, req.Trailer)

	// but I can set it
	b.Trailer = http.Header{"color": []string{"red"}}
	req, err = b.Request(context.Background())
	require.NoError(t, err)
	require.Equal(t, b.Trailer, req.Trailer)
}

func TestBuilder_Request_Header(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	req, err := b.Request(context.Background())
	require.NoError(t, err)
	// should be empty by default
	require.Empty(t, req.Header)

	// but I can set it
	b.Header = http.Header{"color": []string{"red"}}
	req, err = b.Request(context.Background())
	require.NoError(t, err)
	require.Equal(t, b.Header, req.Header)
}

func TestBuilder_Request_Context(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	req, err := b.Request(context.WithValue(context.Background(), "color", "red"))
	require.NoError(t, err)
	require.Equal(t, "red", req.Context().Value("color"))
}

func TestBuilder_Do(t *testing.T) {
	cl, mux, srv := testServer()
	defer srv.Close()

	b, err := New(
		URLString("http://blue.com/server"),
		AddHeader("color", "red"),
	)
	require.NoError(t, err)
	b.Doer = cl

	var req *http.Request

	// Do() just creates a request and sends it to the Doer.  That's all we're confirming here
	mux.Handle("/server", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req = r
		w.WriteHeader(204)
	}))

	resp, err := b.Do(context.Background())
	require.NoError(t, err)

	// confirm the request went through
	require.NotNil(t, req)
	assert.Equal(t, "red", req.Header.Get("color"))
	assert.Equal(t, 204, resp.StatusCode)
}

func TestBuilder_Receive(t *testing.T) {
	cl, mux, srv := testServer()
	defer srv.Close()

	succBuilder, err := New(
		URLString("http://blue.com/model.json"),
	)
	require.NoError(t, err)
	succBuilder.Doer = cl

	failBuilder, err := succBuilder.With(RelativeURLString("/err"))
	require.NoError(t, err)

	mux.HandleFunc("/model.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(206)
		w.Write([]byte(`{"color":"red","count":30}`))
	})

	mux.HandleFunc("/err", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{"color":"red","count":30}`))
	})

	cases := []struct{
		succ, fail bool
	} {
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("succ=%v,fail=%v", c.succ, c.fail), func(t *testing.T) {

			var succ, fail testModel
			doCall := func(b *Builder) (*http.Response, []byte, error){
				switch {
				case c.succ && c.fail:
					return b.Receive(context.Background(), &succ, &fail)
				case !c.succ && !c.fail:
					return b.Receive(context.Background(), nil, nil)
				case c.succ:
					return b.Receive(context.Background(), &succ, nil)
				default:
					return b.Receive(context.Background(), nil, &fail)
				}
			}

			resp, body, err := doCall(succBuilder)
			require.NoError(t, err)
			assert.Equal(t, 206, resp.StatusCode)
			assert.Equal(t, `{"color":"red","count":30}`, string(body))
			if c.succ {
				assert.Equal(t, testModel{"red", 30}, succ)
			}

			resp, body, err = doCall(failBuilder)
			assert.Equal(t, 500, resp.StatusCode)
			assert.Equal(t, `{"color":"red","count":30}`, string(body))
			if c.fail {
				assert.Equal(t, testModel{"red", 30}, fail)
			}
		})
	}
}

func TestBuilder_ReceiveSuccess(t *testing.T) {
	cl, mux, srv := testServer()
	defer srv.Close()

	b, err := New(
		URLString("http://blue.com/model.json"),
	)
	require.NoError(t, err)
	b.Doer = cl

	mux.HandleFunc("/model.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(206)
		w.Write([]byte(`{"color":"red","count":30}`))
	})

	// this is just a special case of Receive(v, nil)
	var m testModel
	resp, body, err := b.ReceiveSuccess(context.Background(), &m)
	require.NoError(t, err)
	assert.Equal(t, 206, resp.StatusCode)
	assert.Equal(t, `{"color":"red","count":30}`, string(body))
	assert.Equal(t, testModel{"red", 30}, m)

}

// Testing Utils

// testServer returns an http Client, ServeMux, and Server. The client proxies
// requests to the server and handlers can be registered on the mux to handle
// requests. The caller must close the test server.
func testServer() (*http.Client, *http.ServeMux, *httptest.Server) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}
	client := &http.Client{Transport: transport}
	return client, mux, server
}
