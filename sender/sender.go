package sender

import (
	"bili/config"
	"bili/getter"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var csrfToken string
var cookieStr string
var initialized bool

// Run 初始化发送弹幕功能
func Run() {
	// 从Cookie中提取bili_jct作为csrf token
	attrs := strings.Split(config.Config.Cookie, ";")
	kvs := make(map[string]string)
	for _, attr := range attrs {
		kv := strings.SplitN(strings.TrimSpace(attr), "=", 2)
		if len(kv) == 2 {
			kvs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	csrfToken = kvs["bili_jct"]
	cookieStr = config.Config.Cookie

	if csrfToken == "" {
		fmt.Println("Cookie中未找到bili_jct，发送弹幕功能将不可用")
		return
	}

	// 验证Cookie是否有效
	valid, _, err := verifyCookie()
	if err != nil || !valid {
		fmt.Println("Cookie验证失败，发送弹幕功能将不可用")
		return
	}

	initialized = true
	fmt.Println("发送弹幕功能已就绪")
}

// verifyCookie 验证Cookie是否有效
func verifyCookie() (bool, int64, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/nav", nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			IsLogin bool  `json:"isLogin"`
			Mid     int64 `json:"mid"`
		} `json:"data"`
	}

	if err := decodeJSON(resp.Body, &result); err != nil {
		return false, 0, err
	}

	return result.Data.IsLogin, result.Data.Mid, nil
}

// SendMsg 发送弹幕
func SendMsg(roomId int64, msg string, busChan chan getter.DanmuMsg) {
	if !initialized {
		busChan <- getter.DanmuMsg{Author: "system", Content: "发送弹幕失败: 客户端未初始化", Type: "NOTICE_MSG"}
		return
	}

	msgRune := []rune(msg)
	for i := 0; i < len(msgRune); i += 20 {
		chunk := ""
		if i+20 < len(msgRune) {
			chunk = string(msgRune[i : i+20])
		} else {
			chunk = string(msgRune[i:])
		}

		err := sendDanmaku(roomId, 16777215, 25, 1, chunk, 0)
		if err != nil {
			busChan <- getter.DanmuMsg{Author: "system", Content: fmt.Sprintf("发送弹幕失败: %v", err), Type: "NOTICE_MSG"}
		}

		// 如果还有后续分段，等待1秒避免发送过快
		if i+20 < len(msgRune) {
			time.Sleep(time.Second * 1)
		}
	}
}

// sendDanmaku 发送单条弹幕到B站直播间
func sendDanmaku(roomID int64, color int64, fontsize int, mode int, msg string, bubble int) error {
	client := &http.Client{Timeout: 10 * time.Second}

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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", fmt.Sprintf("https://live.bilibili.com/%d", roomID))

	resp, err := client.Do(req)
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
