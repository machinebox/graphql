# graphql

Low-level GraphQL client for Go.

* Simple, familiar API
* Respects `context.Context` cancallation
* Build and execute any kind of GraphQL request
* Use strong Go types for response data
* Use variables and upload files
* Simple error handling

```go
ctx := context.Background()
ctx := graphql.NewContext(ctx, "https://machinebox.io/graphql")
req := graphql.NewRequest(`
    query ($key: String!) {
        items (id:$key) {
            field1
            field2
            field3
        }
    }
`)
req.Var("key", "value")
var respData ResponseStruct
if err := req.Run(ctx, &respData); err != nil {
    log.Fatalln(err)
}
```
