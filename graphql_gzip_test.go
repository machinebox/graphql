package graphql

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDoJSONGzipServerError(t *testing.T) {

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Header.Get("Content-Encoding"), "gzip")

		calls++
		assert.Equal(t, r.Method, http.MethodPost)

		body, err := io.ReadAll(r.Body)
		assert.Nil(t, err)

		compressedData := bytes.NewReader(body)

		reader, err := gzip.NewReader(compressedData)
		assert.Nil(t, err)

		decodedData, err := io.ReadAll(reader)
		assert.Nil(t, err)

		b := decodedData

		assert.Equal(t, string(b), `{"query":"query {}","variables":null}`+"\n")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `Internal Server Error`)
	}))
	defer srv.Close()

	ctx := context.Background()
	client := NewClient(srv.URL, UseGzip())

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	var responseData map[string]interface{}
	err := client.Run(ctx, &Request{q: "query {}"}, &responseData)
	assert.Equal(t, calls, 1) // calls
	assert.Equal(t, err.Error(), "graphql: server returned a non-200 status code: 500")
}
