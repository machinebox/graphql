package graphql

import (
	"context"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoJSON(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, string(b), `{"query":"query {}","variables":null}`+"\n")
		_, _ = io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.NoError(t, err)
	assert.Equal(t, calls, 1) // calls
	assert.Equal(t, responseData["something"], "yes")
}

func TestDoJSONServerError(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, string(b), `{"query":"query {}","variables":null}`+"\n")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `Internal Server Error`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.Equal(t, calls, 1) // calls
	assert.Equal(t, err.Error(), "graphql: server returned a non-200 status code: 500")
}

func TestDoJSONBadRequestErr(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, http.MethodPost, r.Method)
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, `{"query":"query {}","variables":null}`+"\n", string(b))
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{
			"errors": [{
				"message": "miscellaneous message as to why the the request was bad"
			}]
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.Equal(t, calls, 1) // calls
	assert.Equal(t, "graphql: miscellaneous message as to why the the request was bad", err.Error())
}

func TestQueryJSON(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, `{"query":"query {}","variables":{"username":"matryer"}}`+"\n", string(b))
		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		assert.NoError(t, err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL)

	req := NewRequest("query {}")
	req.Var("username", "matryer")

	// check variables
	assert.NotNil(t, req)
	assert.Equal(t, "matryer", req.vars["username"])

	var resp struct {
		Value string
	}
	err := client.Run(ctx, req, &resp)
	assert.NoError(t, err)
	assert.Equal(t, calls, 1)

	assert.Equal(t, "some data", resp.Value)
}

func TestHeader(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "123", r.Header.Get("X-Custom-Header"))

		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		assert.NoError(t, err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL)

	req := NewRequest("query {}")
	req.Header.Set("X-Custom-Header", "123")

	var resp struct {
		Value string
	}
	err := client.Run(ctx, req, &resp)
	assert.NoError(t, err)
	assert.Equal(t, calls, 1)

	assert.Equal(t, "some data", resp.Value)
}
