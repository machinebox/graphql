// Package graphql provides a low level GraphQL client.
//
//  // create a client (safe to share across requests)
//  client := graphql.NewClient("https://machinebox.io/graphql")
//
//  // make a request
//  req := graphql.NewRequest(`
//      query ($key: String!) {
//          items (id:$key) {
//              field1
//              field2
//              field3
//          }
//      }
//  `)
//
//  // set any variables
//  req.Var("key", "value")
//
//  // run it and capture the response
//  var respData ResponseStruct
//  if err := client.Run(ctx, req, &respData); err != nil {
//      log.Fatal(err)
//  }
//
// Specify client
//
// To specify your own http.Client, use the WithHTTPClient option:
//  httpclient := &http.Client{}
//  client := graphql.NewClient("https://machinebox.io/graphql", graphql.WithHTTPClient(httpclient))
package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// Client is a client for interacting with a GraphQL API.
type Client struct {
	endpoint         string
	httpClient       *http.Client
	useMultipartForm bool
	retryConfig      RetryConfig
	defaultHeaders   map[string]string

	// Log is called with various debug information.
	// To log to standard out, use:
	//  client.Log = func(s string) { log.Println(s) }
	Log func(s string)
}

// NewClient makes a new Client capable of making GraphQL requests.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	c := &Client{
		endpoint: endpoint,
		Log:      func(string) {},
	}
	for _, optionFunc := range opts {
		optionFunc(c)
	}
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}
	if c.retryConfig.Policy == "" {
		c.retryConfig = defaultNoRetryConfig
	}
	return c
}

func (c *Client) logf(format string, args ...interface{}) {
	c.Log(fmt.Sprintf(format, args...))
}

// RetryConfig defines possible fields that client can supply for their retry strategies
type RetryConfig struct {
	// Optional - Max number of times client should retry
	MaxTries int `json:"maxTries"`
	// Required - Time interval to wait before trying attempt sending a request again
	Interval float64 `json:"interval"`
	// Required - Defines a policy to be used for retry
	Policy PolicyType `json:"policy"`
	// Optional - The max interval of time to wait before retrying
	MaxInterval float64 `json:"maxInterval"`
	// Optional - A mapping of statuses that client should retry.
	// If not specifed, we will use default retry behavior on certain statuses
	RetryStatus map[int]bool `json:"statusToRetry"`
	// Client can use this function to supply some logic to further debug GraphQL request & response
	BeforeRetry func(req *http.Request, resp *http.Response, err error, attemptNum int)
}

// PolicyType defines a type of different possible Policies to be applied towards retrying
type PolicyType string

const (
	// ExponentialBackoff - the interval is doubled after every try until hitting MaxInterval or MaxTries
	ExponentialBackoff PolicyType = "exponential_backoff"
	// Linear - the interval stays the same every try until hitting MaxTries
	Linear PolicyType = "linear"
)

var (
	defaultLinearRetryConfig = RetryConfig{
		MaxTries: 5,
		Interval: 2,
		Policy:   Linear,
	}

	defaultExponentialRetryConfig = RetryConfig{
		MaxTries:    5,
		Interval:    1,
		Policy:      ExponentialBackoff,
		MaxInterval: 16,
	}

	defaultNoRetryConfig = RetryConfig{
		MaxTries: 1,
	}
)

// Wrapper method to send request while optionally applying retry policy
func (c *Client) sendRequest(retryConfig *RetryConfig, req *http.Request) (*http.Response, error) {
	// Client did not specify retry config
	if retryConfig.Policy == "" {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return resp, fmt.Errorf("Error getting response: %v", err)
		}
		return resp, nil
	}

	c.logf("debug original request: %+v", req)

	// Persist request body
	var body io.Reader = req.Body
	tryCount := 0
	statusCode := 0
	for ; tryCount < retryConfig.MaxTries; tryCount++ {
		// Assign request body for new request before retry with a temp buf
		buf := new(bytes.Buffer)
		req.Body = ioutil.NopCloser(io.TeeReader(body, buf))
		c.logf("body buffer: %s", buf.String())

		if req.Context().Err() == context.Canceled {
			return nil, context.Canceled
		}

		c.logf("debug request: %+v", req)
		resp, err := c.httpClient.Do(req)
		c.logf("debug response: %+v", resp)

		if err != nil && !isErrRetryable(err) {
			return resp, err
		}
		if err == nil {
			statusCode = resp.StatusCode
			if !retryConfig.shouldRetry(statusCode) {
				return resp, err
			}
		}

		// Assign buf back to body
		body = buf

		if retryConfig.BeforeRetry != nil {
			retryConfig.BeforeRetry(req, resp, err, tryCount+1)
		}

		c.Log("Will retry after interval expires")

		// Wait for interval
		c.logf("Waiting for interval(%f) to expire...", retryConfig.Interval)
		timer := time.NewTimer(time.Duration(retryConfig.Interval) * time.Second)

		ctx := req.Context()

		select {
		case <-ctx.Done():
			return nil, errors.New("Context finished unexpectedly")

		case <-timer.C:
			// Increase interval
			retryConfig.increaseInterval()
			c.logf("New interval: %f", retryConfig.Interval)
		}
	}

	return nil, fmt.Errorf("Client has retried %d times but unable to get a successful response. Status code: %d", tryCount, statusCode)
}

