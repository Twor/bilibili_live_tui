package login

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"golang.org/x/net/publicsuffix"
)

// QrCodeLoginInfo 二维码登录信息
type QrCodeLoginInfo struct {
	QrCodeURL string
	QrCodeKey string
}

// LoginResult 登录结果
type LoginResult struct {
	Cookies      map[string]string
	ExpiresIn    int64 // 过期时间（秒）
	Success      bool
	Message      string
	RefreshToken string // refresh_token for cookie refresh
}

// LoginManager 登录管理器
type LoginManager struct {
	client *http.Client
}

// NewLoginManager 创建登录管理器
func NewLoginManager() *LoginManager {
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	return &LoginManager{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

// GetQrCode 获取登录二维码
func (l *LoginManager) GetQrCode() (*QrCodeLoginInfo, error) {
	apiURL := "https://passport.bilibili.com/x/passport-login/web/qrcode/generate"

	resp, err := l.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取二维码失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			Url   string `json:"url"`
			QrKey string `json:"qrcode_key"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("获取二维码失败: %s", result.Message)
	}

	return &QrCodeLoginInfo{
		QrCodeURL: result.Data.Url,
		QrCodeKey: result.Data.QrKey,
	}, nil
}

// CheckQrCodeStatus 检查二维码扫描状态
func (l *LoginManager) CheckQrCodeStatus(qrKey string) (*LoginResult, error) {
	apiURL := fmt.Sprintf("https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=%s", qrKey)

	resp, err := l.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("检查二维码状态失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code    int `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Code         int    `json:"code"`
			Message      string `json:"message"`
			Url          string `json:"url"`
			RefreshToken string `json:"refresh_token"`
			Timestamp    int64  `json:"timestamp"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	loginResult := &LoginResult{
		Success: false,
	}

	// 状态码说明：
	// 0: 成功
	// 86038: 二维码已过期
	// 86090: 已扫码未确认
	// 86101: 未扫码
	switch result.Data.Code {
	case 0:
		// 登录成功，访问返回的URL获取Cookie
		loginResult.Success = true
		loginResult.Message = "登录成功"
		loginResult.RefreshToken = result.Data.RefreshToken
		loginResult.ExpiresIn = 30 * 24 * 60 * 60 // 30天

		// 访问返回的URL以设置Cookie
		if result.Data.Url != "" {
			cookies, err := l.getCookiesFromRedirect(result.Data.Url)
			if err != nil {
				return nil, err
			}
			loginResult.Cookies = cookies
		}
	case 86038:
		loginResult.Message = "二维码已过期，请重新获取"
	case 86090:
		loginResult.Message = "已扫码，请确认登录"
	case 86101:
		loginResult.Message = "等待扫码..."
	default:
		loginResult.Message = fmt.Sprintf("未知状态: %d, %s", result.Data.Code, result.Data.Message)
	}

	return loginResult, nil
}

// getCookiesFromRedirect 访问重定向URL获取Cookie
func (l *LoginManager) getCookiesFromRedirect(redirectURL string) (map[string]string, error) {
	cookies := make(map[string]string)

	// 解析URL获取查询参数中的Cookie信息
	u, err := url.Parse(redirectURL)
	if err != nil {
		return cookies, fmt.Errorf("解析URL失败: %w", err)
	}

	// 从查询参数中提取关键信息
	query := u.Query()
	if dedeUserID := query.Get("DedeUserID"); dedeUserID != "" {
		cookies["DedeUserID"] = dedeUserID
	}

	// 访问该URL以获取完整Cookie
	resp, err := l.client.Get(redirectURL)
	if err != nil {
		return cookies, fmt.Errorf("访问重定向URL失败: %w", err)
	}
	defer resp.Body.Close()

	// 从Cookie Jar中获取Cookie
	biliURL, _ := url.Parse("https://bilibili.com")
	for _, cookie := range l.client.Jar.Cookies(biliURL) {
		cookies[cookie.Name] = cookie.Value
	}

	apiURL, _ := url.Parse("https://api.bilibili.com")
	for _, cookie := range l.client.Jar.Cookies(apiURL) {
		cookies[cookie.Name] = cookie.Value
	}

	// 从响应头获取Set-Cookie
	for _, c := range resp.Cookies() {
		cookies[c.Name] = c.Value
	}

	return cookies, nil
}

// PrintQrCode 在终端打印二维码
func PrintQrCode(content string) error {
	// 生成小尺寸二维码字符串，直接在终端显示
	qr, err := qrcode.New(content, qrcode.Low)
	if err != nil {
		return fmt.Errorf("生成二维码失败: %w", err)
	}

	// 禁用边框，减小尺寸
	qr.DisableBorder = true

	// 获取二维码矩阵
	matrix := qr.Bitmap()

	// 使用紧凑格式打印二维码（每行两个像素合并为一个字符）
	var sb strings.Builder
	sb.WriteString("\n请使用B站APP扫描二维码登录:\n\n")

	for y := 0; y < len(matrix); y += 2 {
		for x := 0; x < len(matrix[y]); x++ {
			// 使用半块字符来表示两个像素
			top := matrix[y][x]
			bottom := false
			if y+1 < len(matrix) {
				bottom = matrix[y+1][x]
			}

			var char string
			if top && bottom {
				char = "█" // 上下都是黑色
			} else if top && !bottom {
				char = "▀" // 只有上面是黑色
			} else if !top && bottom {
				char = "▄" // 只有下面是黑色
			} else {
				char = " " // 都是白色
			}
			sb.WriteString(char)
		}
		sb.WriteString("\n")
	}

	fmt.Println(sb.String())
	fmt.Println("或访问以下链接扫码:")
	fmt.Println(content)
	fmt.Println()

	return nil
}

// WaitForLogin 等待用户扫码登录
func (l *LoginManager) WaitForLogin(qrInfo *QrCodeLoginInfo, timeout time.Duration) (*LoginResult, error) {
	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	fmt.Println("等待扫码登录...")

	for {
		select {
		case <-ticker.C:
			if time.Since(start) > timeout {
				return nil, fmt.Errorf("登录超时")
			}

			result, err := l.CheckQrCodeStatus(qrInfo.QrCodeKey)
			if err != nil {
				continue
			}

			fmt.Printf("\r[%s] %s                    ", time.Now().Format("15:04:05"), result.Message)

			if result.Success {
				fmt.Println("\n登录成功！")
				return result, nil
			}

			// 二维码过期，需要重新获取
			if result.Message == "二维码已过期，请重新获取" {
				return nil, fmt.Errorf("二维码已过期")
			}
		}
	}
}

// DoLogin 执行完整的扫码登录流程
func DoLogin() (map[string]string, error) {
	manager := NewLoginManager()

	// 获取二维码
	qrInfo, err := manager.GetQrCode()
	if err != nil {
		return nil, err
	}

	// 打印二维码
	if err := PrintQrCode(qrInfo.QrCodeURL); err != nil {
		return nil, err
	}

	// 等待登录，超时3分钟
	result, err := manager.WaitForLogin(qrInfo, 3*time.Minute)
	if err != nil {
		return nil, err
	}

	return result.Cookies, nil
}

// RefreshCookie 刷新Cookie有效期
func RefreshCookie(cookieStr string) error {
	// 通过访问nav接口来刷新Cookie有效期
	// B站会在每次请求时自动刷新Cookie有效期
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	// 解析并设置Cookie
	cookies := parseCookieString(cookieStr)
	biliURL, _ := url.Parse("https://api.bilibili.com")
	jar.SetCookies(biliURL, cookies)

	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/nav", nil)
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("刷新Cookie失败: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// parseCookieString 解析Cookie字符串为Cookie列表
func parseCookieString(cookieStr string) []*http.Cookie {
	var cookies []*http.Cookie
	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			cookies = append(cookies, &http.Cookie{
				Name:  strings.TrimSpace(kv[0]),
				Value: strings.TrimSpace(kv[1]),
			})
		}
	}
	return cookies
}

// CookiesToString 将Cookie map转换为字符串
func CookiesToString(cookies map[string]string) string {
	var parts []string
	for k, v := range cookies {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, "; ")
}

// VerifyCookie 验证Cookie是否有效
func VerifyCookie(cookieStr string) (bool, int64, error) {
	jar, _ := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/nav", nil)
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

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, 0, err
	}

	return result.Data.IsLogin, result.Data.Mid, nil
}