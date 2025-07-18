package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SlackNotifier struct {
	webhookURL string
	enabled    bool
}

type SlackMessage struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color  string  `json:"color"`
	Fields []Field `json:"fields"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewSlackNotifier(webhookURL string, enabled bool) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		enabled:    enabled,
	}
}

func (s *SlackNotifier) NotifyError(errorType, details string) error {
	if !s.enabled || s.webhookURL == "" {
		return nil
	}

	message := SlackMessage{
		Text: "Pi-hole同期エラーが発生しました",
		Attachments: []Attachment{
			{
				Color: "danger",
				Fields: []Field{
					{
						Title: "エラー種別",
						Value: errorType,
						Short: true,
					},
					{
						Title: "発生時刻",
						Value: time.Now().Format("2006-01-02 15:04:05"),
						Short: true,
					},
					{
						Title: "詳細",
						Value: details,
						Short: false,
					},
				},
			},
		},
	}

	return s.sendMessage(message)
}

func (s *SlackNotifier) sendMessage(message SlackMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}
