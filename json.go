package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

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

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, &requestBody)

	if err != nil {
		return err
	}

	r.Close = c.closeReq
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("Accept", "application/json; charset=utf-8")

	req.CopyHeaders(r)

	return c.Do(ctx, r, gr)
}
