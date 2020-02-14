package graphql

import (
    "context"
    "encoding/json"
    "flag"
    "github.com/gorilla/websocket"
    "github.com/matryer/is"
    "log"
    "net/http"
    "net/http/httptest"
    "testing"
)

var upgrader = websocket.Upgrader{} // use default options



func TestSub(t *testing.T) {
    is := is.New(t)

    flag.Parse()
    log.SetFlags(0)
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        c, err := upgrader.Upgrade(w, r, nil)
        is.NoErr(err)
        var pl subscriptionMessage
        defer c.Close()

        pl.Type = gql_connection_ack
        is.NoErr(c.WriteJSON(pl))

        is.NoErr(c.ReadJSON(&pl))
        is.Equal(string(pl.Type), gql_start)
        var tmp1  struct {Query string `json:"query"`; Variables map[string]string `json:"variables"`}
        is.NoErr(json.Unmarshal(*pl.Payload, &tmp1))
        is.Equal(tmp1.Query, `subscription ($q: String) { cnt }`)
        is.Equal(len(tmp1.Variables),1)
        is.Equal(tmp1.Variables["q"],"foo")
        is.Equal(*pl.Id, "0")

        id := *pl.Id

        pl.Payload = nil
        pl.Id = nil
        pl.Type = gql_connection_ack
        is.NoErr(c.WriteJSON(pl))


        pl.Id = &id
        pl_pl := json.RawMessage(`{"data": "bar"}`)
        pl.Payload = &pl_pl
        pl.Type = gql_data
        is.NoErr(c.WriteJSON(pl))

        is.NoErr(c.ReadJSON(&pl))
        is.Equal(string(pl.Type), gql_stop)
        is.Equal(*pl.Id, id)
        pl.Payload = nil
        pl.Type = gql_complete
        is.NoErr(c.WriteJSON(pl))

        is.NoErr(c.ReadJSON(&pl))
        is.Equal(string(pl.Type), gql_connection_terminate)
    }))
    defer srv.Close()

    ctx, cancel := context.WithCancel(context.Background())
    client := NewClient(srv.URL)
    header := http.Header{}


    cl, err  := client.SubscriptionClient(ctx, header)
    is.NoErr(err)

    vars := make(map[string]interface{})
    vars["q"] = "foo"
    sub, err := cl.Subscribe(&Request{Header:header, q:`subscription ($q: String) { cnt }`, vars: vars})
    is.NoErr(err)
    res := <- sub
    is.Equal(string(*res.Data), `{"data":"bar"}`)

    cl.Unsubscribe(sub)

    defer cancel()
    defer is.NoErr(cl.Close())
}
