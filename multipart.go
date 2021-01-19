package graphql

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/pkg/errors"
)

func (c *Client) runWithPostFields(ctx context.Context, req *Request, resp interface{}) error {
	operations, err := req.OperationsJSON()
	if err != nil {
		return err
	}

	params := map[string]string{
		"operations": string(operations),
		"map":        req.FileMap(),
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for i, file := range req.files {
		key := fmt.Sprintf("%d", i+1)

		part, err := writer.CreateFormFile(key, file.Name)
		if err != nil {
			return errors.Wrap(err, "creating form")
		}

		if _, err := io.Copy(part, file.R); err != nil {
			return errors.Wrap(err, "preparing file")
		}
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err = writer.Close()
	if err != nil {
		return errors.Wrap(err, "writing fields")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	c.logf(">> operators: %s", operations)
	c.logf(">> files: %d", len(req.files))
	c.logf(">> query: %s", req.q)

	gr := &graphResponse{
		Data: resp,
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, &body)
	if err != nil {
		return err
	}

	r.Close = c.closeReq
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Accept", "application/json; charset=utf-8")

	req.CopyHeaders(r)

	return c.Do(ctx, r, gr)
}
