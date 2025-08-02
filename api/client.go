package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"GPT/config"
)

type ResponseData struct {
	Code int `json:"code"`
	Data struct {
		List []struct {
			CarID  string `json:"carID"`
			Status int    `json:"status"`
		} `json:"list"`
	} `json:"data"`
}

type CarClient struct {
	config  *config.Config
	headers map[string]string
}

func NewCarClient(cfg *config.Config) *CarClient {
	return &CarClient{
		config: cfg,
		headers: map[string]string{
			"Content-Type": cfg.Headers.ContentType,
			"Accept":       cfg.Headers.Accept,
			"Origin":       cfg.Headers.Origin,
			"Referer":      cfg.Headers.Referer,
			"User-Agent":   cfg.Headers.UserAgent,
		},
	}
}

func (c *CarClient) GetActiveCars() (*ResponseData, error) {
	url := c.config.API.BaseURL + c.config.API.CarPage

	data := map[string]interface{}{
		"page":  1,
		"size":  50,
		"sort":  "desc",
		"order": "sort",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling data: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %v", err)
	}

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	var responseData ResponseData
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("Error decoding response: %v", err)
	}

	return &responseData, nil
}

func (c *CarClient) GetRandomActiveCar() (string, error) {
	responseData, err := c.GetActiveCars()
	if err != nil {
		return "", err
	}

	var activeCars []string
	for _, car := range responseData.Data.List {
		if car.Status == 1 {
			activeCars = append(activeCars, car.CarID)
		}
	}

	if len(activeCars) == 0 {
		return "", fmt.Errorf("没有找到符合条件的车队")
	}

	rand.Seed(time.Now().UnixNano())
	randomCar := activeCars[rand.Intn(len(activeCars))]
	return randomCar, nil
}

func (c *CarClient) GetLoginURL(carID string) string {
	return c.config.API.BaseURL + c.config.API.LoginPage + "?carid=" + carID
}

func (c *CarClient) GetListURL() string {
	return c.config.API.BaseURL + c.config.API.ListPage
}
