package sender

import (
	"bili/config"
	"bili/getter"
	"bili/login"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultColor    = 16777215 // 0xFFFFFF 白色
	defaultFontSize = 25
	defaultMode     = 1 // 弹幕模式：滚动
	defaultBubble   = 0
	chunkSize       = 20 // 弹幕分段发送长度
	sendInterval    = 1 * time.Second
	userAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var (
	csrfToken   string
	cookieStr   string
	initialized bool
	httpClient  = &http.Client{Timeout: 10 * time.Second}
)

// Run 初始化发送弹幕功能
func Run() {
	cookieStr = config.Config.Cookie
	csrfToken = extractCSRFToken(cookieStr)

	if csrfToken == "" {
		fmt.Println("Cookie中未找到bili_jct，发送弹幕功能将不可用")
		return
	}

	// 复用 login 包的 VerifyCookie 验证 Cookie
	valid, _, err := login.VerifyCookie(cookieStr)
	if err != nil || !valid {
		fmt.Println("Cookie验证失败，发送弹幕功能将不可用")
		return
	}

	initialized = true
	fmt.Println("发送弹幕功能已就绪")
}

// extractCSRFToken 从Cookie字符串中提取bili_jct
func extractCSRFToken(cookie string) string {
	attrs := strings.Split(cookie, ";")
	for _, attr := range attrs {
		kv := strings.SplitN(strings.TrimSpace(attr), "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "bili_jct" {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

// SendMsg 发送弹幕（自动分段）
func SendMsg(roomId int64, msg string, busChan chan getter.DanmuMsg) {
	if !initialized {
		busChan <- getter.DanmuMsg{
			Author:  "system",
			Content: "发送弹幕失败: 客户端未初始化",
			Type:    "NOTICE_MSG",
		}
		return
	}

	msgRunes := []rune(msg)
	for i := 0; i < len(msgRunes); i += chunkSize {
		end := i + chunkSize
		if end > len(msgRunes) {
			end = len(msgRunes)
		}
		chunk := string(msgRunes[i:end])

		if err := sendDanmaku(roomId, defaultColor, defaultFontSize, defaultMode, chunk, defaultBubble); err != nil {
			busChan <- getter.DanmuMsg{
				Author:  "system",
				Content: fmt.Sprintf("发送弹幕失败: %v", err),
				Type:    "NOTICE_MSG",
			}
		}

		// 分段发送间隔
		if i+chunkSize < len(msgRunes) {
			time.Sleep(sendInterval)
		}
	}
}

// sendDanmaku 发送单条弹幕到B站直播间
func sendDanmaku(roomID int64, color int64, fontsize int, mode int, msg string, bubble int) error {
	form := url.Values{}
	form.Set("roomid", strconv.FormatInt(roomID, 10))
	form.Set("color", strconv.FormatInt(color, 10))
	form.Set("fontsize", strconv.Itoa(fontsize))
	form.Set("mode", strconv.Itoa(mode))
	form.Set("msg", msg)
	form.Set("bubble", strconv.Itoa(bubble))
	form.Set("rnd", strconv.FormatInt(time.Now().Unix(), 10))
	form.Set("csrf", csrfToken)
	form.Set("csrf_token", csrfToken)

	req, err := http.NewRequest("POST", "https://api.live.bilibili.com/msg/send", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", fmt.Sprintf("https://live.bilibili.com/%d", roomID))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := decodeJSON(resp.Body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("B站返回错误: code=%d, message=%s", result.Code, result.Message)
	}
	return nil
}

// decodeJSON 解码JSON响应体
func decodeJSON(body io.Reader, v interface{}) error {
	return json.NewDecoder(body).Decode(v)
}
