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

func TestURLString(t *testing.T) {
	cases := []string{"http://a.io/", "http://b.io", "/relPath", "relPath", ""}
	for _, base := range cases {
		t.Run("", func(t *testing.T) {
			b, errFromNew := New(URLString(base))
			u, err := url.Parse(base)
			if err == nil {
				require.Equal(t, u, b.URL)
			} else {
				require.EqualError(t, errFromNew, err.Error())
			}
		})
	}

	t.Run("errors", func(t *testing.T) {
		b, err := New(URLString("cache_object:foo/bar"))
		require.Error(t, err)
		require.Nil(t, b)
	})
}

func TestRelativeURLString(t *testing.T) {
	cases := []struct {
		base     string
		relPath  string
		expected string
	}{
		{"http://a.io/", "foo", "http://a.io/foo"},
		{"http://a.io/", "/foo", "http://a.io/foo"},
		{"http://a.io", "foo", "http://a.io/foo"},
		{"http://a.io", "/foo", "http://a.io/foo"},
		{"http://a.io/foo/", "bar", "http://a.io/foo/bar"},
		// base should end in trailing slash if it is to be URLString extended
		{"http://a.io/foo", "bar", "http://a.io/bar"},
		{"http://a.io/foo", "/bar", "http://a.io/bar"},
		// relPath extension is absolute
		{"http://a.io", "http://b.io/", "http://b.io/"},
		{"http://a.io/", "http://b.io/", "http://b.io/"},
		{"http://a.io", "http://b.io", "http://b.io"},
		{"http://a.io/", "http://b.io", "http://b.io"},
		// empty base, empty relPath
		{"", "http://b.io", "http://b.io"},
		{"http://a.io", "", "http://a.io"},
		{"", "", ""},
		{"/red", "", "/red"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New()
			require.NoError(t, err)
			if c.base != "" {
				err := b.Apply(URLString(c.base))
				require.NoError(t, err)
			}
			err = b.Apply(RelativeURLString(c.relPath))
			require.NoError(t, err)
			require.Equal(t, c.expected, b.URL.String())
		})
	}

	t.Run("errors", func(t *testing.T) {
		b, err := New(URLString("http://test.com/red"))
		require.NoError(t, err)
		err = b.Apply(RelativeURLString("cache_object:foo/bar"))
		require.Error(t, err)
		require.Equal(t, "http://test.com/red", b.URL.String())
	})
}

func TestURL(t *testing.T) {
	cases := []string{
		"http://test.com",
		"",
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			var u *url.URL
			if c != "" {
				var err error
				u, err = url.Parse(c)
				require.NoError(t, err)
			}
			b, err := New(URL(u))
			require.NoError(t, err)
			require.Equal(t, u, b.URL)
		})
	}
}

func TestRelativeURL(t *testing.T) {
	cases := []struct {
		base        string
		relPath     string
		expectedURL string
	}{
		{"http://a.io/", "foo", "http://a.io/foo"},
		{"http://a.io/", "/foo", "http://a.io/foo"},
		{"http://a.io", "foo", "http://a.io/foo"},
		{"http://a.io", "/foo", "http://a.io/foo"},
		{"http://a.io/foo/", "bar", "http://a.io/foo/bar"},
		// base should end in trailing slash if it is to be URLString extended
		{"http://a.io/foo", "bar", "http://a.io/bar"},
		{"http://a.io/foo", "/bar", "http://a.io/bar"},
		// relPath extension is absolute
		{"http://a.io", "http://b.io/", "http://b.io/"},
		{"http://a.io/", "http://b.io/", "http://b.io/"},
		{"http://a.io", "http://b.io", "http://b.io"},
		{"http://a.io/", "http://b.io", "http://b.io"},
		// empty base, empty relPath
		{"", "http://b.io", "http://b.io"},
		{"http://a.io", "", "http://a.io"},
		{"", "", ""},
		{"/red", "", "/red"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			var u *url.URL
			if c.relPath != "" {
				var err error
				u, err = url.Parse(c.relPath)
				require.NoError(t, err)
			}
			b, err := New()
			require.NoError(t, err)
			if c.base != "" {
				err := b.Apply(URLString(c.base))
				require.NoError(t, err)
			}
			err = b.Apply(RelativeURL(u))
			require.NoError(t, err)
			if c.expectedURL == "" {
				require.Nil(t, b.URL)
			} else {
				require.Equal(t, c.expectedURL, b.URL.String())
			}
		})
	}

}

