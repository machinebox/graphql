package graphql

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithClient(t *testing.T) {
	var calls int

	testClient := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			resp := &http.Response{
				Body: ioutil.NopCloser(strings.NewReader(`{"data":{"key":"value"}}`)),
			}
			return resp, nil
		}),
	}

	ctx := context.Background()
	client := NewClient("", WithHTTPClient(testClient), UseMultipartForm())

	req := NewRequest(`mutation test()`)
	err := client.Run(ctx, req, nil)
	require.NoError(t, err)

	require.Equal(t, calls, 1) // calls
}

func TestDoUseMultipartForm(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		operations := r.FormValue("operations")
		maps := r.FormValue("map")
		require.Equal(t, operations, `{"operationName":"test","variables":null,"query":"mutation test()"}`)
		require.Equal(t, maps, "")
		_, err := io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
		require.NoError(t, err)
	}))

	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	req := NewRequest(`mutation test()`)
	err := client.Run(ctx, req, &responseData)
	require.NoError(t, err)
	require.Equal(t, calls, 1) // calls
	require.Equal(t, responseData["something"], "yes")
}

func TestImmediatelyCloseReqBody(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		operations := r.FormValue("operations")
		require.Equal(t, operations, `{"operationName":"test","variables":null,"query":"mutation test()"}`)
		_, err := io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, ImmediatelyCloseReqBody(), UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	req := NewRequest(`mutation test()`)
	err := client.Run(ctx, req, &responseData)
	require.NoError(t, err)
	require.Equal(t, calls, 1) // calls
	require.Equal(t, responseData["something"], "yes")
}

func TestDoErr(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		operations := r.FormValue("operations")
		require.Equal(t, operations, `{"operationName":"test","variables":null,"query":"mutation test()"}`)
		_, err := io.WriteString(w, `{
			"errors": [{
				"message": "Something went wrong"
			}]
		}`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	req := NewRequest(`mutation test()`)
	err := client.Run(ctx, req, &responseData)
	require.Error(t, err)
	require.Equal(t, err.Error(), "graphql: Something went wrong")
}

func TestDoServerErr(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		operations := r.FormValue("operations")
		require.Equal(t, operations, `{"operationName":"test","variables":null,"query":"mutation test()"}`)
		w.WriteHeader(http.StatusInternalServerError)

		_, err := io.WriteString(w, `Internal Server Error`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	req := NewRequest(`mutation test()`)
	err := client.Run(ctx, req, &responseData)
	require.Equal(t, err.Error(), "graphql: server returned a non-200 status code: 500")
}

func TestDoBadRequestErr(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		operations := r.FormValue("operations")
		require.Equal(t, operations, `{"operationName":"test","variables":null,"query":"mutation test()"}`)
		w.WriteHeader(http.StatusBadRequest)
		_, err := io.WriteString(w, `{
			"errors": [{
				"message": "miscellaneous message as to why the the request was bad"
			}]
		}`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var responseData map[string]interface{}

	req := NewRequest(`mutation test()`)
	err := client.Run(ctx, req, &responseData)
	require.Equal(t, err.Error(), "graphql: miscellaneous message as to why the the request was bad")
}

func TestDoNoResponse(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, r.Method, http.MethodPost)
		operations := r.FormValue("operations")
		require.Equal(t, operations, `{"operationName":"test","variables":null,"query":"mutation test()"}`)
		_, err := io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	req := NewRequest(`mutation test()`)
	err := client.Run(ctx, req, nil)
	require.NoError(t, err)
	require.Equal(t, calls, 1) // calls
}

func TestQuery(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		operations := r.FormValue("operations")
		require.Equal(t, operations, `{"operationName":"test","variables":{"username":"matryer"},"query":"mutation test()"}`)
		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL, UseMultipartForm())

	req := NewRequest("mutation test()")
	req.Var("username", "matryer")

	// check variables
	require.True(t, req != nil)
	require.Equal(t, req.vars["username"], "matryer")

	var resp struct {
		Value string
	}

	err := client.Run(ctx, req, &resp)
	require.NoError(t, err)
	require.Equal(t, calls, 1)

	require.Equal(t, resp.Value, "some data")
}

func TestFile(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		file, header, err := r.FormFile("1")
		require.NoError(t, err)

		require.Equal(t, header.Filename, "filename.txt")
		b, err := ioutil.ReadAll(file)
		require.NoError(t, err)
		require.Equal(t, string(b), `This is a file`)

		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		require.NoError(t, err)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL, UseMultipartForm())
	req := NewRequest(`mutation test($input: testInput!) {test(input: $input) {}}`)
	f := strings.NewReader(`This is a file`)
	req.File("variables.input.file", "filename.txt", f)

	err := client.Run(ctx, req, nil)
	require.NoError(t, err)
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
