package controller

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const logExportFlushInterval = 500

var usageLogCSVHeaders = []string{
	"id",
	"created_at",
	"created_time",
	"type",
	"type_name",
	"channel",
	"channel_name",
	"user_id",
	"username",
	"token_id",
	"token_name",
	"model_name",
	"group",
	"prompt_tokens",
	"completion_tokens",
	"total_tokens",
	"quota",
	"use_time",
	"is_stream",
	"ip",
	"request_id",
	"upstream_request_id",
	"content",
	"other",
}

func parseLogExportQuery(c *gin.Context, userId *int) (model.LogExportQuery, error) {
	query := model.LogExportQuery{
		ModelName:         c.Query("model_name"),
		Username:          c.Query("username"),
		TokenName:         c.Query("token_name"),
		Group:             c.Query("group"),
		RequestId:         c.Query("request_id"),
		UpstreamRequestId: c.Query("upstream_request_id"),
		UserId:            userId,
	}

	var err error
	if query.LogType, err = parseOptionalLogExportInt(c, "type"); err != nil {
		return query, err
	}
	if query.Channel, err = parseOptionalLogExportInt(c, "channel"); err != nil {
		return query, err
	}
	if query.StartTimestamp, err = parseRequiredLogExportTimestamp(c, "start_timestamp"); err != nil {
		return query, err
	}
	if query.EndTimestamp, err = parseRequiredLogExportTimestamp(c, "end_timestamp"); err != nil {
		return query, err
	}
	if query.StartTimestamp > query.EndTimestamp {
		return query, fmt.Errorf("start_timestamp must not be after end_timestamp")
	}
	return query, nil
}

func parseOptionalLogExportInt(c *gin.Context, key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}

func parseRequiredLogExportTimestamp(c *gin.Context, key string) (int64, error) {
	value := c.Query(key)
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive Unix timestamp", key)
	}
	return parsed, nil
}

func ExportAllLogs(c *gin.Context) {
	exportLogsCSV(c, nil)
}

func ExportUserLogs(c *gin.Context) {
	userId := c.GetInt("id")
	exportLogsCSV(c, &userId)
}

func exportLogsCSV(c *gin.Context, userId *int) {
	query, err := parseLogExportQuery(c, userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	iterator, err := model.OpenLogExportIterator(c.Request.Context(), query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer func() {
		if closeErr := iterator.Close(); closeErr != nil {
			logger.LogError(c.Request.Context(), "failed to close usage log export iterator: "+closeErr.Error())
		}
	}()

	firstLog, err := iterator.Next()
	if err != nil && err != io.EOF {
		common.ApiError(c, err)
		return
	}

	filename := fmt.Sprintf(
		"usage-logs-%s-%s.csv",
		time.Unix(query.StartTimestamp, 0).UTC().Format("20060102-1504"),
		time.Unix(query.EndTimestamp, 0).UTC().Format("20060102-1504"),
	)
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("X-Accel-Buffering", "no")
	c.Header("X-Content-Type-Options", "nosniff")

	if _, err = c.Writer.Write([]byte{0xef, 0xbb, 0xbf}); err != nil {
		return
	}
	csvWriter := csv.NewWriter(c.Writer)
	csvWriter.UseCRLF = true
	if err = csvWriter.Write(usageLogCSVHeaders); err != nil {
		return
	}

	written := 0
	if firstLog != nil {
		if err = csvWriter.Write(usageLogCSVRecord(firstLog)); err != nil {
			return
		}
		written++
	}

	for err != io.EOF {
		log, nextErr := iterator.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			logger.LogError(c.Request.Context(), "failed while streaming usage log export: "+nextErr.Error())
			return
		}
		if err = csvWriter.Write(usageLogCSVRecord(log)); err != nil {
			return
		}
		written++
		if written%logExportFlushInterval == 0 {
			csvWriter.Flush()
			if err = csvWriter.Error(); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}

	csvWriter.Flush()
	if err = csvWriter.Error(); err != nil {
		logger.LogError(c.Request.Context(), "failed to finish usage log CSV export: "+err.Error())
	}
}

func usageLogCSVRecord(log *model.Log) []string {
	return []string{
		strconv.Itoa(log.Id),
		strconv.FormatInt(log.CreatedAt, 10),
		time.Unix(log.CreatedAt, 0).UTC().Format("2006-01-02 15:04:05"),
		strconv.Itoa(log.Type),
		usageLogTypeName(log.Type),
		strconv.Itoa(log.ChannelId),
		spreadsheetSafeCSVText(log.ChannelName),
		strconv.Itoa(log.UserId),
		spreadsheetSafeCSVText(log.Username),
		strconv.Itoa(log.TokenId),
		spreadsheetSafeCSVText(log.TokenName),
		spreadsheetSafeCSVText(log.ModelName),
		spreadsheetSafeCSVText(log.Group),
		strconv.Itoa(log.PromptTokens),
		strconv.Itoa(log.CompletionTokens),
		strconv.Itoa(log.PromptTokens + log.CompletionTokens),
		strconv.Itoa(log.Quota),
		strconv.Itoa(log.UseTime),
		strconv.FormatBool(log.IsStream),
		spreadsheetSafeCSVText(log.Ip),
		spreadsheetSafeCSVText(log.RequestId),
		spreadsheetSafeCSVText(log.UpstreamRequestId),
		spreadsheetSafeCSVText(log.Content),
		spreadsheetSafeCSVText(log.Other),
	}
}

func spreadsheetSafeCSVText(value string) string {
	if value == "" {
		return value
	}
	if strings.ContainsRune("=+-@\t\r", rune(value[0])) {
		return "'" + value
	}
	return value
}

func usageLogTypeName(logType int) string {
	switch logType {
	case model.LogTypeTopup:
		return "Top-up"
	case model.LogTypeConsume:
		return "Consume"
	case model.LogTypeManage:
		return "Manage"
	case model.LogTypeSystem:
		return "System"
	case model.LogTypeError:
		return "Error"
	case model.LogTypeRefund:
		return "Refund"
	case model.LogTypeLogin:
		return "Login"
	default:
		return "Unknown"
	}
}
