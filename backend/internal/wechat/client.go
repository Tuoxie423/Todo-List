// Package wechat 封装微信小程序的登录凭证校验。
package wechat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Session 是 code 换取到的微信会话信息。
type Session struct {
	OpenID string
}

// Exchanger 把小程序登录 code 换成微信会话，测试可注入假实现。
type Exchanger interface {
	ExchangeCode(code string) (Session, error)
}

// Client 通过 jscode2session 接口换取会话。
type Client struct {
	appID      string
	secret     string
	httpClient *http.Client
}

func NewClient(appID, secret string) *Client {
	return &Client{
		appID:      strings.TrimSpace(appID),
		secret:     strings.TrimSpace(secret),
		httpClient: &http.Client{Timeout: 8 * time.Second},
	}
}

func (c *Client) ExchangeCode(code string) (Session, error) {
	if c.appID == "" || c.secret == "" {
		return Session{}, errors.New("wechat appid or secret is not configured")
	}

	query := url.Values{}
	query.Set("appid", c.appID)
	query.Set("secret", c.secret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")

	response, err := c.httpClient.Get("https://api.weixin.qq.com/sns/jscode2session?" + query.Encode())
	if err != nil {
		return Session{}, fmt.Errorf("wechat login request failed: %w", err)
	}
	defer response.Body.Close()

	var body struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return Session{}, fmt.Errorf("decode wechat login response failed: %w", err)
	}
	if body.ErrCode != 0 {
		message := body.ErrMsg
		if message == "" {
			message = "wechat login failed"
		}
		return Session{}, fmt.Errorf("wechat login failed: %s", message)
	}

	return Session{OpenID: strings.TrimSpace(body.OpenID)}, nil
}

var secretPattern = regexp.MustCompile(`(?i)(secret=)[^&\s]+`)

// RedactSecret 隐藏错误信息里的 secret，避免把凭证写进日志。
func RedactSecret(message string) string {
	return secretPattern.ReplaceAllString(message, "${1}REDACTED")
}
