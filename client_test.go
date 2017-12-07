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

	"github.com/matryer/is"
)

func TestDo(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		query := r.FormValue("query")
		is.Equal(query, `query {}`)
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()
	c := &client{
		endpoint: srv.URL,
		httpclient: &http.Client{
			Timeout: 1 * time.Second,
		},
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := c.Do(ctx, &Request{q: "query {}"}, &responseData)
	is.NoErr(err)
	is.Equal(calls, 1) // calls
	is.Equal(responseData["something"], "yes")
}

func TestDoErr(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		query := r.FormValue("query")
		is.Equal(query, `query {}`)
		io.WriteString(w, `{
			"errors": [{
				"message": "Something went wrong"
			}]
		}`)
	}))
	defer srv.Close()
	c := &client{
		endpoint: srv.URL,
		httpclient: &http.Client{
			Timeout: 1 * time.Second,
		},
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := c.Do(ctx, &Request{q: "query {}"}, &responseData)
	is.True(err != nil)
	is.Equal(err.Error(), "graphql: Something went wrong")
}

func TestDoNoResponse(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		query := r.FormValue("query")
		is.Equal(query, `query {}`)
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()
	c := &client{
		endpoint: srv.URL,
		httpclient: &http.Client{
			Timeout: 1 * time.Second,
		},
	}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err := c.Do(ctx, &Request{q: "query {}"}, nil)
	is.NoErr(err)
	is.Equal(calls, 1) // calls
}

func TestQuery(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		query := r.FormValue("query")
		is.Equal(query, "query {}")
		is.Equal(r.FormValue("variables"), `{"username":"matryer"}`+"\n")
		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		is.NoErr(err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	ctx = NewContext(ctx, srv.URL)

	req := NewRequest("query {}")
	req.Var("username", "matryer")

	// check variables
	is.True(req != nil)
	is.Equal(req.vars["username"], "matryer")

	var resp struct {
		Value string
	}
	err := req.Run(ctx, &resp)
	is.NoErr(err)
	is.Equal(calls, 1)

	is.Equal(resp.Value, "some data")

}

func TestFile(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		file, header, err := r.FormFile("file")
		is.NoErr(err)
		defer file.Close()
		is.Equal(header.Filename, "filename.txt")
		is.Equal(header.Size, int64(14))

		b, err := ioutil.ReadAll(file)
		is.NoErr(err)
		is.Equal(string(b), `This is a file`)

		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		is.NoErr(err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	ctx = NewContext(ctx, srv.URL)

	f := strings.NewReader(`This is a file`)

	req := NewRequest("query {}")
	req.File("filename.txt", f)

	err := req.Run(ctx, nil)
	is.NoErr(err)

}
