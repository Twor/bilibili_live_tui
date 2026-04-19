package login

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/tidwall/gjson"
)

// LoginResult 登录结果
type LoginResult struct {
	Cookie       string
	Csrf         string
	RefreshToken string
}

// printSmallQR 打印缩小版的二维码
func printSmallQR(content string) {
	// 强制使用低容错率以减小尺寸
	qr, err := qrcode.New(content, qrcode.Low)
	if err != nil {
		fmt.Println("生成二维码失败:", err)
		return
	}
	bitmap := qr.Bitmap()
	size := len(bitmap)

	// 建议保留 2 个单位的边框（Quiet Zone）
	border := 2

	// 纵向步长为 2，因为一个字符代表两行像素
	for y := border; y < size-border; y += 2 {
		for x := border; x < size-border; x++ {
			top := bitmap[y][x]

			// 检查下一行是否越界（防止总行数为奇数）
			bottom := false
			if y+1 < size-border {
				bottom = bitmap[y+1][x]
			}

			// 根据上下两行像素的状态选择字符
			if top && bottom {
				fmt.Print("█") // 上下皆黑
			} else if top {
				fmt.Print("▀") // 上黑下白
			} else if bottom {
				fmt.Print("▄") // 上白下黑
			} else {
				fmt.Print(" ") // 上白下白
			}
		}
		fmt.Println() // 换行
	}
}

const UserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/97.0.4692.99 Safari/537.36 Edg/97.0.1072.69`

// getLoginKeyAndLoginUrl 获取二维码内容和密钥
func getLoginKeyAndLoginUrl() (string, string) {
	url := "https://passport.bilibili.com/x/passport-login/web/qrcode/generate"
	client := http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	data := gjson.ParseBytes(body)
	loginKey := data.Get("data.qrcode_key").String()
	loginUrl := data.Get("data.url").String()
	return loginKey, loginUrl
}

// getQRCodeState 获取二维码状态，返回cookie和refreshToken
func getQRCodeState(loginKey string) (string, string, error) {
	for {
		apiUrl := "https://passport.bilibili.com/x/passport-login/web/qrcode/poll"
		client := http.Client{}
		req, _ := http.NewRequest("GET", apiUrl+fmt.Sprintf("?qrcode_key=%s", loginKey), nil)
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		if err != nil {
			return "", "", err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		data := gjson.ParseBytes(body)
		switch data.Get("data.code").Int() {
		case 0:
			cookieUrl := data.Get("data.url").String()
			parsedUrl, err := url.Parse(cookieUrl)
			if err != nil {
				return "", "", err
			}
			cookieContentList := strings.Split(parsedUrl.RawQuery, "&")
			cookieContent := ""
			for _, cookie := range cookieContentList[:len(cookieContentList)-1] {
				cookieContent = cookieContent + cookie + ";"
			}
			cookieContent = strings.TrimSuffix(cookieContent, ";")
			refreshToken := data.Get("data.refresh_token").String()
			fmt.Println("扫码成功")
			return cookieContent, refreshToken, nil
		case 86038:
			fmt.Println("二维码已失效，正在重新生成")
			return "", "", fmt.Errorf("二维码失效")
		case 86090:
			fmt.Println("已扫码，请确认")
		case 86101:
		default:
			return "", "", fmt.Errorf("未知code: %d", data.Get("data.code").Int())
		}
		time.Sleep(time.Second * 3)
	}
}

// DoLogin 执行扫码登录，自定义二维码显示
func DoLogin() (*LoginResult, error) {
	for {
		fmt.Println("未登录或cookie已过期，请扫码登录")
		loginKey, loginUrl := getLoginKeyAndLoginUrl()
		fmt.Println("请使用手机Bilibili扫描下方二维码:")
		printSmallQR(loginUrl)
		cookie, refreshToken, err := getQRCodeState(loginKey)
		if err != nil {
			fmt.Println(err)
			continue
		}
		// 提取csrf
		reg := regexp.MustCompile(`bili_jct=([0-9a-zA-Z]+)`)
		csrf := reg.FindStringSubmatch(cookie)[1]
		return &LoginResult{
			Cookie:       cookie,
			Csrf:         csrf,
			RefreshToken: refreshToken,
		}, nil
	}
}

// RefreshCookie 刷新Cookie有效期
// 通过访问nav接口来刷新Cookie有效期（B站会在每次请求时刷新Cookie有效期）
func RefreshCookie(cookieStr string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/nav", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("刷新Cookie失败: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// VerifyCookie 验证Cookie是否有效
func VerifyCookie(cookieStr string) (bool, int64, error) {
	client := &http.Client{Timeout: 30 * time.Second}

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, err
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			IsLogin bool  `json:"isLogin"`
			Mid     int64 `json:"mid"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, 0, err
	}

	return result.Data.IsLogin, result.Data.Mid, nil
}

// ParseCookieString 解析Cookie字符串为键值对map
func ParseCookieString(cookieStr string) map[string]string {
	kvs := make(map[string]string)
	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			kvs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return kvs
}
