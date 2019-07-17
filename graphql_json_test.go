package graphql

import (
	"context"
	"github.com/mathew-bowersox/jflect"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"log"

	"github.com/matryer/is"
)
// the code in this file is derived from the machinebox graphql project code and subject to licensing terms in included APACHE_LICENSE

type SimpleResponse struct {
	Data struct {
		Something string `json:"something"`
	} `json:"data"`
}

func TestDoJSON(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		is.NoErr(err)
		is.Equal(string(b), `{"query":"query {}","variables":null}`+"\n")
		io.WriteString(w, `{
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
	is.NoErr(err)
	is.Equal(calls, 1) // calls
	is.Equal(responseData["something"], "yes")
}

func TestSimpleJsonStructGeneration(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		is.NoErr(err)
		is.Equal(string(b), `{"query":"query {}","variables":null}`+"\n")
		io.WriteString(w, `{
			"data": {
				"something": "yes"
			}
		}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)
	client.Log = func(s string) { log.Println(s) }
	strNme := "Results"
    client.ProcessResult = func (r io.Reader) error {
    	err := generate.Generate(r, os.Stdout, &strNme)
    	return err
    }
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	responseData := SimpleResponse{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	is.NoErr(err)
}



func TestDoJSONServerError(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		is.NoErr(err)
		is.Equal(string(b), `{"query":"query {}","variables":null}`+"\n")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `Internal Server Error`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	is.Equal(calls, 1) // calls
	is.Equal(err.Error(), "graphql: server returned a non-200 status code: 500")
}

func TestDoJSONBadRequestErr(t *testing.T) {
	is := is.New(t)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Method, http.MethodPost)
		b, err := ioutil.ReadAll(r.Body)
		is.NoErr(err)
		is.Equal(string(b), `{"query":"query {}","variables":null}`+"\n")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{
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
	is.Equal(calls, 1) // calls
	is.Equal(err.Error(), "graphql: miscellaneous message as to why the the request was bad")
}

func TestQueryJSON(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, err := ioutil.ReadAll(r.Body)
		is.NoErr(err)
		is.Equal(string(b), `{"query":"query {}","variables":{"username":"matryer"}}`+"\n")
		_, err = io.WriteString(w, `{"data":{"value":"some data"}}`)
		is.NoErr(err)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := NewClient(srv.URL)

	req := NewRequest("query {}")
	req.Var("username", "matryer")

	// check variables
	is.True(req != nil)
	is.Equal(req.vars["username"], "matryer")

	var resp struct {
		Value string
	}
	err := client.Run(ctx, req, &resp)
	is.NoErr(err)
	is.Equal(calls, 1)

	is.Equal(resp.Value, "some data")
}

func TestHeader(t *testing.T) {
	is := is.New(t)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		is.Equal(r.Header.Get("X-Custom-Header"), "123")

		_, err := io.WriteString(w, `{"data":{"value":"some data"}}`)
		is.NoErr(err)
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
	is.NoErr(err)
	is.Equal(calls, 1)

	is.Equal(resp.Value, "some data")
}
