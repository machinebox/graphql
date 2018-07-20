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

func TestLinearPolicy(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx := context.Background()
	retryConfig := RetryConfig{
		Policy:      Linear,
		MaxTries:    3,
		Interval:    1,
		MaxInterval: 10,
	}
	client := NewClient(srv.URL, WithRetryConfig(retryConfig))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if !strings.HasPrefix(err.Error(), "Error getting response with retry:") {
		is.Fail()
	}
}

func TestNoPolicySpecified(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)

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
	retryConfig := RetryConfig{}
	client := NewClient(srv.URL, WithRetryConfig(retryConfig))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	is.NoErr(err)
	is.Equal(responseData["something"], "yes")
}

func TestCustomRetryStatus(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	retryStatus := make(map[int]bool)
	retryStatus[http.StatusOK] = true
	retryConfig := RetryConfig{
		Policy:      Linear,
		MaxTries:    3,
		Interval:    1,
		MaxInterval: 10,
		RetryStatus: retryStatus,
	}
	client := NewClient(srv.URL, WithRetryConfig(retryConfig))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	t.Logf("error: %s", err)
	if !strings.HasPrefix(err.Error(), "Error getting response with retry:") {
		is.Fail()
	}
}

func TestExponentialBackoffPolicy(t *testing.T) {
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx := context.Background()
	retryConfig := RetryConfig{
		Policy:      ExponentialBackoff,
		MaxTries:    3,
		Interval:    1,
		MaxInterval: 10,
	}
	client := NewClient(srv.URL, WithRetryConfig(retryConfig))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	t.Logf("error: %s", err)
	if !strings.HasPrefix(err.Error(), "Error getting response with retry:") {
		is.Fail()
	}
}
