package sling

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestRequest(t *testing.T) {
	req, err := Request(Get("http://blue.com/red"))
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Equal(t, "http://blue.com/red", req.URL.String())
}

func TestRequestContext(t *testing.T) {
	req, err := RequestContext(
		context.WithValue(context.Background(), "color", "green"),
		Get("http://blue.com/red"),
	)
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Equal(t, "http://blue.com/red", req.URL.String())
	assert.Equal(t, "green", req.Context().Value("color"))
}

func TestDo(t *testing.T) {
	client, mux, server := testServer()
	defer server.Close()

	mux.HandleFunc("/red", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	resp, err := Do(Get("http://blue.com/red"), WithDoer(client))
	require.NoError(t, err)

	assert.Equal(t, 204, resp.StatusCode)
}

func TestDoContext(t *testing.T) {
	client, mux, server := testServer()
	defer server.Close()

	mux.HandleFunc("/red", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	var ctx context.Context
	resp, err := DoContext(
		context.WithValue(context.Background(), "color", "blue"),
		Get("http://blue.com/red"),
		WithDoer(client),
		Use(captureRequestContextMiddleware(&ctx)),
	)

	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode)
	assert.Equal(t, "blue", ctx.Value("color"))
}

func TestReceive(t *testing.T) {
	client, mux, server := testServer()
	defer server.Close()

	mux.HandleFunc("/red", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(205)
		w.Write([]byte(`{"count":25}`))

	})

	var m testModel
	resp, body, err := Receive(&m, Get("http://blue.com/red"), WithDoer(client))
	require.NoError(t, err)

	assert.Equal(t, `{"count":25}`, body)
	assert.Equal(t, 205, resp.StatusCode)
	assert.Equal(t, 25, m.Count)

	t.Run("Context", func(t *testing.T) {
		var m testModel
		var ctx context.Context

		resp, body, err := ReceiveContext(
			context.WithValue(context.Background(), "color", "yellow"),
			&m,
			Get("http://blue.com/red"),
			WithDoer(client),
			Use(captureRequestContextMiddleware(&ctx)),
		)
		require.NoError(t, err)

		assert.Equal(t, `{"count":25}`, body)
		assert.Equal(t, 205, resp.StatusCode)
		assert.Equal(t, 25, m.Count)
		assert.Equal(t, "yellow", ctx.Value("color"))
	})

	t.Run("Full", func(t *testing.T) {
		mux.HandleFunc("/err", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(HeaderContentType, ContentTypeJSON)
			w.WriteHeader(500)
			w.Write([]byte(`{"color":"red"}`))
		})

		var m testModel

		resp, body, err := ReceiveFull(
			&m, nil,
			Get("http://blue.com/red"),
			WithDoer(client),
		)
		require.NoError(t, err)

		assert.Equal(t, `{"count":25}`, body)
		assert.Equal(t, 205, resp.StatusCode)
		assert.Equal(t, 25, m.Count)

		m = testModel{}
		resp, body, err = ReceiveFull(
			nil, &m,
			Get("http://blue.com/err"),
			WithDoer(client),
		)
		require.NoError(t, err)

		assert.Equal(t, `{"color":"red"}`, body)
		assert.Equal(t, 500, resp.StatusCode)
		assert.Equal(t, "red", m.Color)

		t.Run("Context", func(t *testing.T) {
			var m testModel
			var ctx context.Context

			resp, body, err := ReceiveFullContext(
				context.WithValue(context.Background(), "color", "yellow"),
				&m, nil,
				Get("http://blue.com/red"),
				WithDoer(client),
				Use(captureRequestContextMiddleware(&ctx)),
			)
			require.NoError(t, err)

			assert.Equal(t, `{"count":25}`, body)
			assert.Equal(t, 205, resp.StatusCode)
			assert.Equal(t, 25, m.Count)

			m = testModel{}
			resp, body, err = ReceiveFullContext(
				context.WithValue(context.Background(), "color", "yellow"),
				nil, &m,
				Get("http://blue.com/err"),
				WithDoer(client),
				Use(captureRequestContextMiddleware(&ctx)),
			)
			require.NoError(t, err)

			assert.Equal(t, `{"color":"red"}`, body)
			assert.Equal(t, 500, resp.StatusCode)
			assert.Equal(t, "red", m.Color)
		})
	})
}