// Increase interval for exponential backoff policy until hitting MaxInterval
func (config *RetryConfig) increaseInterval() {
	if config.Policy == ExponentialBackoff && config.Interval < config.MaxInterval {
		config.Interval = math.Min(config.Interval*2, config.MaxInterval)
	}
}

// Check if err is retryable
func isErrRetryable(err error) bool {
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
}

// Determines whether the client should retry the request
// If specified, the client will use consumer-specified RetryStatus to retry request based on status code
// Otherwise, retry on 502, 503, 504, and 507
func (config *RetryConfig) shouldRetry(status int) bool {
	if len(config.RetryStatus) > 0 {
		return config.RetryStatus[status]
	}
	// return status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout || status == http.StatusInsufficientStorage
	return (status >= 500 && status <= 599) || status == 429
}

// Determines whether RetryConfig is valid
func (config *RetryConfig) isValid() bool {
	isConfigOptional := config.Policy == ""
	return isConfigOptional || (config.MaxTries > 0 && config.Interval <= config.MaxInterval)
}

// WithRetryConfig allows consumer to assign their retryConfig to the client's private retryConfig
func WithRetryConfig(config RetryConfig) ClientOption {
	return func(client *Client) {
		client.retryConfig = config
	}
}

// WithDefaultLinearRetryConfig provides a default set of value for linear policy
func WithDefaultLinearRetryConfig() ClientOption {
	return func(client *Client) {
		client.retryConfig = defaultLinearRetryConfig
	}
}

// WithDefaultExponentialRetryConfig provides a default set of value for exponential backoff policy
func WithDefaultExponentialRetryConfig() ClientOption {
	return func(client *Client) {
		client.retryConfig = defaultExponentialRetryConfig
	}
}

// WithBeforeRetryHandler provides a handler for beforeRetry
func WithBeforeRetryHandler(beforeRetryHandler func(*http.Request, *http.Response, error, int)) ClientOption {
	return func(client *Client) {
		client.retryConfig.BeforeRetry = beforeRetryHandler
	}
}

// WithDefaultHeaders provides a default set of header values
func WithDefaultHeaders(defaultHeaders map[string]string) ClientOption {
	return func(client *Client) {
		client.defaultHeaders = defaultHeaders
	}
}

// Run executes the query and unmarshals the response from the data field
// into the response object.
// Pass in a nil response object to skip response parsing.
// If the request fails or the server returns an error, the first error
// will be returned.
func (c *Client) Run(ctx context.Context, req *Request, resp interface{}) error {
	// TODO: validate retryConfig

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if len(req.files) > 0 && !c.useMultipartForm {
		return errors.New("cannot send files with PostFields option")
	}
	if c.useMultipartForm {
		return c.runWithPostFields(ctx, req, resp)
	}
	return c.runWithJSON(ctx, req, resp)
}

func (c *Client) runWithJSON(ctx context.Context, req *Request, resp interface{}) error {
	var requestBody bytes.Buffer
	requestBodyObj := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     req.q,
		Variables: req.vars,
	}
	if err := json.NewEncoder(&requestBody).Encode(requestBodyObj); err != nil {
		return errors.Wrap(err, "encode body")
	}
	c.logf(">> variables: %v", req.vars)
	c.logf(">> query: %s", req.q)
	gr := &graphResponse{
		Data: resp,
	}
	r, err := http.NewRequest(http.MethodPost, c.endpoint, &requestBody)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Accept", "application/json; charset=utf-8")
	for key, value := range c.defaultHeaders {
		r.Header.Add(key, value)
	}
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}
	c.logf(">> headers: %v", r.Header)
	r = r.WithContext(ctx)

	return c.executeRequest(gr, r)
}

func getGraphQLResp(reader io.ReadCloser, schema interface{}) error {
	defer reader.Close()

	err := json.NewDecoder(reader).Decode(schema)
	if err != nil {
		return errors.Wrap(err, "decoding response")
	}

	return nil
}

