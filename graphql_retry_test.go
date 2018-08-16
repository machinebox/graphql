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

func getTestDuration(sec int) time.Duration {
	return time.Duration(sec)*time.Second + 1*time.Second
}

func TestLinearPolicy(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, WithDefaultLinearRetryConfig())
	client.Log = func(str string) {
		t.Log(str)
	}

	ctx, cancel := context.WithTimeout(ctx, getTestDuration(10))
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if !strings.HasPrefix(err.Error(), "Error getting response with retry:") {
		is.Fail()
	}
}

func TestNoPolicySpecified(t *testing.T) {
	t.Parallel()
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
	client := NewClient(srv.URL)
	client.Log = func(str string) {
		t.Log(str)
	}

	ctx, cancel := context.WithTimeout(ctx, getTestDuration(1))
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	is.NoErr(err)
	is.Equal(responseData["something"], "yes")
}

func TestCustomRetryStatus(t *testing.T) {
	t.Parallel()
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
		MaxTries:    1,
		Interval:    1,
		RetryStatus: retryStatus,
	}
	client := NewClient(srv.URL, WithRetryConfig(retryConfig))
	client.Log = func(str string) {
		t.Log(str)
	}

	ctx, cancel := context.WithTimeout(ctx, getTestDuration(1))
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if !strings.HasPrefix(err.Error(), "Error getting response with retry:") {
		is.Fail()
	}
}

func TestExponentialBackoffPolicy(t *testing.T) {
	t.Parallel()
	is := is.New(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, WithDefaultExponentialRetryConfig())
	client.retryConfig.BeforeRetry = func(req *http.Request, resp *http.Response, attemptCount int) {
		t.Logf("Retrying request: %+v", req)
		t.Logf("Retrying after last response: %+v", resp)
		t.Logf("Retrying attempt count: %d", attemptCount)
	}
	client.Log = func(str string) {
		t.Log(str)
	}

	ctx, cancel := context.WithTimeout(ctx, getTestDuration(31))
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	if !strings.HasPrefix(err.Error(), "Error getting response with retry:") {
		is.Fail()
	}
}
