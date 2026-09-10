package middleware

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// RequestContent records one HTTP exchange, not an inferred conversation.
func RequestContent() gin.HandlerFunc {
	enabled := common.GetEnvOrDefaultBool("REQUEST_CONTENT_LOG_ENABLED", true)
	// Fixed retention and bounds keep the first version predictable.
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		supported := path == "/v1/chat/completions" || path == "/v1/responses" || path == "/v1/messages" || strings.HasSuffix(path, ":generateContent") || strings.HasSuffix(path, ":streamGenerateContent")
		if !enabled || c.Request.Method != "POST" || !supported || c.GetInt("id") <= 0 {
			c.Next()
			return
		}
		bs, err := common.GetBodyStorage(c)
		if err != nil {
			c.Next()
			return
		}
		body, err := bs.Bytes()
		if err != nil {
			c.Next()
			return
		}
		var request map[string]any
		if common.Unmarshal(body, &request) != nil {
			c.Next()
			return
		}
		input := contentMessages(request)
		inputJSON, clipped := boundedContent(input)
		preview := ""
		// Only a trailing user message is a new question. Tool continuations are not relabelled.
		if len(input) > 0 && input[len(input)-1].Role == "user" {
			text := []rune(strings.ReplaceAll(input[len(input)-1].Text, "\x00", "�"))
			preview = string(text[:min(120, len(text))])
		}
		writer := &contentResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		output, complete := responseContent(writer.body.Bytes(), strings.Contains(writer.Header().Get("Content-Type"), "text/event-stream"))
		outputJSON, outputClipped := boundedContent(output)
		modelName, _ := request["model"].(string)
		if modelName == "" {
			modelName = c.GetString("original_model")
		}
		modelRunes := []rune(strings.ReplaceAll(modelName, "\x00", "�"))
		modelName = string(modelRunes[:min(255, len(modelRunes))])
		entry := &model.RequestContent{
			RequestId: c.GetString(common.RequestIdKey), UserId: c.GetInt("id"), TokenId: c.GetInt("token_id"), Model: modelName,
			ExpiresAt: common.GetTimestamp() + 30*24*60*60, Preview: preview, Input: inputJSON, Output: outputJSON,
			Truncated:  clipped || outputClipped || writer.truncated,
			Complete:   complete && !writer.failed && !writer.truncated && writer.Status() < 400 && c.Request.Context().Err() == nil,
			StatusCode: writer.Status(),
		}
		// Bound the DB write and keep failures out of the relay response. No unbounded background jobs.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := model.SaveRequestContent(ctx, entry); err != nil {
			common.SysError("request content save failed for " + entry.RequestId + ": " + err.Error())
		}
	}
}

type contentResponseWriter struct {
	gin.ResponseWriter
	body      bytes.Buffer
	truncated bool
	failed    bool
}

func (w *contentResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if err != nil {
		w.failed = true
	}
	left := max(0, 1024*1024-w.body.Len())
	w.body.Write(b[:min(n, left)])
	if n > left {
		w.truncated = true
	}
	return n, err
}
func (w *contentResponseWriter) WriteString(s string) (int, error) { return w.Write([]byte(s)) }

// Preserve ResponseController error propagation used by streamed relays.
func (w *contentResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *contentResponseWriter) Flush()                      { _ = w.FlushError() }
func (w *contentResponseWriter) FlushError() error {
	err := http.NewResponseController(w.ResponseWriter).Flush()
	if err != nil {
		w.failed = true
	}
	return err
}

type capturedMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// Only visible text is retained; binary, encrypted reasoning and attachment URLs are omitted.
func visibleText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if text := visibleText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		if text, ok := v["text"].(string); ok {
			return text
		}
		if text, ok := v["refusal"].(string); ok {
			return text
		}
		kind, _ := v["type"].(string)
		switch kind {
		case "image", "image_url", "input_image", "input_file", "file", "input_audio":
			return "[attachment omitted]"
		case "tool_result", "function_call_output":
			return visibleText(v["content"]) + visibleText(v["output"])
		case "tool_use", "function_call":
			name, _ := v["name"].(string)
			return "[tool: " + name + "]"
		}
		if v["inlineData"] != nil || v["fileData"] != nil {
			return "[attachment omitted]"
		}
		if v["functionCall"] != nil {
			return "[tool call]"
		}
		if v["functionResponse"] != nil {
			return "[tool result]"
		}
	}
	return ""
}

