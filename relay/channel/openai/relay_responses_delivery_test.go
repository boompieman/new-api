package openai

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failedResponsesWriter struct {
	gin.ResponseWriter
	failAt, writes int
	failFlush      bool
	err            error
}

func (w *failedResponsesWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return 0, w.err
	}
	return w.ResponseWriter.Write(p)
}

func (w *failedResponsesWriter) FlushError() error {
	if w.failFlush {
		return w.err
	}
	w.ResponseWriter.Flush()
	return nil
}

// The upstream stays open after its first event until the relay closes it.
type waitingResponsesBody struct {
	*strings.Reader
	closed chan struct{}
	once   sync.Once
}

func (b *waitingResponsesBody) Read(p []byte) (int, error) {
	if b.Reader.Len() > 0 {
		return b.Reader.Read(p)
	}
	<-b.closed
	return 0, io.EOF
}

func (b *waitingResponsesBody) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

func TestResponsesStreamStopsUpstreamOnDeliveryFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 1
	t.Cleanup(func() { constant.StreamingTimeout = previousTimeout })
	for _, tc := range []struct {
		name      string
		failAt    int
		failFlush bool
		completed bool
	}{
		{name: "event write", failAt: 1},
		{name: "data write", failAt: 2},
		{name: "event separator", failAt: 3},
		{name: "flush", failFlush: true},
		{name: "completion usage survives delivery failure", failAt: 1, completed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, _ := gin.CreateTestContext(httptest.NewRecorder())
			writeErr := errors.New("downstream connection reset")
			writer := &failedResponsesWriter{ResponseWriter: base.Writer, failAt: tc.failAt, failFlush: tc.failFlush, err: writeErr}
			c, _ := gin.CreateTestContext(writer) // Exercise Gin's wrapper hiding FlushError.
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			data := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n"
			if tc.completed {
				data = "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"usage\":{\"input_tokens\":7,\"output_tokens\":3,\"total_tokens\":10}}}\n\n"
			}
			body := &waitingResponsesBody{Reader: strings.NewReader(data), closed: make(chan struct{})}
			t.Cleanup(func() { _ = body.Close() })
			info := &relaycommon.RelayInfo{DisablePing: true, ChannelMeta: &relaycommon.ChannelMeta{}}
			usage, apiErr := OaiResponsesStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: body})
			require.Nil(t, apiErr) // Never retry a partially delivered stream as a new request.
			if tc.completed {
				require.NotNil(t, usage)
				assert.Equal(t, 7, usage.PromptTokens)
				assert.Equal(t, 3, usage.CompletionTokens)
				assert.Equal(t, 10, usage.TotalTokens)
			}
			require.NotNil(t, info.StreamStatus)
			assert.Equal(t, relaycommon.StreamEndReasonHandlerStop, info.StreamStatus.EndReason)
			assert.ErrorIs(t, info.StreamStatus.EndError, writeErr)
			assert.True(t, info.StreamStatus.HasErrors())
			select {
			case <-body.closed:
			default:
				t.Fatal("upstream body was not closed after delivery failure")
			}
		})
	}
}

func TestResponsesStreamRetriesFailureBeforeOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = previousTimeout })
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"status":"in_progress"}}`,
		`data: {"type":"response.in_progress","response":{"status":"in_progress"}}`,
		`data: {"type":"response.failed","response":{"status":"failed","error":{"type":"service_unavailable_error","code":"server_is_overloaded","message":"The AI service is temporarily overloaded."}}}`,
		"",
	}, "\n\n")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := &relaycommon.RelayInfo{DisablePing: true, ChannelMeta: &relaycommon.ChannelMeta{}}

	usage, apiErr := OaiResponsesStreamHandler(c, info, &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	})

	require.Nil(t, usage)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.StatusCode)
	assert.Equal(t, "server_is_overloaded", string(apiErr.GetErrorCode()))
	assert.Empty(t, recorder.Body.String(), "retryable attempts must not leak an SSE prelude to the client")
}

func TestResponsesStreamDoesNotRetryFailureAfterOutput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = previousTimeout })
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"status":"in_progress"}}`,
		`data: {"type":"response.output_text.delta","delta":"partial"}`,
		`data: {"type":"response.failed","response":{"status":"failed","error":{"type":"service_unavailable_error","code":"server_is_overloaded","message":"The AI service is temporarily overloaded."}}}`,
		"",
	}, "\n\n")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	info := &relaycommon.RelayInfo{DisablePing: true, ChannelMeta: &relaycommon.ChannelMeta{}}

	usage, apiErr := OaiResponsesStreamHandler(c, info, &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	})

	require.NotNil(t, usage)
	require.Nil(t, apiErr)
	assert.Contains(t, recorder.Body.String(), `event: response.created`)
	assert.Contains(t, recorder.Body.String(), `event: response.output_text.delta`)
	assert.Contains(t, recorder.Body.String(), `event: response.failed`)
}
