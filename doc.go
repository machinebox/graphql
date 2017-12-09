// Package graphql provides a low level GraphQL client.
//
//  // create a client (safe to share across requests)
//  ctx := context.Background()
//  client, err := graphql.NewClient(ctx, "https://machinebox.io/graphql")
//  if err != nil {
//      log.Fatal(err)
//  }
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
package graphql
