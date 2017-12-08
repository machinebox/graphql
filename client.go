package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/net/context/ctxhttp"
)

// Client accesses a GraphQL API.
type client struct {
	endpoint string
}

// Do executes a query request and returns the response.
func (c *client) Do(ctx context.Context, request *Request, response interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	if err := writer.WriteField("query", request.q); err != nil {
		return errors.Wrap(err, "write query field")
	}
	if len(request.vars) > 0 {
		variablesField, err := writer.CreateFormField("variables")
		if err != nil {
			return errors.Wrap(err, "create variables field")
		}
		if err := json.NewEncoder(variablesField).Encode(request.vars); err != nil {
			return errors.Wrap(err, "encode variables")
		}
	}
	for i := range request.files {
		filename := fmt.Sprintf("file-%d", i+1)
		if i == 0 {
			// just use "file" for the first one
			filename = "file"
		}
		part, err := writer.CreateFormFile(filename, request.files[i].Name)
		if err != nil {
			return errors.Wrap(err, "create form file")
		}
		if _, err := io.Copy(part, request.files[i].R); err != nil {
			return errors.Wrap(err, "preparing file")
		}
	}
	if err := writer.Close(); err != nil {
		return errors.Wrap(err, "close writer")
	}
	var graphResponse = struct {
		Data   interface{}
		Errors []graphErr
	}{
		Data: response,
	}
	req, err := http.NewRequest(http.MethodPost, c.endpoint, &requestBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	client, ok := ctx.Value(httpclientContextKey).(*http.Client)
	if !ok {
		client = http.DefaultClient
	}
	res, err := ctxhttp.Do(ctx, client, req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		return errors.Wrap(err, "reading body")
	}
	if err := json.NewDecoder(&buf).Decode(&graphResponse); err != nil {
		return errors.Wrap(err, "decoding response")
	}
	if len(graphResponse.Errors) > 0 {
		// return first error
		return graphResponse.Errors[0]
	}
	return nil
}

// WithClient specifies the http.Client that requests will use.
func WithClient(ctx context.Context, client *http.Client) context.Context {
	return context.WithValue(ctx, httpclientContextKey, client)
}

type graphErr struct {
	Message string
}

func (e graphErr) Error() string {
	return "graphql: " + e.Message
}

// Request is a GraphQL request.
type Request struct {
	q     string
	vars  map[string]interface{}
	files []file
}

// NewRequest makes a new Request with the specified string.
func NewRequest(q string) *Request {
	req := &Request{
		q: q,
	}
	return req
}

// Run executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
// If the request fails or the server returns an error, the first error
// will be returned. Use IsGraphQLErr to determine which it was.
func (req *Request) Run(ctx context.Context, response interface{}) error {
	client, err := fromContext(ctx)
	if err != nil {
		return err
	}
	return client.Do(ctx, req, response)
}

// Var sets a variable.
func (req *Request) Var(key string, value interface{}) {
	if req.vars == nil {
		req.vars = make(map[string]interface{})
	}
	req.vars[key] = value
}

// File sets a file to upload.
func (req *Request) File(filename string, r io.Reader) {
	req.files = append(req.files, file{
		Name: filename,
		R:    r,
	})
}

// file represents a file to upload.
type file struct {
	Name string
	R    io.Reader
}