func TestMethod(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedMethod string
	}{
		{[]Option{Method("red")}, "red"},
		{[]Option{Head()}, "HEAD"},
		{[]Option{Get()}, "GET"},
		{[]Option{Post()}, "POST"},
		{[]Option{Put()}, "PUT"},
		{[]Option{Patch()}, "PATCH"},
		{[]Option{Delete()}, "DELETE"},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedMethod, b.Method)
		})
	}
}

func TestHeader(t *testing.T) {
	cases := []http.Header{
		{"red": []string{"green"}},
		nil,
	}
	for _, c := range cases {
		b, err := New(Header(c))
		require.NoError(t, err)
		require.Equal(t, c, b.Header)
	}
}

func TestAddHeader(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedHeader http.Header
	}{
		{[]Option{AddHeader("authorization", "OAuth key=\"value\"")}, http.Header{"Authorization": {"OAuth key=\"value\""}}},
		// header keys should be canonicalized
		{[]Option{AddHeader("content-tYPE", "application/json"), AddHeader("User-AGENT", "sling")}, http.Header{"Content-Type": {"application/json"}, "User-Agent": {"sling"}}},
		// values for existing keys should be appended
		{[]Option{AddHeader("A", "B"), AddHeader("a", "c")}, http.Header{"A": {"B", "c"}}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedHeader, b.Header)
		})
	}
}

func TestSetHeader(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedHeader http.Header
	}{
		// should replace existing values associated with key
		{[]Option{AddHeader("A", "B"), SetHeader("a", "c")}, http.Header{"A": []string{"c"}}},
		{[]Option{SetHeader("content-type", "A"), SetHeader("Content-Type", "B")}, http.Header{"Content-Type": []string{"B"}}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			// type conversion from Header to alias'd map for deep equality comparison
			require.Equal(t, c.expectedHeader, b.Header)
		})
	}
}

func TestBasicAuth(t *testing.T) {
	cases := []struct {
		options      []Option
		expectedAuth []string
	}{
		// basic auth: username & password
		{[]Option{BasicAuth("Aladdin", "open sesame")}, []string{"Aladdin", "open sesame"}},
		// empty username
		{[]Option{BasicAuth("", "secret")}, []string{"", "secret"}},
		// empty password
		{[]Option{BasicAuth("admin", "")}, []string{"admin", ""}},
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			req, err := b.Request(context.Background())
			require.NoError(t, err)
			username, password, ok := req.BasicAuth()
			require.True(t, ok, "basic auth missing when expected")
			auth := []string{username, password}
			require.Equal(t, c.expectedAuth, auth)
		})
	}
}

func TestBearerAuth(t *testing.T) {
	cases := []string{
		"red",
		"",
	}
	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(BearerAuth(c))
			require.NoError(t, err)
			if c == "" {
				require.Empty(t, b.Header.Get("Authorization"))
			} else {
				require.Equal(t, "Bearer "+c, b.Header.Get("Authorization"))
			}
		})
	}

	t.Run("clearing", func(t *testing.T) {
		b, err := New(BearerAuth("green"))
		require.NoError(t, err)
		err = b.Apply(BearerAuth(""))
		require.NoError(t, err)
		_, ok := b.Header["Authorization"]
		require.False(t, ok, "should have removed Authorization header, instead was %s", b.Header.Get("Authorization"))
	})
}

func TestQueryParams(t *testing.T) {
	cases := []struct {
		options        []Option
		expectedParams url.Values
	}{
		{nil, nil},
		{[]Option{QueryParams(nil)}, url.Values{}},
		{[]Option{QueryParams(paramsA)}, url.Values{"limit": []string{"30"}}},
		{[]Option{QueryParams(paramsA), QueryParams(paramsA)}, url.Values{"limit": []string{"30", "30"}}},
		{[]Option{QueryParams(paramsA), QueryParams(paramsB)}, url.Values{"limit": []string{"30"}, "kind_name": []string{"recent"}, "count": []string{"25"}}},
		{[]Option{QueryParams(paramsA, paramsB)}, url.Values{"limit": []string{"30"}, "kind_name": []string{"recent"}, "count": []string{"25"}}},
		{[]Option{QueryParams(url.Values{"red": []string{"green"}})}, url.Values{"red": []string{"green"}}},
		{[]Option{QueryParams(map[string][]string{"red": []string{"green"}})}, url.Values{"red": []string{"green"}}},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			b, err := New(c.options...)
			require.NoError(t, err)
			require.Equal(t, c.expectedParams, b.QueryParams)
		})
	}
}