func contentMessages(request map[string]any) []capturedMessage {
	result := []capturedMessage{}
	for _, key := range []string{"instructions", "system", "systemInstruction"} {
		value := request[key]
		if m, ok := value.(map[string]any); ok {
			value = m["parts"]
		}
		if text := visibleText(value); text != "" {
			result = append(result, capturedMessage{"system", text})
		}
	}
	var items []any
	for _, key := range []string{"messages", "input", "contents", "output", "content"} {
		if value, ok := request[key].(string); ok {
			result = append(result, capturedMessage{"user", value})
			return result
		}
		if value, ok := request[key].([]any); ok {
			items = value
			break
		}
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		kind, _ := m["type"].(string)
		if role == "model" {
			role = "assistant"
		}
		if kind == "function_call_output" || kind == "tool_result" {
			role = "tool"
		}
		if kind == "function_call" || kind == "tool_use" {
			role = "assistant"
		}
		if role == "" {
			role = "assistant"
		}
		text := visibleText(m["content"])
		if m["parts"] != nil {
			text = visibleText(m["parts"])
		}
		if text == "" {
			text = visibleText(m)
		}
		if m["tool_calls"] != nil {
			text += "\n[tool call]"
		}
		// Claude tool results use the user role on the wire, but are not a new question.
		if blocks, ok := m["content"].([]any); ok && len(blocks) > 0 {
			if last, ok := blocks[len(blocks)-1].(map[string]any); ok && last["type"] == "tool_result" {
				role = "tool"
			}
		}
		if text != "" {
			result = append(result, capturedMessage{role, text})
		}
	}
	return result
}

func boundedContent(messages []capturedMessage) (string, bool) {
	// JSON-escaped length must fit MySQL TEXT as well as SQLite/PostgreSQL.
	result := make([]capturedMessage, 0, min(len(messages), 200))
	used := 2
	clipped := false
	for _, message := range messages {
		if len(result) == 200 {
			clipped = true
			break
		}
		runes := []rune(strings.ToValidUTF8(message.Text, "�"))
		if len(runes) > 8000 {
			runes = runes[:8000]
			clipped = true
		}
		message.Text = string(runes)
		encoded, err := common.Marshal(message)
		if err != nil {
			clipped = true
			continue
		}
		if used+len(encoded)+1 > 48*1024 {
			clipped = true
			break
		}
		result = append(result, message)
		used += len(encoded) + 1
	}
	data, _ := common.Marshal(result)
	return string(data), clipped
}

func responseContent(body []byte, stream bool) ([]capturedMessage, bool) {
	result := []capturedMessage{}
	complete := false
	frames := [][]byte{body}
	if stream {
		frames = bytes.Split(body, []byte("\n"))
	}
	for _, frame := range frames {
		if stream {
			if !bytes.HasPrefix(frame, []byte("data:")) {
				continue
			}
			frame = bytes.TrimSpace(bytes.TrimPrefix(frame, []byte("data:")))
			if string(frame) == "[DONE]" {
				complete = true
				continue
			}
		}
		var m map[string]any
		if common.Unmarshal(frame, &m) != nil {
			continue
		}
		kind, _ := m["type"].(string)
		if kind == "response.completed" {
			if response, ok := m["response"].(map[string]any); ok {
				return contentMessages(response), true
			}
		}
		if kind == "response.failed" || kind == "response.incomplete" || kind == "error" {
			return result, false
		}
		if kind == "message_stop" {
			complete = true
		}
		text := ""
		if kind == "content_block_start" {
			text = visibleText(m["content_block"])
		}
		if kind == "response.output_item.added" {
			text = visibleText(m["item"])
		}
		if kind == "response.output_text.delta" {
			text, _ = m["delta"].(string)
		}
		if delta, ok := m["delta"].(map[string]any); ok {
			text = visibleText(delta)
		}
		if choices, ok := m["choices"].([]any); ok {
			for _, choice := range choices {
				c, ok := choice.(map[string]any)
				if !ok {
					continue
				}
				if delta, ok := c["delta"].(map[string]any); ok {
					text += visibleText(delta["content"])
					if calls, ok := delta["tool_calls"].([]any); ok {
						for _, call := range calls {
							item, ok := call.(map[string]any)
							if !ok {
								continue
							}
							fn, ok := item["function"].(map[string]any)
							if !ok {
								continue
							}
							if name, ok := fn["name"].(string); ok {
								text += "[tool: " + name + "]"
							}
						}
					}
				}
				if message, ok := c["message"].(map[string]any); ok {
					text += visibleText(message["content"])
					if message["tool_calls"] != nil {
						text += "[tool call]"
					}
				}
			}
		}
		if candidates, ok := m["candidates"].([]any); ok {
			for _, candidate := range candidates {
				c, ok := candidate.(map[string]any)
				if !ok {
					continue
				}
				if content, ok := c["content"].(map[string]any); ok {
					text += visibleText(content["parts"])
				}
				if c["finishReason"] != nil {
					complete = true
				}
			}
		}
		if !stream {
			if text == "" {
				result = contentMessages(m)
			}
			complete = m["error"] == nil && (m["choices"] != nil || m["candidates"] != nil || m["content"] != nil || m["status"] == "completed")
		}
		if text != "" {
			if len(result) == 0 {
				result = append(result, capturedMessage{"assistant", text})
			} else {
				result[len(result)-1].Text += text
			}
		}
	}
	return result, complete
}
