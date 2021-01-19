package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// Request is a GraphQL request.
type Request struct {
	q     string
	vars  map[string]interface{}
	files []File

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

// OperationName parses operation name from query.
func (req *Request) OperationName() string {
	pattern := regexp.MustCompile(`(mutation|query)\s*(.*?)\(`)
	operation := pattern.FindStringSubmatch(req.Query())[2]

	return operation
}

// Var sets a variable.
func (req *Request) Var(key string, value interface{}) {
	if req.vars == nil {
		req.vars = make(map[string]interface{})
	}

	req.vars[key] = value
}

// Vars gets the variables for this Request.
func (req *Request) Vars() map[string]interface{} {
	return req.vars
}

// Files gets the files in this request.
func (req *Request) Files() []File {
	return req.files
}

// Query gets the query string of this request.
func (req *Request) Query() string {
	return req.q
}

// File sets a file to upload.
// Files are only supported with a Client that was created with
// the UseMultipartForm option.
func (req *Request) File(fieldname, filename string, r io.Reader) {
	req.files = append(req.files, File{
		Field: fieldname,
		Name:  filename,
		R:     r,
	})
}

// OperationsJSON serializes queries following: https://github.com/jaydenseric/graphql-multipart-request-spec.
func (req *Request) OperationsJSON() ([]byte, error) {
	type Operations struct {
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
		Query         string                 `json:"query"`
	}

	operations := Operations{
		OperationName: req.OperationName(),
		Variables:     req.Vars(),
		Query:         req.Query(),
	}

	return json.Marshal(operations)
}

// FileMap returns a string with all the files positions.
func (req *Request) FileMap() string {
	fileMap := ""
	for i, file := range req.Files() {
		fileMap += fmt.Sprintf(`{"%d": ["%s"]}`, i+1, file.Field)
	}

	return fileMap
}

// CopyHeaders copies Request headers to http.Request.
func (req Request) CopyHeaders(r *http.Request) {
	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}
}
