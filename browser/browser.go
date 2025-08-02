package browser

import (
	"GPT/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type BrowserConfig struct {
	Browser struct {
		RemoteDebugURL string `json:"remote_debug_url"`
		TargetURL      string `json:"target_url"`
		Cookie         string `json:"cookie"`
	} `json:"browser"`
	API struct {
		BaseURL   string `json:"base_url"`
		LoginPage string `json:"login_page"`
		ListPage  string `json:"list_page"`
	} `json:"api"`
	DingTalk struct {
		UserToken string `json:"usertoken"`
	} `json:"dingtalk"`
}

type CarResponse struct {
	Code int `json:"code"`
	Data struct {
		List []struct {
			CarID  string `json:"carID"`
			Name   string `json:"name"`
			Status int    `json:"status"`
		} `json:"list"`
		Total int `json:"total"`
	} `json:"data"`
	Message string `json:"message"`
}

func GetCookie() (string, bool) {
	// 加载完整配置
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		fmt.Printf("[错误] 加载配置失败: %v\n", err)
		return "", false
	}

	// 选择车队
	carID, err := selectActiveCar(cfg)
	if err != nil {
		fmt.Printf("[警告] 选择车队失败: %v，使用默认车队ID\n", err)
		carID = getDefaultCarID()
	}

	loginURL := fmt.Sprintf("%s%s?carid=%s", cfg.API.BaseURL, cfg.API.LoginPage, carID)

	// 创建ChromeDP上下文
	ctx, cancel := chromedp.NewRemoteAllocator(context.Background(), cfg.Browser.RemoteDebugURL)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	// 设置超时
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 执行浏览器操作
	var cookies string
	err = chromedp.Run(ctx,
		// 导航到登录页面
		chromedp.Navigate(loginURL),
		chromedp.Sleep(5*time.Second),

		// 等待页面加载完成
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),

		// 获取cookie
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Evaluate(`
				(function() {
					var cookies = [];
					var cookieArray = document.cookie.split(';');
					for(var i = 0; i < cookieArray.length; i++) {
						var cookie = cookieArray[i].trim();
						if (cookie) cookies.push(cookie);
					}
					return cookies.join('; ');
				})()
			`, &cookies).Do(ctx)
		}),
	)

	if err != nil {
		fmt.Printf("[错误] 浏览器操作失败: %v\n", err)
		return "", false
	}

	if cookies == "" {
		fmt.Println("[错误] 未能获取到Cookie")
		return "", false
	}

	// 更新配置文件中的cookie
	cfg.Browser.Cookie = cookies
	if err := config.SaveConfig("config.json", cfg); err != nil {
		fmt.Printf("[错误] 保存配置失败: %v\n", err)
		return "", false
	}

	return cookies, true
}

// selectActiveCar 选择活跃的车队
func selectActiveCar(cfg *config.Config) (string, error) {
	// 从配置文件读取完整URL
	url := cfg.API.BaseURL + cfg.API.CarPage

	// 构建请求数据
	requestData := map[string]interface{}{
		"page":  1,
		"size":  50,
		"sort":  "desc",
		"order": "sort",
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("序列化请求数据失败: %w", err)
	}

	// 创建POST请求
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头 - 从配置文件读取Origin
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", cfg.API.BaseURL)
	req.Header.Set("Referer", cfg.API.BaseURL+"/list")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求车队列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	var responseData CarResponse
	if err := json.Unmarshal(body, &responseData); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if responseData.Code != 1000 {
		return "", fmt.Errorf("API返回错误码: %d, 消息: %s", responseData.Code, responseData.Message)
	}

	var activeCars []string
	for _, car := range responseData.Data.List {
		if car.Status == 1 {
			activeCars = append(activeCars, car.CarID)
		}
	}

	if len(activeCars) == 0 {
		return "", fmt.Errorf("没有找到活跃的车队")
	}

	// 随机选择一个活跃车队
	rand.Seed(time.Now().UnixNano())
	selected := activeCars[rand.Intn(len(activeCars))]
	return selected, nil
}

// getDefaultCarID 获取默认车队ID
func getDefaultCarID() string {
	return "1"
}
