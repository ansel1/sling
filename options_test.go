package sling

import (
	"testing"
	"net/url"
	"github.com/stretchr/testify/require"
	"net/http"
	"context"
	"github.com/stretchr/testify/assert"
)

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

func TestJSON(t *testing.T) {
	b, err := New(JSON(false))
	require.NoError(t, err)
	if assert.IsType(t, &JSONMarshaler{}, b.Marshaler) {
		assert.False(t, b.Marshaler.(*JSONMarshaler).Indent)
	}

	err = b.Apply(JSON(true))
	require.NoError(t, err)
	if assert.IsType(t, &JSONMarshaler{}, b.Marshaler) {
		assert.True(t, b.Marshaler.(*JSONMarshaler).Indent)
	}
}

func TestXML(t *testing.T) {
	b, err := New(XML(false))
	require.NoError(t, err)
	if assert.IsType(t, &XMLMarshaler{}, b.Marshaler) {
		assert.False(t, b.Marshaler.(*XMLMarshaler).Indent)
	}

	err = b.Apply(XML(true))
	require.NoError(t, err)
	if assert.IsType(t, &XMLMarshaler{}, b.Marshaler) {
		assert.True(t, b.Marshaler.(*XMLMarshaler).Indent)
	}
}

func TestForm(t *testing.T) {
	b, err := New(Form())
	require.NoError(t, err)
	assert.IsType(t, &FormMarshaler{}, b.Marshaler)
}