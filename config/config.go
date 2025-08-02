package config

import (
	"encoding/json"
	"os"
)

type DingTalkConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type BrowserConfig struct {
	UserToken       string `json:"usertoken"`
	Cookie          string `json:"cookie"`
	ConversationID  string `json:"conversation_id"`
	ParentMessageID string `json:"parent_message_id"`
	RemoteDebugURL  string `json:"remote_debug_url"`
	TargetURL       string `json:"target_url"`
}

type APIConfig struct {
	BaseURL   string `json:"base_url"`
	CarPage   string `json:"car_page"`
	LoginPage string `json:"login_page"`
	ListPage  string `json:"list_page"`
}

type HeadersConfig struct {
	ContentType string `json:"content_type"`
	Accept      string `json:"accept"`
	Origin      string `json:"origin"`
	Referer     string `json:"referer"`
	UserAgent   string `json:"user_agent"`
}

type Config struct {
	Browser         BrowserConfig     `json:"browser"`
	API             APIConfig         `json:"api"`
	DingTalk        DingTalkConfig    `json:"dingtalk"`
	Headers         map[string]string `json:"headers"`
	ConversationID  string            `json:"conversation_id"`
	ParentMessageID string            `json:"parent_message_id"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func SaveConfig(filename string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
