// Package graphql provides a low level GraphQL client.
//
//     ctx := context.Background()
//     ctx = graphql.NewContext(ctx, "https://machinebox.io/graphql")
//     r := graphql.NewRequest(`
//         query ($key: String!) {
//	           items (id:$key) {
//	               field1
//                 field2
//                 field3
//             }
//         }
//     `)
//     r.Var("key", "value")
//     var res ResponseStruct
//     if err := r.Run(ctx, &resp); err != nil {
//         log.Fatalln(err)
//     }
package graphql
