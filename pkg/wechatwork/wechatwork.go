// wechatwork/wechatwork.go
package wechatwork

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// WeChatWorkMessage 企业微信Webhook消息结构
type WeChatWorkMessage struct {
	MsgType  string          `json:"msgtype"`
	Text     TextContent     `json:"text,omitempty"`
	Markdown MarkdownContent `json:"markdown,omitempty"`
}

type TextContent struct {
	Content             string   `json:"content"`
	MentionedList       []string `json:"mentioned_list,omitempty"`
	MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
}

type MarkdownContent struct {
	Content string `json:"content"`
}

// NotificationSender 通知发送器
type NotificationSender struct {
	WebhookKey string
	Enabled    bool
}

// NewNotificationSender 创建新的通知发送器
func NewNotificationSender(webhookKey string) *NotificationSender {
	enabled := webhookKey != ""
	return &NotificationSender{
		WebhookKey: webhookKey,
		Enabled:    enabled,
	}
}

// SendText 发送文本消息
func (ns *NotificationSender) SendText(content string, mentionedList, mentionedMobileList []string) error {
	if !ns.Enabled {
		return nil
	}

	message := WeChatWorkMessage{
		MsgType: "text",
		Text: TextContent{
			Content:             content,
			MentionedList:       mentionedList,
			MentionedMobileList: mentionedMobileList,
		},
	}

	return ns.send(message)
}

// SendMarkdown 发送Markdown消息
func (ns *NotificationSender) SendMarkdown(content string) error {
	if !ns.Enabled {
		return nil
	}

	message := WeChatWorkMessage{
		MsgType: "markdown",
		Markdown: MarkdownContent{
			Content: content,
		},
	}

	return ns.send(message)
}

// send 发送消息
func (ns *NotificationSender) send(message WeChatWorkMessage) error {
	// 构建Webhook URL
	webhookURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s", ns.WebhookKey)

	// 将消息转换为JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("无法序列化通知消息: %w", err)
	}

	// 发送HTTP POST请求
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送企业微信通知失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("企业微信通知返回非200状态码: %d", resp.StatusCode)
	}

	log.Println("企业微信通知发送成功")
	return nil
}
