# graphql

Low-level GraphQL client for Go.

* Simple, familiar API
* Respects `context.Context` timeouts and cancallation
* Build and execute any kind of GraphQL request
* Use strong Go types for response data
* Use variables and upload files
* Simple error handling

```go
// make a request
req := graphql.NewRequest(`
    query ($key: String!) {
        items (id:$key) {
            field1
            field2
            field3
        }
    }
`)

// set any variables
req.Var("key", "value")

// get a context
ctx := context.Background()
ctx := graphql.NewContext(ctx, "https://machinebox.io/graphql")

// run it and capture the response
var respData ResponseStruct
if err := req.Run(ctx, &respData); err != nil {
    log.Fatalln(err)
}
```
