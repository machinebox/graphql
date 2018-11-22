package graphql

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	errNotFound           graphErrType = "not_found"
	errNotAllowed         graphErrType = "not_allowed"
	errInvalidInput       graphErrType = "invalid_input"
	errCapacityExceeded   graphErrType = "capacity_exceeded"
	errAuthentication     graphErrType = "authentication_error"
	errImplemented        graphErrType = "not_implemented"
	errServiceUnavailable graphErrType = "service_unavailable"
	errServiceFailure     graphErrType = "service_failure"
	errInternal           graphErrType = "internal_error"
)

type graphErr struct {
	Message    string        `json:"message,omitempty"`
	Name       graphErrType  `json:"name,omitempty"`
	TimeThrown string        `json:"time_thrown,omitempty"`
	Data       interface{}   `json:"data,omitempty"`
	Path       []interface{} `json:"path,omitempty"`
	Locations  []graphErrLoc `json:"locations,omitempty"`
}

type graphErrData struct {
	ObjectID   string `json:"objectId,omitempty"`
	ObjectType string `json:"objectType,omitempty"`
	ErrorID    string `json:"errorId,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
}

type graphErrLoc struct {
	Line   int64 `json:"line"`
	Column int64 `json:"column"`
}

type graphErrType string

func getAggrErr(errList []graphErr) error {
	var buffer bytes.Buffer
	buffer.WriteString("graphql: ")
	for idx, err := range errList {
		buffer.WriteString(fmt.Sprintf("error %d: message (%s), data (%+v). ", idx, err.Message, err.Data))
	}

	return errors.New(buffer.String())
}

func shouldRetry(errList []graphErr) bool {
	for _, err := range errList {
		if err.Name == errCapacityExceeded || err.Name == errServiceUnavailable || err.Name == errServiceFailure || err.Name == errInternal {
			return true
		}
	}

	return false
}
