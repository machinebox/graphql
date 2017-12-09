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
)

// Client is a client for accessing a GraphQL dataset.
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(endpoint string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		endpoint: endpoint,
	}
	for _, optionFunc := range opts {
		optionFunc(c)
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	return c, nil
}

// Run executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
// If the request fails or the server returns an error, the first error
// will be returned. Use IsGraphQLErr to determine which it was.
func (c *Client) Run(ctx context.Context, request *Request, response interface{}) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
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
	req = req.WithContext(ctx)
	res, err := c.httpClient.Do(req)
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

// WithHTTPClient specifies the underlying http.Client to use when
// making requests.
func WithHTTPClient(httpclient *http.Client) ClientOption {
	return ClientOption(func(client *Client) {
		client.httpClient = httpclient
	})
}

// ClientOption is a function that modifies the client in some way.
type ClientOption func(*Client)

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
