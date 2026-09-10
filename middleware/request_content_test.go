package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRequestContentExchange(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(filepath.Join(t.TempDir(), "content.db"))
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN not configured")
				}
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			var version string
			versionQuery := "SELECT version()"
			if dialect == "sqlite" {
				versionQuery = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(versionQuery).Scan(&version).Error)
			t.Log("database version:", version)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })
			previousDB := model.DB
			model.DB = db
			t.Cleanup(func() { model.DB = previousDB })
			for _, upgrade := range []bool{false, true} {
				require.NoError(t, db.Migrator().DropTable(&model.RequestContent{}, &model.RequestContentAccess{}, &model.Log{}))
				// Log's persisted shape is unchanged from release; emulate an existing deployment.
				if upgrade {
					require.NoError(t, db.AutoMigrate(&model.Log{}))
					require.NoError(t, db.Create(&model.Log{UserId: 9, Content: "existing billing record", RequestId: "legacy"}).Error)
				}
				require.NoError(t, db.AutoMigrate(&model.RequestContent{}, &model.RequestContentAccess{}))
				require.NoError(t, db.AutoMigrate(&model.RequestContent{}, &model.RequestContentAccess{}))
				if upgrade {
					var log model.Log
					require.NoError(t, db.Where("request_id = ?", "legacy").First(&log).Error)
					assert.Equal(t, "existing billing record", log.Content)
				}
				t.Setenv("REQUEST_CONTENT_LOG_ENABLED", "true")
				router := gin.New()
				router.Use(func(c *gin.Context) {
					c.Set("id", 7)
					c.Set("token_id", 42)
					c.Set(common.RequestIdKey, "test-content")
					c.Next()
				})
				router.Use(BodyStorageCleanup(), RequestContent())
				response := "data: {\"choices\":[{\"delta\":{\"content\":\"你好\"}}]}\n\ndata: [DONE]\n\n"
				router.POST("/v1/chat/completions", func(c *gin.Context) {
					bs, err := common.GetBodyStorage(c)
					require.NoError(t, err)
					body, err := bs.Bytes()
					require.NoError(t, err)
					assert.Contains(t, string(body), "hello")
					c.Header("Content-Type", "text/event-stream")
					_, err = c.Writer.WriteString(response)
					require.NoError(t, err)
					require.NoError(t, http.NewResponseController(c.Writer).Flush())
				})
				request := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"test-model","messages":[{"role":"user","content":"hello\u0000😀"}]}`))
				request.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, request)
				assert.Equal(t, response, rec.Body.String())
				_, deniedErr := model.GetRequestContent(context.Background(), "test-content", 7, false)
				require.ErrorIs(t, deniedErr, gorm.ErrRecordNotFound)
				require.NoError(t, model.SetRequestContentAccess(context.Background(), 7, true))
				require.NoError(t, model.SetRequestContentAccess(context.Background(), 7, true))
				entry, err := model.GetRequestContent(context.Background(), "test-content", 7, false)
				require.NoError(t, err)
				assert.True(t, entry.Complete)
				assert.Equal(t, 42, entry.TokenId)
				assert.Equal(t, "hello�😀", entry.Preview)
				assert.Contains(t, entry.Output, "你好")
				_, err = model.GetRequestContent(context.Background(), "test-content", 8, false)
				assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
				_, err = model.GetRequestContent(context.Background(), "test-content", 0, false)
				assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
				_, err = model.GetRequestContent(context.Background(), "test-content", 8, true)
				require.NoError(t, err)
				logs := []*model.Log{{UserId: 7, RequestId: "test-content"}, {UserId: 8, RequestId: "test-content"}}
				require.NoError(t, model.AddRequestContentPreviews(context.Background(), logs))
				assert.Equal(t, "hello�😀", logs[0].RequestPreview)
				assert.False(t, logs[1].HasRequestContent)
				require.NoError(t, model.SetRequestContentAccess(context.Background(), 7, false))
				_, deniedErr = model.GetRequestContent(context.Background(), "test-content", 7, false)
				require.ErrorIs(t, deniedErr, gorm.ErrRecordNotFound)
				_, err = model.GetRequestContent(context.Background(), "test-content", 8, true)
				require.NoError(t, err)
				require.NoError(t, model.SetRequestContentAccess(context.Background(), 7, true))
				duplicate := *entry
				duplicate.ID = 0
				assert.Error(t, model.SaveRequestContent(context.Background(), &duplicate))
				require.NoError(t, db.Model(entry).Update("expires_at", common.GetTimestamp()-1).Error)
				_, err = model.GetRequestContent(context.Background(), "test-content", 7, false)
				assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
				require.NoError(t, model.DeleteExpiredRequestContent(context.Background()))
				var count int64
				require.NoError(t, db.Model(&model.RequestContent{}).Count(&count).Error)
				assert.Zero(t, count)
			}
			require.NoError(t, db.Migrator().DropTable(&model.RequestContent{}, &model.RequestContentAccess{}, &model.Log{}))
		})
	}
}

func TestRequestContentParsing(t *testing.T) {
	for _, tc := range []struct{ name, body, role, text string }{
		{"responses", `{"input":"new question"}`, "user", "new question"},
		{"tool continuation", `{"input":[{"type":"function_call_output","output":"tool result"}]}`, "tool", "tool result"},
		{"claude", `{"messages":[{"role":"user","content":[{"type":"text","text":"hi"},{"type":"image","source":{"data":"SECRET"}}]}]}`, "user", "hi\n[attachment omitted]"},
		{"gemini", `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`, "user", "hello"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]any
			require.NoError(t, common.Unmarshal([]byte(tc.body), &m))
			messages := contentMessages(m)
			require.Len(t, messages, 1)
			assert.Equal(t, capturedMessage{tc.role, tc.text}, messages[0])
		})
	}
	for _, tc := range []struct {
		name, body, text string
		stream, complete bool
	}{
		{"chat", `{"choices":[{"message":{"role":"assistant","content":"answer"}}]}`, "answer", false, true},
		{"claude", `{"content":[{"type":"text","text":"answer"}]}`, "answer", false, true},
		{"gemini", `{"candidates":[{"content":{"parts":[{"text":"answer"}]},"finishReason":"STOP"}]}`, "answer", false, true},
		{"responses final replaces deltas", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"answer\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"answer\"}]}]}}\n", "answer", true, true},
		{"interrupted", "data: {\"choices\":[{\"delta\":{\"content\":\"part\"}}]}\n", "part", true, false},
		{"claude stream", "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"answer\"}}\n\ndata: {\"type\":\"message_stop\"}\n", "answer", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			messages, complete := responseContent([]byte(tc.body), tc.stream)
			assert.Equal(t, tc.complete, complete)
			require.Len(t, messages, 1)
			assert.Equal(t, tc.text, messages[0].Text)
		})
	}
	messagesWithEmptyFinal, done := responseContent([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":\"answer\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"output\":[]}}\n"), true)
	require.True(t, done)
	require.Equal(t, []capturedMessage{{Role: "assistant", Text: "answer"}}, messagesWithEmptyFinal)

	encoded, clipped := boundedContent([]capturedMessage{{"user", strings.Repeat("世\x00", 20000)}})
	assert.True(t, clipped)
	assert.Less(t, len(encoded), 65535)
	assert.True(t, utf8.ValidString(encoded))
	var messages []capturedMessage
	require.NoError(t, common.Unmarshal([]byte(encoded), &messages))
}

func TestRequestContentDisabledDoesNotCapture(t *testing.T) {
	t.Setenv("REQUEST_CONTENT_LOG_ENABLED", "false")
	router := gin.New()
	router.Use(RequestContent())
	router.POST("/v1/responses", func(c *gin.Context) { c.String(200, "unchanged") })
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("POST", "/v1/responses", strings.NewReader("not json")))
	assert.Equal(t, "unchanged", rec.Body.String())
}

func TestRequestContentWriterPreservesFailuresAndBounds(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	writer := &contentResponseWriter{ResponseWriter: c.Writer}
	body := strings.Repeat("a", 1024*1024+1)
	n, err := writer.WriteString(body)
	require.NoError(t, err)
	assert.Equal(t, len(body), n)
	assert.Equal(t, body, recorder.Body.String())
	assert.True(t, writer.truncated)
	assert.Equal(t, 1024*1024, writer.body.Len())
	failing := &contentFailureWriter{ResponseWriter: c.Writer}
	writer = &contentResponseWriter{ResponseWriter: failing}
	_, err = writer.WriteString("test")
	assert.Error(t, err)
	assert.True(t, writer.failed)
	writer.failed = false
	assert.Error(t, http.NewResponseController(writer).Flush())
	assert.True(t, writer.failed)
}

type contentFailureWriter struct{ gin.ResponseWriter }

func (w *contentFailureWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func (w *contentFailureWriter) FlushError() error         { return errors.New("flush failed") }
