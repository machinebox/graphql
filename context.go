package graphql

import "context"

// contextKey provides unique keys for context values.
type contextKey string

// clientContextKey is the context value key for the Client.
var clientContextKey = contextKey("graphql client context")

// fromContext gets the client from the specified
// Context.
func fromContext(ctx context.Context) *client {
	c, _ := ctx.Value(clientContextKey).(*client)
	return c
}

// NewContext makes a new context.Context that enables requests.
func NewContext(parent context.Context, endpoint string) context.Context {
	client := &client{
		endpoint: endpoint,
	}
	return context.WithValue(parent, clientContextKey, client)
}
