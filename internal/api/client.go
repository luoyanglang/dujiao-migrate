package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client API 客户端
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	retryTimes int
	retryDelay time.Duration
}

// NewClient 创建 API 客户端
func NewClient(baseURL string, retryTimes int, retryDelay int) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryTimes: retryTimes,
		retryDelay: time.Duration(retryDelay) * time.Second,
	}
}

// Response API 响应
type Response struct {
	StatusCode int         `json:"status_code"`
	Msg        string      `json:"msg"`
	Data       interface{} `json:"data"`
}

// Login 登录获取 Token
func (c *Client) Login(username, password string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
	}

	resp, err := c.post("/login", payload, false)
	if err != nil {
		return fmt.Errorf("登录请求失败: %w", err)
	}

	if resp.StatusCode != 0 {
		return fmt.Errorf("登录失败: %s", resp.Msg)
	}

	// 解析 token
	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("登录响应数据格式错误")
	}

	if token, ok := dataMap["token"].(string); ok {
		c.token = token
		return nil
	}

	return fmt.Errorf("登录响应中未找到 token")
}

// Post 发送 POST 请求
func (c *Client) Post(endpoint string, payload interface{}) (*Response, error) {
	return c.post(endpoint, payload, true)
}

// Get 发送 GET 请求
func (c *Client) Get(endpoint string) (*Response, error) {
	return c.get(endpoint)
}

func (c *Client) post(endpoint string, payload interface{}, withAuth bool) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt < c.retryTimes; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay)
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("序列化请求数据失败: %w", err)
		}

		req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewBuffer(data))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		if withAuth && c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		var result Response
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = err
			continue
		}

		return &result, nil
	}

	return nil, fmt.Errorf("请求失败，已达最大重试次数: %w", lastErr)
}

func (c *Client) get(endpoint string) (*Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UploadFile 上传文件
func (c *Client) UploadFile(filePath string) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt < c.retryTimes; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay)
		}

		// 打开文件
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("打开文件失败: %w", err)
		}

		// 创建 multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			file.Close()
			lastErr = err
			continue
		}

		if _, err := io.Copy(part, file); err != nil {
			file.Close()
			lastErr = err
			continue
		}

		// 添加 scene 字段
		writer.WriteField("scene", "goods")
		writer.Close()
		file.Close()

		req, err := http.NewRequest("POST", c.baseURL+"/upload", body)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		var result Response
		if err := json.Unmarshal(respBody, &result); err != nil {
			lastErr = err
			continue
		}

		return &result, nil
	}

	return nil, fmt.Errorf("上传失败，已达最大重试次数: %w", lastErr)
}
