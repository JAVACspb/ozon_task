package graph

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/websocket"
	"github.com/ozosanek/ozon_task/internal/graph/generated"
	"github.com/ozosanek/ozon_task/internal/service"
	"github.com/ozosanek/ozon_task/internal/storage/memory"
	"github.com/stretchr/testify/require"
)

func TestCommentAddedSubscriptionGraphQLTransportWS(t *testing.T) {
	svc := service.New(memory.New())
	resolver := NewResolver(svc)

	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	srv.AddTransport(transport.Websocket{KeepAlivePingInterval: time.Millisecond})
	srv.AddTransport(transport.POST{})

	httpServer := httptest.NewServer(srv)
	defer httpServer.Close()

	postID := mustGraphQLPost(t, httpServer.URL, `mutation {
		createPost(input: {
			authorName: "roman"
			title: "post"
			body: "body"
			commentsEnabled: true
		}) { id }
	}`, "createPost", "id")

	dialer := websocket.Dialer{Subprotocols: []string{"graphql-transport-ws"}}
	conn, resp, err := dialer.Dial(strings.Replace(httpServer.URL, "http://", "ws://", 1), nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	defer conn.Close()

	require.NoError(t, conn.WriteJSON(map[string]any{"type": "connection_init", "payload": map[string]any{}}))
	requireMessageType(t, conn, "connection_ack")

	require.NoError(t, conn.WriteJSON(map[string]any{
		"id":   "sub-1",
		"type": "subscribe",
		"payload": map[string]any{
			"query": `subscription {
				commentAdded(postId: "` + postID + `") {
					id
					postId
					authorName
					body
				}
			}`,
		},
	}))

	commentID := mustGraphQLPost(t, httpServer.URL, `mutation {
		createComment(input: {
			postId: "`+postID+`"
			authorName: "alex"
			body: "live"
		}) { id }
	}`, "createComment", "id")

	message := requireMessageType(t, conn, "next")
	payload := message["payload"].(map[string]any)
	data := payload["data"].(map[string]any)
	commentAdded := data["commentAdded"].(map[string]any)
	require.Equal(t, commentID, commentAdded["id"])
	require.Equal(t, postID, commentAdded["postId"])
}

func requireMessageType(t *testing.T, conn *websocket.Conn, want string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	require.NoError(t, conn.SetReadDeadline(deadline))

	for {
		var message map[string]any
		require.NoError(t, conn.ReadJSON(&message))
		if message["type"] == "ping" {
			require.NoError(t, conn.WriteJSON(map[string]any{"type": "pong"}))
			continue
		}
		require.Equal(t, want, message["type"], "message: %#v", message)
		return message
	}
}

func mustGraphQLPost(t *testing.T, url string, query string, object string, field string) string {
	t.Helper()

	body, err := json.Marshal(map[string]any{"query": query})
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload struct {
		Data map[string]map[string]string `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))

	return payload.Data[object][field]
}