func TestBody(t *testing.T) {
	b, err := New(Body("hey"))
	require.NoError(t, err)
	require.Equal(t, "hey", b.Body)
}

type testMarshaler struct{}

func (*testMarshaler) Unmarshal(data []byte, contentType string, v interface{}) error {
	panic("implement me")
}

func (*testMarshaler) Marshal(v interface{}) (data []byte, contentType string, err error) {
	panic("implement me")
}

func TestWithMarshaler(t *testing.T) {
	m := &testMarshaler{}
	b, err := New(WithMarshaler(m))
	require.NoError(t, err)
	require.Equal(t, m, b.Marshaler)
}

func TestWithUnmarshaler(t *testing.T) {
	m := &testMarshaler{}
	b, err := New(WithUnmarshaler(m))
	require.NoError(t, err)
	require.Equal(t, m, b.Unmarshaler)
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

func TestBuilder_Request_urlAndMethod(t *testing.T) {
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

func TestBuilder_Request_queryStructs(t *testing.T) {
	cases := []struct {
		options []Option
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

func TestBuilder_Request_body(t *testing.T) {
	cases := []struct {
		options []Option
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

func TestBuilder_Request_contentLength(t *testing.T) {
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

func TestBuilder_Request_getBody(t *testing.T) {
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
	b.Trailer = http.Header{"color":[]string{"red"}}
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
	b.Header = http.Header{"color":[]string{"red"}}
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

func TestBuilder_Request_JSONMarshaler(t *testing.T) {
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
}

//// Sending
//
//type APIError struct {
//	Message string `json:"message"`
//	Code    int    `json:"code"`
//}
//
//func TestDo_onSuccess(t *testing.T) {
//	const expectedText = "Some text"
//	const expectedFavoriteCount int64 = 24
//
//	client, mux, server := testServer()
//	defer server.Close()
//	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
//		w.Header().Set("Content-Type", "application/json")
//		fmt.Fprintf(w, `{"text": "Some text", "favorite_count": 24}`)
//	})
//
//	sling := New().Client(client)
//	req, _ := http.NewRequest("GET", "http://example.com/success", nil)
//
//	model := new(FakeModel)
//	apiError := new(APIError)
//	resp, err := sling.Do(req, model, apiError)
//
//	if err != nil {
//		t.Errorf("expected nil, got %v", err)
//	}
//	if resp.StatusCode != 200 {
//		t.Errorf("expected %d, got %d", 200, resp.StatusCode)
//	}
//	if model.Text != expectedText {
//		t.Errorf("expected %s, got %s", expectedText, model.Text)
//	}
//	if model.FavoriteCount != expectedFavoriteCount {
//		t.Errorf("expected %d, got %d", expectedFavoriteCount, model.FavoriteCount)
//	}
//}
//
//func TestDo_onSuccessWithNilValue(t *testing.T) {
//	client, mux, server := testServer()
//	defer server.Close()
//	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
//		w.Header().Set("Content-Type", "application/json")
//		fmt.Fprintf(w, `{"text": "Some text", "favorite_count": 24}`)
//	})
//
//	sling := New().Client(client)
//	req, _ := http.NewRequest("GET", "http://example.com/success", nil)
//
//	apiError := new(APIError)
//	resp, err := sling.Do(req, nil, apiError)
//
//	if err != nil {
//		t.Errorf("expected nil, got %v", err)
//	}
//	if resp.StatusCode != 200 {
//		t.Errorf("expected %d, got %d", 200, resp.StatusCode)
//	}
//	expected := &APIError{}
//	if !reflect.DeepEqual(expected, apiError) {
//		t.Errorf("failureV should not be populated, exepcted %v, got %v", expected, apiError)
//	}
//}
//
//func TestDo_onFailure(t *testing.T) {
//	const expectedMessage = "Invalid argument"
//	const expectedCode int = 215
//
//	client, mux, server := testServer()
//	defer server.Close()
//	mux.HandleFunc("/failure", func(w http.ResponseWriter, r *http.Request) {
//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(400)
//		fmt.Fprintf(w, `{"message": "Invalid argument", "code": 215}`)
//	})
//
//	sling := New().Client(client)
//	req, _ := http.NewRequest("GET", "http://example.com/failure", nil)
//
//	model := new(FakeModel)
//	apiError := new(APIError)
//	resp, err := sling.Do(req, model, apiError)
//
//	if err != nil {
//		t.Errorf("expected nil, got %v", err)
//	}
//	if resp.StatusCode != 400 {
//		t.Errorf("expected %d, got %d", 400, resp.StatusCode)
//	}
//	if apiError.Message != expectedMessage {
//		t.Errorf("expected %s, got %s", expectedMessage, apiError.Message)
//	}
//	if apiError.Code != expectedCode {
//		t.Errorf("expected %d, got %d", expectedCode, apiError.Code)
//	}
//}
//
//func TestDo_onFailureWithNilValue(t *testing.T) {
//	client, mux, server := testServer()
//	defer server.Close()
//	mux.HandleFunc("/failure", func(w http.ResponseWriter, r *http.Request) {
//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(420)
//		fmt.Fprintf(w, `{"message": "Enhance your calm", "code": 88}`)
//	})
//
//	sling := New().Client(client)
//	req, _ := http.NewRequest("GET", "http://example.com/failure", nil)
//
//	model := new(FakeModel)
//	resp, err := sling.Do(req, model, nil)
//
//	if err != nil {
//		t.Errorf("expected nil, got %v", err)
//	}
//	if resp.StatusCode != 420 {
//		t.Errorf("expected %d, got %d", 420, resp.StatusCode)
//	}
//	expected := &FakeModel{}
//	if !reflect.DeepEqual(expected, model) {
//		t.Errorf("successV should not be populated, exepcted %v, got %v", expected, model)
//	}
//}
//
//func TestDo_skipDecodingIfContentTypeWrong(t *testing.T) {
//	client, mux, server := testServer()
//	defer server.Close()
//	mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
//		w.Header().Set("Content-Type", "text/html")
//		fmt.Fprintf(w, `{"text": "Some text", "favorite_count": 24}`)
//	})
//
//	sling := New().Client(client)
//	req, _ := http.NewRequest("GET", "http://example.com/success", nil)
//
//	model := new(FakeModel)
//	sling.Do(req, model, nil)
//
//	expectedModel := &FakeModel{}
//	if !reflect.DeepEqual(expectedModel, model) {
//		t.Errorf("decoding should have been skipped, Content-Type was incorrect")
//	}
//}
//
//func TestReceive_success(t *testing.T) {
//	client, mux, server := testServer()
//	defer server.Close()
//	mux.HandleFunc("/foo/submit", func(w http.ResponseWriter, r *http.Request) {
//		assertMethod(t, "POST", r)
//		assertQuery(t, map[string]string{"kind_name": "vanilla", "count": "11"}, r)
//		assertPostForm(t, map[string]string{"kind_name": "vanilla", "count": "11"}, r)
//		w.Header().Set("Content-Type", "application/json")
//		fmt.Fprintf(w, `{"text": "Some text", "favorite_count": 24}`)
//	})
//
//	endpoint := New().Client(client).Base("http://example.com/").Path("foo/").Post("submit")
//	// encode url-tagged struct in query params and as post body for testing purposes
//	params := FakeParams{KindName: "vanilla", Count: 11}
//	model := new(FakeModel)
//	apiError := new(APIError)
//	resp, err := endpoint.New().QueryStruct(params).BodyForm(params).Receive(model, apiError)
//
//	if err != nil {
//		t.Errorf("expected nil, got %v", err)
//	}
//	if resp.StatusCode != 200 {
//		t.Errorf("expected %d, got %d", 200, resp.StatusCode)
//	}
//	expectedModel := &FakeModel{Text: "Some text", FavoriteCount: 24}
//	if !reflect.DeepEqual(expectedModel, model) {
//		t.Errorf("expected %v, got %v", expectedModel, model)
//	}
//	expectedAPIError := &APIError{}
//	if !reflect.DeepEqual(expectedAPIError, apiError) {
//		t.Errorf("failureV should be zero valued, exepcted %v, got %v", expectedAPIError, apiError)
//	}
//}
//
//func TestReceive_failure(t *testing.T) {
//	client, mux, server := testServer()
//	defer server.Close()
//	mux.HandleFunc("/foo/submit", func(w http.ResponseWriter, r *http.Request) {
//		assertMethod(t, "POST", r)
//		assertQuery(t, map[string]string{"kind_name": "vanilla", "count": "11"}, r)
//		assertPostForm(t, map[string]string{"kind_name": "vanilla", "count": "11"}, r)
//		w.Header().Set("Content-Type", "application/json")
//		w.WriteHeader(429)
//		fmt.Fprintf(w, `{"message": "Rate limit exceeded", "code": 88}`)
//	})
//
//	endpoint := New().Client(client).Base("http://example.com/").Path("foo/").Post("submit")
//	// encode url-tagged struct in query params and as post body for testing purposes
//	params := FakeParams{KindName: "vanilla", Count: 11}
//	model := new(FakeModel)
//	apiError := new(APIError)
//	resp, err := endpoint.New().QueryStruct(params).BodyForm(params).Receive(model, apiError)
//
//	if err != nil {
//		t.Errorf("expected nil, got %v", err)
//	}
//	if resp.StatusCode != 429 {
//		t.Errorf("expected %d, got %d", 429, resp.StatusCode)
//	}
//	expectedAPIError := &APIError{Message: "Rate limit exceeded", Code: 88}
//	if !reflect.DeepEqual(expectedAPIError, apiError) {
//		t.Errorf("expected %v, got %v", expectedAPIError, apiError)
//	}
//	expectedModel := &FakeModel{}
//	if !reflect.DeepEqual(expectedModel, model) {
//		t.Errorf("successV should not be zero valued, expected %v, got %v", expectedModel, model)
//	}
//}
//
//func TestReceive_errorCreatingRequest(t *testing.T) {
//	expectedErr := errors.New("json: unsupported value: +Inf")
//	resp, err := New().BodyJSON(FakeModel{Temperature: math.Inf(1)}).Receive(nil, nil)
//	if err == nil || err.Error() != expectedErr.Error() {
//		t.Errorf("expected %v, got %v", expectedErr, err)
//	}
//	if resp != nil {
//		t.Errorf("expected nil resp, got %v", resp)
//	}
//}
//
//// Testing Utils
//
//// testServer returns an http Client, ServeMux, and Server. The client proxies
//// requests to the server and handlers can be registered on the mux to handle
//// requests. The caller must close the test server.
//func testServer() (*http.Client, *http.ServeMux, *httptest.Server) {
//	mux := http.NewServeMux()
//	server := httptest.NewServer(mux)
//	transport := &http.Transport{
//		Proxy: func(req *http.Request) (*url.URL, error) {
//			return url.Parse(server.URL)
//		},
//	}
//	client := &http.Client{Transport: transport}
//	return client, mux, server
//}
//
//func assertMethod(t *testing.T, expectedMethod string, req *http.Request) {
//	if actualMethod := req.Method; actualMethod != expectedMethod {
//		t.Errorf("expected method %s, got %s", expectedMethod, actualMethod)
//	}
//}
//
//// assertQuery tests that the Request has the expected url query key/val pairs
//func assertQuery(t *testing.T, expected map[string]string, req *http.Request) {
//	queryValues := req.URL.Query() // net/url Values is a map[string][]string
//	expectedValues := url.Values{}
//	for key, value := range expected {
//		expectedValues.Add(key, value)
//	}
//	if !reflect.DeepEqual(expectedValues, queryValues) {
//		t.Errorf("expected parameters %v, got %v", expected, req.URL.RawQuery)
//	}
//}
//
//// assertPostForm tests that the Request has the expected key values pairs url
//// encoded in its Body
//func assertPostForm(t *testing.T, expected map[string]string, req *http.Request) {
//	req.ParseForm() // parses request Body to put url.Values in r.Form/r.PostForm
//	expectedValues := url.Values{}
//	for key, value := range expected {
//		expectedValues.Add(key, value)
//	}
//	if !reflect.DeepEqual(expectedValues, req.PostForm) {
//		t.Errorf("expected parameters %v, got %v", expected, req.PostForm)
//	}
//}
