package graphql

import (
	"context"
	"testing"

	"github.com/matryer/is"
)

func TestNewContext(t *testing.T) {
	is := is.New(t)

	testContextKey := contextKey("something")

	ctx := context.Background()
	ctx = context.WithValue(ctx, testContextKey, true)

	endpoint := "https://server.com/graphql"
	ctx = NewContext(ctx, endpoint)

	vclient, err := fromContext(ctx)
	is.NoErr(err)
	is.Equal(vclient.endpoint, endpoint)

	vclient2, err := fromContext(ctx)
	is.NoErr(err)
	is.Equal(vclient, vclient2)

	is.Equal(ctx.Value(testContextKey), true) // normal context stuff should work
}
