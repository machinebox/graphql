package graphql

import (
	"context"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

	req := NewRequest(``)
	client.Run(ctx, req, nil)

	assert.Equal(t, calls, 1) // calls
}

func TestDoUseMultipartForm(t *testing.T) {
	
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		query := r.FormValue("query")
		assert.Equal(t, query, `query {}`)
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.NoError(t, err)
	assert.Equal(t, calls, 1) // calls
	assert.Equal(t, responseData["something"], "yes")
}
func TestImmediatelyCloseReqBody(t *testing.T) {
	
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		query := r.FormValue("query")
		assert.Equal(t, query, `query {}`)
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, ImmediatelyCloseReqBody(), UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.NoError(t, err)
	assert.Equal(t, calls, 1) // calls
	assert.Equal(t, responseData["something"], "yes")
}

func TestDoErr(t *testing.T) {
	
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		query := r.FormValue("query")
		assert.Equal(t, query, `query {}`)
		io.WriteString(w, `{
			"errors": [{
				"message": "Something went wrong"
			}]
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "graphql: Something went wrong")
}

func TestDoServerErr(t *testing.T) {
	
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		query := r.FormValue("query")
		assert.Equal(t, query, `query {}`)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `Internal Server Error`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.Equal(t, err.Error(), "graphql: server returned a non-200 status code: 500")
}

func TestDoBadRequestErr(t *testing.T) {
	
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		query := r.FormValue("query")
		assert.Equal(t, query, `query {}`)
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{
			"errors": [{
				"message": "miscellaneous message as to why the the request was bad"
			}]
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.Equal(t, err.Error(), "graphql: miscellaneous message as to why the the request was bad")
}

func TestDoNoResponse(t *testing.T) {
	
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, r.Method, http.MethodPost)
		query := r.FormValue("query")
		assert.Equal(t, query, `query {}`)
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseMultipartForm())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err := client.Run(ctx, &Request{q: "query {}"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, calls, 1) // calls
}

func TestQuery(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		query := r.FormValue("query")
		assert.Equal(t, "query {}", query)
		assert.Equal(t, `{"username":"matryer"}`+"\n", r.FormValue("variables"))
		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		assert.NoError(t, err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL, UseMultipartForm())

	req := NewRequest("query {}")
	req.Var("username", "matryer")

	// check variables
	assert.NotNil(t, req)
	assert.Equal(t, req.vars["username"], "matryer")

	var resp struct {
		Value string
	}
	err := client.Run(ctx, req, &resp)
	assert.NoError(t, err)
	assert.Equal(t, calls, 1)

	assert.Equal(t, resp.Value, "some data")

}

func TestFile(t *testing.T) {
	

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		file, header, err := r.FormFile("file")
		assert.NoError(t, err)
		defer file.Close()
		assert.Equal(t, header.Filename, "filename.txt")

		b, err := ioutil.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, string(b), `This is a file`)

		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		assert.NoError(t, err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	client := NewClient(srv.URL, UseMultipartForm())
	f := strings.NewReader(`This is a file`)
	req := NewRequest("query {}")
	req.File("file", "filename.txt", f)
	err := client.Run(ctx, req, nil)
	assert.NoError(t, err)
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
