package browser

import (
	"GPT/config"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Data struct {
	P string      `json:"p"`
	O string      `json:"o"`
	V interface{} `json:"v"`
}

func SendMessage(message string) string {
	// 加载配置
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		return "加载配置失败"
	}

	// 发送聊天请求
	response, err := sendChatRequest(cfg, message)
	if err != nil {
		return "发送请求失败"
	}

	// 处理响应
	processedContent := processResponse(response)
	reply := extractReplyContent(processedContent)
	updateConversationIDs(cfg, processedContent)

	return reply
}

func sendChatRequest(cfg *config.Config, message string) (string, error) {
	url := cfg.API.BaseURL + "/backend-api/conversation"

	requestBody := map[string]interface{}{
		"action": "next",
		"messages": []map[string]interface{}{
			{
				"author": map[string]string{
					"role": "user",
				},
				"content": map[string]interface{}{
					"content_type": "text",
					"parts":        []string{message},
				},
			},
		},
		"model": "auto",
	}

	if cfg.ConversationID != "" {
		requestBody["conversation_id"] = cfg.ConversationID
	}
	if cfg.ParentMessageID != "" {
		requestBody["parent_message_id"] = cfg.ParentMessageID
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	headers := map[string]string{
		"Accept":        "text/event-stream",
		"Authorization": "Bearer " + cfg.Browser.UserToken,
		"Content-Type":  "application/json",
		"Cookie":        cfg.Browser.Cookie,
		"Origin":        cfg.API.BaseURL,
		"Referer":       cfg.API.BaseURL + "/",
		"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	}

	for key, value := range cfg.Headers {
		headers[key] = value
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	return string(body), nil
}

func updateConversationIDs(cfg *config.Config, processedContent string) {
	conversationID, messageID := extractConversationAndMessageID(processedContent)
	if conversationID == "" || messageID == "" {
		return
	}

	cfg.ConversationID = conversationID
	cfg.ParentMessageID = messageID

	_ = config.SaveConfig("config.json", cfg)
}

func processResponse(response string) string {
	var result strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(response))
	var current *Data

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") ||
			line == `data: "v1"` ||
			line == `data: [DONE]` ||
			strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			var d Data
			jsonStr := strings.TrimPrefix(line, "data: ")
			if err := json.Unmarshal([]byte(jsonStr), &d); err != nil {
				result.WriteString(line + "\n")
				continue
			}

			switch {
			case d.P != "" && d.O != "":
				if current != nil {
					writeData(&result, current)
				}
				if vs, ok := d.V.(string); ok {
					current = &Data{P: d.P, O: d.O, V: vs}
				} else {
					result.WriteString(line + "\n")
					current = nil
				}

			case d.V != nil && d.P == "" && d.O == "":
				if current != nil {
					if vs, ok := d.V.(string); ok {
						current.V = current.V.(string) + vs
					} else {
						writeData(&result, current)
						result.WriteString(line + "\n")
						current = nil
					}
				} else {
					result.WriteString(line + "\n")
				}

			default:
				if current != nil {
					writeData(&result, current)
					current = nil
				}
				result.WriteString(line + "\n")
			}
		} else {
			result.WriteString(line + "\n")
		}
	}

	if current != nil {
		writeData(&result, current)
	}

	return result.String()
}

func writeData(writer *strings.Builder, data *Data) {
	jsonData, _ := json.Marshal(data)
	writer.WriteString(fmt.Sprintf("data:%s\n", jsonData))
}

func extractReplyContent(content string) string {
	var replyContent string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			var data Data
			if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
				continue
			}
			if data.P == "/message/content/parts/0" && data.O == "append" {
				if text, ok := data.V.(string); ok {
					replyContent += text
				}
			}
		}
	}

	return strings.TrimSpace(replyContent)
}

func extractConversationAndMessageID(content string) (string, string) {
	var conversationID, messageID string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data:") {
			var data map[string]interface{}
			jsonStr := strings.TrimPrefix(line, "data: ")
			if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
				continue
			}

			if v, ok := data["v"].(map[string]interface{}); ok {
				if msg, ok := v["message"].(map[string]interface{}); ok {
					if author, ok := msg["author"].(map[string]interface{}); ok {
						if role, ok := author["role"].(string); ok && role == "assistant" {
							if id, ok := msg["id"].(string); ok {
								messageID = id
							}
						}
					}
				}
				if convID, ok := v["conversation_id"].(string); ok {
					conversationID = convID
				}
			}
		}
	}

	return conversationID, messageID
}
