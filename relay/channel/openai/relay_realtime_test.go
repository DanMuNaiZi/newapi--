package openai

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type realtimeHandlerResult struct {
	err   *types.NewAPIError
	usage *dto.RealtimeUsage
}

type controlledRealtimeConn struct {
	net.Conn
	blockWrites  atomic.Bool
	writeStarted chan struct{}
	releaseWrite chan struct{}
	closed       chan struct{}
	writeOnce    sync.Once
	releaseOnce  sync.Once
	closeOnce    sync.Once
}

func (c *controlledRealtimeConn) Write(p []byte) (int, error) {
	if c.blockWrites.Load() {
		c.writeOnce.Do(func() { close(c.writeStarted) })
		<-c.releaseWrite
	}
	return c.Conn.Write(p)
}

func (c *controlledRealtimeConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

func (c *controlledRealtimeConn) release() {
	c.releaseOnce.Do(func() { close(c.releaseWrite) })
}

type controlledRealtimeListener struct {
	net.Listener
	accepted chan *controlledRealtimeConn
}

func (l *controlledRealtimeListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	controlled := &controlledRealtimeConn{
		Conn:         conn,
		writeStarted: make(chan struct{}),
		releaseWrite: make(chan struct{}),
		closed:       make(chan struct{}),
	}
	l.accepted <- controlled
	return controlled, nil
}

func newRealtimeWebsocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()

	type acceptResult struct {
		conn *websocket.Conn
		err  error
	}
	accepted := make(chan acceptResult, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		accepted <- acceptResult{conn: conn, err: err}
	}))
	t.Cleanup(server.Close)

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	result := <-accepted
	require.NoError(t, result.err)
	require.NotNil(t, result.conn)

	t.Cleanup(func() {
		_ = client.Close()
		_ = result.conn.Close()
	})
	return result.conn, client
}

func newControlledRealtimeWebsocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn, *controlledRealtimeConn) {
	t.Helper()

	type acceptResult struct {
		conn *websocket.Conn
		err  error
	}
	websocketAccepted := make(chan acceptResult, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		websocketAccepted <- acceptResult{conn: conn, err: err}
	}))
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	controlledListener := &controlledRealtimeListener{
		Listener: listener,
		accepted: make(chan *controlledRealtimeConn, 1),
	}
	server.Listener = controlledListener
	server.Start()
	t.Cleanup(server.Close)

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	controlled := <-controlledListener.accepted
	result := <-websocketAccepted
	require.NoError(t, result.err)
	require.NotNil(t, result.conn)

	t.Cleanup(func() {
		controlled.release()
		_ = client.Close()
		_ = result.conn.Close()
	})
	return result.conn, client, controlled
}

func runRealtimeResponseDone(t *testing.T, status string, normalClientClose bool) map[string]interface{} {
	t.Helper()

	clientServerConn, downstreamClient := newRealtimeWebsocketPair(t)
	upstreamServer, targetClientConn := newRealtimeWebsocketPair(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
	info := &relaycommon.RelayInfo{
		ClientWs:       clientServerConn,
		TargetWs:       targetClientConn,
		StartTime:      time.Now(),
		UsePrice:       true,
		IsFirstRequest: true,
	}
	resultCh := make(chan realtimeHandlerResult, 1)
	go func() {
		err, usage := OpenaiRealtimeHandler(c, info)
		resultCh <- realtimeHandlerResult{err: err, usage: usage}
	}()

	payload, err := common.Marshal(dto.RealtimeEvent{
		Type: dto.RealtimeEventTypeResponseDone,
		Response: &dto.RealtimeResponse{
			Status: status,
			Usage:  &dto.RealtimeUsage{TotalTokens: 1, OutputTokens: 1},
		},
	})
	require.NoError(t, err)
	require.NoError(t, upstreamServer.WriteMessage(websocket.TextMessage, payload))
	_, _, err = downstreamClient.ReadMessage()
	require.NoError(t, err)

	if normalClientClose {
		require.NoError(t, downstreamClient.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		))
	} else {
		require.NoError(t, downstreamClient.UnderlyingConn().Close())
	}

	select {
	case result := <-resultCh:
		require.Nil(t, result.err)
		require.NotNil(t, result.usage)
		return service.GenerateWssOtherInfo(c, info, result.usage, 1, 1, 1, 1, 1, 0, 1)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "realtime handler did not return")
		return nil
	}
}

func TestOpenaiRealtimeHandlerRecordsCompletedResponseEvidence(t *testing.T) {
	other := runRealtimeResponseDone(t, "completed", true)

	assert.Equal(t, true, other["realtime_completed"])
}

func TestOpenaiRealtimeHandlerRejectsNonCompletedResponseEvidence(t *testing.T) {
	for _, status := range []string{"", "failed", "incomplete"} {
		t.Run(status, func(t *testing.T) {
			other := runRealtimeResponseDone(t, status, true)

			assert.NotEqual(t, true, other["realtime_completed"])
		})
	}
}

