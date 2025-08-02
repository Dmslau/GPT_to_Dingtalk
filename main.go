package main

import (
	"GPT/browser"
	"GPT/config"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/logger"
)

var globalConfig *config.Config

func OnChatBotMessageReceived(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
	userMessage := strings.TrimSpace(data.Text.Content)
	log.Println("Received DingTalk message:", userMessage)

	// 收到消息处理
	cookie, success := browser.GetCookie()
	if success {
		fmt.Println("Cookie获取成功:", cookie)

		// 调用chat发送消息
		reply := browser.SendMessage(userMessage)
		fmt.Println("收到回复:", reply)

		// 回复钉钉
		replyMsg := []byte(fmt.Sprintf("%s", reply))
		replier := chatbot.NewChatbotReplier()
		if err := replier.SimpleReplyText(ctx, data.SessionWebhook, replyMsg); err != nil {
			return nil, err
		}

	} else {
		fmt.Println("Cookie获取失败")
		// 回复钉钉错误信息
		replyMsg := []byte("Cookie获取失败")
		replier := chatbot.NewChatbotReplier()
		if err := replier.SimpleReplyText(ctx, data.SessionWebhook, replyMsg); err != nil {
			return nil, err
		}
	}

	return []byte(""), nil
}

func main() {
	logger.SetLogger(logger.NewStdTestLogger())

	// 加载配置
	var err error
	globalConfig, err = config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("无法加载配置文件: %v", err)
	}

	// 启动钉钉客户端
	cli := client.NewStreamClient(
		client.WithAppCredential(
			client.NewAppCredentialConfig(globalConfig.DingTalk.ClientID, globalConfig.DingTalk.ClientSecret),
		),
	)
	cli.RegisterChatBotCallbackRouter(OnChatBotMessageReceived)

	err = cli.Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	log.Println("钉钉客户端启动成功，正在等待消息...")
	select {}
}
