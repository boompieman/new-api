package controller

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLogExportControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))
	return db
}

func requestLogExport(t *testing.T, path string, userId *int) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, path, nil)
	if userId == nil {
		ExportAllLogs(context)
	} else {
		context.Set("id", *userId)
		ExportUserLogs(context)
	}
	return recorder
}

func readLogExportCSV(t *testing.T, recorder *httptest.ResponseRecorder) [][]string {
	t.Helper()

	body := strings.TrimPrefix(recorder.Body.String(), "\ufeff")
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	return records
}

func TestExportAllLogsStreamsEveryMatchingRow(t *testing.T) {
	db := setupLogExportControllerTestDB(t)
	channel := model.Channel{Name: "primary", Key: "test-key", Group: "default"}
	require.NoError(t, db.Create(&channel).Error)

	logs := make([]model.Log, 205)
	for i := range logs {
		logs[i] = model.Log{
			UserId:           7,
			Username:         "alice",
			CreatedAt:        int64(1_700_000_000 + i),
			Type:             model.LogTypeConsume,
			Content:          fmt.Sprintf("row-%d", i+1),
			TokenName:        "default",
			ModelName:        "gpt-test",
			ChannelId:        channel.Id,
			Group:            "default",
			PromptTokens:     10,
			CompletionTokens: 5,
			Other:            "{}",
		}
	}
	logs[len(logs)-1].Content = "=SUM(A1:A2)"
	require.NoError(t, db.Create(&logs).Error)

	recorder := requestLogExport(
		t,
		"/api/log/export?start_timestamp=1700000000&end_timestamp=1700001000&model_name=gpt-test",
		nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "text/csv; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Header().Get("Content-Disposition"), "attachment;")
	records := readLogExportCSV(t, recorder)
	require.Len(t, records, 206)
	assert.Equal(t, usageLogCSVHeaders, records[0])
	assert.Equal(t, "205", records[1][0])
	assert.Equal(t, "primary", records[1][6])
	assert.Equal(t, "'=SUM(A1:A2)", records[1][22])
}

func TestExportUserLogsKeepsOnlyOwnedNonAdminData(t *testing.T) {
	db := setupLogExportControllerTestDB(t)
	channel := model.Channel{Name: "private-channel", Key: "test-key", Group: "default"}
	require.NoError(t, db.Create(&channel).Error)
	require.NoError(t, db.Create(&[]model.Log{
		{
			UserId:    7,
			Username:  "alice",
			CreatedAt: 1_700_000_001,
			Type:      model.LogTypeConsume,
			ChannelId: channel.Id,
			Group:     "default",
			ModelName: "gpt-test",
			Content:   "owned",
			Other:     `{"admin_info":{"secret":"hidden"},"audit_info":{"path":"/admin"},"visible":"yes"}`,
		},
		{
			UserId:    8,
			Username:  "bob",
			CreatedAt: 1_700_000_002,
			Type:      model.LogTypeConsume,
			ChannelId: channel.Id,
			Group:     "default",
			ModelName: "gpt-test",
			Content:   "not-owned",
			Other:     "{}",
		},
	}).Error)

	userId := 7
	recorder := requestLogExport(
		t,
		"/api/log/self/export?start_timestamp=1700000000&end_timestamp=1700001000",
		&userId,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	records := readLogExportCSV(t, recorder)
	require.Len(t, records, 2)
	assert.Equal(t, "1", records[1][0])
	assert.Empty(t, records[1][6])
	assert.Equal(t, "alice", records[1][8])
	assert.Equal(t, "owned", records[1][22])
	assert.NotContains(t, records[1][23], "admin_info")
	assert.NotContains(t, records[1][23], "audit_info")
	assert.Contains(t, records[1][23], "visible")
}

func TestExportLogsRejectsInvalidTimeRangeBeforeStreaming(t *testing.T) {
	recorder := requestLogExport(
		t,
		"/api/log/export?start_timestamp=1700001000&end_timestamp=1700000000",
		nil,
	)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "start_timestamp must not be after end_timestamp")
	assert.Empty(t, recorder.Header().Get("Content-Disposition"))
}