func TestOpenaiRealtimeHandlerRejectsCompletedResponseAfterAbnormalClientClose(t *testing.T) {
	other := runRealtimeResponseDone(t, "completed", false)

	assert.NotEqual(t, true, other["realtime_completed"])
}

func TestOpenaiRealtimeHandlerWaitsForInFlightCompletedResponseBeforeFinalizing(t *testing.T) {
	clientServerConn, downstreamClient, controlledClientConn := newControlledRealtimeWebsocketPair(t)
	upstreamServer, targetClientConn := newRealtimeWebsocketPair(t)
	clientServerConn.SetCloseHandler(func(int, string) error { return nil })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
	info := &relaycommon.RelayInfo{
		ClientWs:       clientServerConn,
		TargetWs:       targetClientConn,
		StartTime:      time.Now(),
		UsePrice:       true,
		IsFirstRequest: true,
	}
	resultCh := make(chan realtimeHandlerResult, 1)
	go func() {
		err, usage := OpenaiRealtimeHandler(c, info)
		resultCh <- realtimeHandlerResult{err: err, usage: usage}
	}()

	controlledClientConn.blockWrites.Store(true)
	payload, err := common.Marshal(dto.RealtimeEvent{
		Type: dto.RealtimeEventTypeResponseDone,
		Response: &dto.RealtimeResponse{
			Status: "completed",
			Usage:  &dto.RealtimeUsage{TotalTokens: 1, OutputTokens: 1},
		},
	})
	require.NoError(t, err)
	require.NoError(t, upstreamServer.WriteMessage(websocket.TextMessage, payload))

	select {
	case <-controlledClientConn.writeStarted:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "completed response did not reach the controlled client write")
	}
	require.NoError(t, downstreamClient.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))

	select {
	case <-controlledClientConn.closed:
	case <-resultCh:
		require.FailNow(t, "handler finalized while the completed response was still in flight")
	case <-time.After(2 * time.Second):
		require.FailNow(t, "handler did not begin a synchronized shutdown")
	}
	controlledClientConn.release()

	select {
	case result := <-resultCh:
		require.Nil(t, result.err)
		require.NotNil(t, result.usage)
		other := service.GenerateWssOtherInfo(c, info, result.usage, 1, 1, 1, 1, 1, 0, 1)
		assert.NotEqual(t, true, other["realtime_completed"])
	case <-time.After(2 * time.Second):
		require.FailNow(t, "realtime handler did not finish after releasing the client write")
	}
}

func TestOpenaiRealtimeHandlerRejectsCompletionWhenFinalUsageSettlementFails(t *testing.T) {
	previousDB := model.DB
	previousRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		common.RedisEnabled = previousRedisEnabled
	})

	clientServerConn, downstreamClient := newRealtimeWebsocketPair(t)
	upstreamServer, targetClientConn := newRealtimeWebsocketPair(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/realtime", nil)
	info := &relaycommon.RelayInfo{
		ClientWs:        clientServerConn,
		TargetWs:        targetClientConn,
		StartTime:       time.Now(),
		UsePrice:        true,
		IsFirstRequest:  true,
		OriginModelName: "gpt-4o-mini",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-4o-mini",
		},
	}
	resultCh := make(chan realtimeHandlerResult, 1)
	go func() {
		handlerErr, usage := OpenaiRealtimeHandler(c, info)
		resultCh <- realtimeHandlerResult{err: handlerErr, usage: usage}
	}()

	completedPayload, err := common.Marshal(dto.RealtimeEvent{
		Type: dto.RealtimeEventTypeResponseDone,
		Response: &dto.RealtimeResponse{
			Status: "completed",
			Usage:  &dto.RealtimeUsage{TotalTokens: 1, OutputTokens: 1},
		},
	})
	require.NoError(t, err)
	require.NoError(t, upstreamServer.WriteMessage(websocket.TextMessage, completedPayload))
	_, _, err = downstreamClient.ReadMessage()
	require.NoError(t, err)

	pendingInput, err := common.Marshal(dto.RealtimeEvent{
		Type: dto.RealtimeEventTypeSessionUpdate,
		Session: &dto.RealtimeSession{
			Instructions: "pending input that must be settled before finalizing",
		},
	})
	require.NoError(t, err)
	require.NoError(t, downstreamClient.WriteMessage(websocket.TextMessage, pendingInput))
	_, _, err = upstreamServer.ReadMessage()
	require.NoError(t, err)
	info.UsePrice = false
	require.NoError(t, downstreamClient.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))

	select {
	case result := <-resultCh:
		require.Nil(t, result.err)
		require.NotNil(t, result.usage)
		other := service.GenerateWssOtherInfo(c, info, result.usage, 1, 1, 1, 1, 1, 0, 1)
		assert.NotEqual(t, true, other["realtime_completed"])
	case <-time.After(2 * time.Second):
		require.FailNow(t, "realtime handler did not return after final settlement failure")
	}
}
