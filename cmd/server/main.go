package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"ai-reading-assistant/internal/config"
	"ai-reading-assistant/internal/dao"
	"ai-reading-assistant/internal/handler"
	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/mongo"
	"ai-reading-assistant/internal/router"
	"ai-reading-assistant/internal/service"
	"ai-reading-assistant/internal/wikipedia"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cfg := config.Global()
	client, err := llm.NewDeepseekClient(cfg.Deepseek.Model, cfg.Deepseek.BaseURL, cfg.Deepseek.APIKey)
	if err != nil {
		log.Fatalf("init deepseek client: %v", err)
	}

	runWikiTest()

	// if err := runDeepseekChatTest(client); err != nil {
	// 	log.Fatalf("deepseek chat test: %v", err)
	// }

	mongoClient, err := mongo.Global(context.Background())
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}

	sessionDao := dao.NewInspirationSessionDao(mongoClient)
	messageDao := dao.NewInspirationMessageDao(mongoClient)
	processDao := dao.NewRequestProcessDao(mongoClient)

	inspirationService := service.NewInspirationService(client, sessionDao, messageDao, processDao)
	inspirationHandler := handler.NewInspirationPlanHandler(inspirationService)

	engine := router.Setup(inspirationHandler)

	log.Printf("travel backend listening on :%s", port)
	if err := engine.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// runDeepseekChatTest gives developers a quick multi-turn chat loop against Deepseek.
func runDeepseekChatTest(client *llm.DeepseekClient) error {
	reader := bufio.NewScanner(os.Stdin)
	fmt.Println("Interactive Deepseek chat started. Type 'exit' to stop.")

	for {
		fmt.Print("\nYou: ")
		if !reader.Scan() {
			if err := reader.Err(); err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			fmt.Println("\nInput closed.")
			return nil
		}

		text := strings.TrimSpace(reader.Text())
		if strings.EqualFold(text, "exit") {
			fmt.Println("Bye!")
			return nil
		}
		if text == "" {
			continue
		}

		messages := []llm.Message{{Role: "user", Content: text}}
		fmt.Print("Assistant: ")
		response, err := client.ChatStream(messages, func(delta string) error {
			fmt.Print(delta)
			return nil
		})
		fmt.Println()
		if err != nil {
			return fmt.Errorf("deepseek chat: %w", err)
		}
		_ = response
	}
}

func runWikiTest() {
	keyword := "福贡县老姆登村"
	// wikiCfg := config.Global().Wikipedia
	// client, err := wikipedia.NewClient(
	// 	wikipedia.WithLanguage(wikiCfg.Language),
	// 	wikipedia.WithUserAgent(wikiCfg.UserAgent),
	// 	wikipedia.WithProxy(wikiCfg.Proxy),
	// )
	client, err := wikipedia.NewClient(
		wikipedia.WithLanguage("zh"),
		wikipedia.WithUserAgent("MyGolangDemo/1.0 (your@email.com)"),
		wikipedia.WithProxy("http://127.0.0.1:6789"),
	)
	if err != nil {
		fmt.Printf("初始化 wikipedia client 失败：%v\n", err)
		return
	}

	summary, err := client.GetSummary(context.Background(), keyword)
	if err != nil {
		fmt.Printf("调用 wikipedia API 失败：%v\n", err)
		return
	}

	fmt.Println("===== 维基百科API调用结果 =====")
	fmt.Printf("关键词：%s\n", summary.Title)
	fmt.Printf("简短释义：%s\n", summary.Extract)
	fmt.Printf("完整页面链接：%s\n", summary.ContentURLs.Desktop.Page)
}

// func runScoreOfTop3Test(svc service.TravelPlanService) {
// 	querys := []string{
// 		"我对文学旅行有点兴趣...能不能给我推荐点有感觉的地方",   // 2+8+0=10
// 		"感觉最近很累,想找个能让人静下来,像书里写的那种地方去走走", // 2+2+0=4
// 		"我想体验一下‘诗意地栖居’,有没有这样的旅行路线",      // 4+2+0=6
// 		"我很喜欢村上春树", // 8+0+0=8
// 		"心情不好的时候,总想逃到一个类似《瓦尔登湖》描述的世界里,但又不知道具体是哪",     //8+8+8=24
// 		"我打算去日本,能不能设计一条有点‘物哀’美学的旅行路线",                // 6+4+6=16
// 		"看了《长安三万里》有点触动,想顺着唐诗去旅行,有没有不太累的走法",           // 8+4+0=12
// 		"我想策划一次‘寻找灵感’的写作之旅,地点最好在欧美,时间一两周吧",           //4+8+4=16
// 		"我是海明威的粉丝,想体验一下他那种‘在路上’的感觉,行程刺激一点",           // 8+2+0=10
// 		"请以《寻路中国》的观察方式为参考,帮我设计一个深入中国乡镇的10天行程,交通工具随意", // 8+4+6=18
// 		"请帮我以三毛的中南美之行为蓝本制定一个15天的行程",                  //
// 		// "我想去圣地巡游",
// 		// "我想去欧洲感受文艺复兴时期的艺术",   // 4+8+4
// 		// "我想去追寻一下书本里汪曾祺故乡的滋味", //
// 	}
// 	for _, query := range querys {
// 		log.Printf("ScoreOfTop3 query: %s", query)
// 		fields := svc.ScoreOfTop3(query)
// 		if len(fields) == 0 {
// 			log.Printf("ScoreOfTop3 returned no data")
// 			return
// 		}

// 		keys := make([]string, 0, len(fields))
// 		for k := range fields {
// 			keys = append(keys, k)
// 		}
// 		sort.Strings(keys)

// 		for _, key := range keys {
// 			field := fields[key]
// 			log.Printf("  %s => score=%d content=%q", field.Field, field.Score, field.Content)
// 		}
// 	}
// }