func (c *Client) executeRequest(gr *graphResponse, r *http.Request) error {
	gqlRetryConfig := c.retryConfig
	var body io.Reader = r.Body
	tryCount := 0
	for ; tryCount < gqlRetryConfig.MaxTries; tryCount++ {
		buf := new(bytes.Buffer)
		r.Body = ioutil.NopCloser(io.TeeReader(body, buf))

		retryConfig := c.retryConfig
		res, err := c.sendRequest(&retryConfig, r)
		if err != nil {
			return err
		}
		c.logf("<< %s", buf.String())

		err = getGraphQLResp(res.Body, &gr)
		if err != nil {
			return err
		}
		c.logf("<< gr: %+v", gr)

		if len(gr.Errors) > 0 {
			// Check to see if we should retry based on error field
			if retryConfig.Interval <= c.retryConfig.Interval && shouldRetry(gr.Errors) {
				body = buf
				timer := time.NewTimer(time.Duration(gqlRetryConfig.Interval) * time.Second)
				ctx := r.Context()

				select {
				case <-ctx.Done():
					return ctx.Err()

				case <-timer.C:
					// Increase interval
					gqlRetryConfig.increaseInterval()
					c.logf("New interval: %f", gqlRetryConfig.Interval)
				}

			} else {
				c.Log("debug return error")
				return getAggrErr(gr.Errors)
			}

		} else {
			// No error so return
			return nil
		}
	}

	return fmt.Errorf("Client has retried %d times but unable to get a successful response. Error: %s", tryCount, getAggrErr(gr.Errors))
}

func (c *Client) runWithPostFields(ctx context.Context, req *Request, resp interface{}) error {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	if err := writer.WriteField("query", req.q); err != nil {
		return errors.Wrap(err, "write query field")
	}
	var variablesBuf bytes.Buffer
	if len(req.vars) > 0 {
		variablesField, err := writer.CreateFormField("variables")
		if err != nil {
			return errors.Wrap(err, "create variables field")
		}
		if err := json.NewEncoder(io.MultiWriter(variablesField, &variablesBuf)).Encode(req.vars); err != nil {
			return errors.Wrap(err, "encode variables")
		}
	}
	for i := range req.files {
		part, err := writer.CreateFormFile(req.files[i].Field, req.files[i].Name)
		if err != nil {
			return errors.Wrap(err, "create form file")
		}
		if _, err := io.Copy(part, req.files[i].R); err != nil {
			return errors.Wrap(err, "preparing file")
		}
	}
	if err := writer.Close(); err != nil {
		return errors.Wrap(err, "close writer")
	}
	c.logf(">> variables: %s", variablesBuf.String())
	c.logf(">> files: %d", len(req.files))
	c.logf(">> query: %s", req.q)
	gr := &graphResponse{
		Data: resp,
	}
	r, err := http.NewRequest(http.MethodPost, c.endpoint, &requestBody)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Accept", "application/json; charset=utf-8")
	for key, value := range c.defaultHeaders {
		r.Header.Add(key, value)
	}
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}
	c.logf(">> headers: %v", r.Header)
	r = r.WithContext(ctx)

	return c.executeRequest(gr, r)
}

// WithHTTPClient specifies the underlying http.Client to use when
// making requests.
//  NewClient(endpoint, WithHTTPClient(specificHTTPClient))
func WithHTTPClient(httpclient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpclient
	}
}

// UseMultipartForm uses multipart/form-data and activates support for
// files.
func UseMultipartForm() ClientOption {
	return func(client *Client) {
		client.useMultipartForm = true
	}
}

// ClientOption are functions that are passed into NewClient to
// modify the behaviour of the Client.
type ClientOption func(*Client)

type graphResponse struct {
	Data   interface{}
	Errors []graphErr
}

// Request is a GraphQL request.
type Request struct {
	q     string
	vars  map[string]interface{}
	files []file

	// Header represent any request headers that will be set
	// when the request is made.
	Header http.Header
}

// NewRequest makes a new Request with the specified string.
func NewRequest(q string) *Request {
	req := &Request{
		q:      q,
		Header: make(map[string][]string),
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
// Files are only supported with a Client that was created with
// the UseMultipartForm option.
func (req *Request) File(fieldname, filename string, r io.Reader) {
	req.files = append(req.files, file{
		Field: fieldname,
		Name:  filename,
		R:     r,
	})
}

// file represents a file to upload.
type file struct {
	Field string
	Name  string
	R     io.Reader
}
