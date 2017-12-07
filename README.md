# graphql

Low-level GraphQL client for Go.

* Simple, familiar API
* Respects context.Context cancallation
* Build and execute any kind of GraphQL request
* Use variables and upload files

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
var res ResponseStruct
if err := req.Run(ctx, &resp); err != nil {
    log.Fatalln(err)
}
```
