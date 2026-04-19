package config

import (
	"bili/login"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type ConfigType struct {
	Cookie       string // 登录cookie
	RoomId       int64  // 直播间id
	Theme        int64  // 主题
	SingleLine   int64  // 是否开启单行
	ShowTime     int64  // 是否显示时间
	Notify       int64  // 是否发送桌面通知
	TimeColor    string // 时间颜色
	NameColor    string // 名字颜色
	ContentColor string // 内容颜色
	FrameColor   string // 边框颜色
	InfoColor    string // 房间信息颜色
	RankColor    string // 排行榜颜色
	Background   string // 背景颜色
	// 新增字段
	LastLoginTime int64  // 最后登录时间（Unix时间戳）
	CookieExpires int64  // Cookie过期时间（Unix时间戳）
	RefreshToken  string // 用于刷新Cookie的token
}

var Config ConfigType
var ConfigFile string // 配置文件路径

func defaultCfgFile() (configFile string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	path := cwd + "/config"
	if err = os.MkdirAll(path, 0755); err != nil {
		return
	}
	configFile = path + "/config.toml"
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		var f *os.File
		config := ConfigType{
			Cookie:       "",
			RoomId:       7777,
			Theme:        3,         // 主题，1=简约，2=经典，3=信息丰富，4=分栏显示
			SingleLine:   1,         // 是否开启单行显示，开启后每条消息只占一行，关闭后多行消息会占用多行显示
			ShowTime:     1,         // 是否显示时间
			Notify:       1,         // 是否发送桌面通知
			TimeColor:    "#FFFFFF", // 时间颜色，支持常见颜色名称和十六进制颜色代码
			NameColor:    "#FFFFFF", // 名字颜色，支持常见颜色名称和十六进制颜色代码
			ContentColor: "#FFFFFF", // 内容颜色，支持常见颜色名称和十六进制颜色代码
			FrameColor:   "#FFFFFF", // 边框颜色，支持常见颜色名称和十六进制颜色代码
			InfoColor:    "#FFFFFF", // 房间信息颜色，支持常见颜色名称和十六进制颜色代码
			RankColor:    "#FFFFFF", // 排行榜颜色，支持常见颜色名称和十六进制颜色代码
			Background:   "NONE",    // 默认无背景颜色 NONE表示无背景颜色
		}
		f, err = os.Create(configFile)
		if err != nil {
			return
		}
		defer f.Close()
		if err = toml.NewEncoder(f).Encode(config); err != nil {
			return
		}

		fmt.Println("配置文件已生成，请修改配置文件后再次运行，配置文件路径为：" + configFile)
		os.Exit(0)
	}

	return
}

// SaveConfig 保存配置到文件
func SaveConfig() error {
	if ConfigFile == "" {
		return fmt.Errorf("配置文件路径未设置")
	}

	f, err := os.Create(ConfigFile)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	defer f.Close()

	if err = toml.NewEncoder(f).Encode(Config); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// CheckAndRefreshCookie 检查Cookie是否有效，如果过期则刷新或重新登录
func CheckAndRefreshCookie() error {
	// 检查Cookie是否过期
	now := time.Now().Unix()

	// 如果Cookie过期时间已到，或者没有设置过期时间
	if Config.CookieExpires == 0 || now >= Config.CookieExpires {
		fmt.Println("Cookie已过期或未设置，需要重新登录...")
		return DoLogin()
	}

	// 如果Cookie即将过期（7天内），尝试刷新
	daysUntilExpiry := (Config.CookieExpires - now) / (24 * 60 * 60)
	if daysUntilExpiry <= 7 {
		fmt.Println("Cookie即将过期，尝试刷新...")
		if err := login.RefreshCookie(Config.Cookie); err != nil {
			fmt.Println("刷新Cookie失败，需要重新登录...")
			return DoLogin()
		}
		// 刷新成功，更新过期时间（延长30天）
		Config.CookieExpires = now + 30*24*60*60
		Config.LastLoginTime = now
		if err := SaveConfig(); err != nil {
			fmt.Printf("保存配置失败: %v\n", err)
		}
		fmt.Println("Cookie刷新成功！")
	}

	// 验证Cookie是否有效
	isValid, _, err := login.VerifyCookie(Config.Cookie)
	if err != nil || !isValid {
		fmt.Println("Cookie验证失败，需要重新登录...")
		return DoLogin()
	}

	return nil
}

// DoLogin 执行扫码登录
func DoLogin() error {
	result, err := login.DoLogin()
	if err != nil {
		return fmt.Errorf("扫码登录失败: %w", err)
	}

	// GoBilibiliLogin直接返回Cookie字符串，无需额外转换
	Config.Cookie = result.Cookie
	Config.LastLoginTime = time.Now().Unix()
	Config.CookieExpires = time.Now().Unix() + 30*24*60*60 // 30天后过期
	Config.RefreshToken = result.RefreshToken

	// 保存配置
	if err := SaveConfig(); err != nil {
		return fmt.Errorf("保存登录信息失败: %w", err)
	}

	fmt.Println("登录信息已保存！")
	return nil
}

func Init() {
	var err error
	configFile := ""
	roomId := int64(-1)
	theme := int64(-1)
	single_line := int64(-1)
	show_time := int64(-1)
	notify := int64(-1)

	// 新增命令行参数
	doLogin := false
	flag.StringVar(&configFile, "c", "", "usage for config")
	flag.Int64Var(&roomId, "r", -1, "usage for room id")
	flag.Int64Var(&theme, "t", -1, "usage for theme")
	flag.Int64Var(&single_line, "l", -1, "usage for single_line")
	flag.Int64Var(&show_time, "s", -1, "usage for show_time")
	flag.Int64Var(&notify, "n", -1, "usage for notify (0=off, 1=on)")
	flag.BoolVar(&doLogin, "login", false, "force QR code login")
	flag.Parse()

	if configFile == "" {
		configFile, err = defaultCfgFile()
		if err != nil {
			panic(err)
		}
	}

	// 保存配置文件路径
	ConfigFile = configFile

	if _, err := toml.DecodeFile(configFile, &Config); err != nil {
		fmt.Printf("Error decoding config.toml: %s\n", err)
	}

	// 如果指定了 --login 参数，强制重新登录
	if doLogin {
		if err := DoLogin(); err != nil {
			panic(fmt.Sprintf("登录失败: %v", err))
		}
	}

	// 检查是否需要登录
	if Config.Cookie == "" || Config.Cookie == "BILIBILI Cookie" {
		fmt.Println("未检测到有效Cookie，开始扫码登录...")
		if err := DoLogin(); err != nil {
			panic(fmt.Sprintf("登录失败: %v", err))
		}
	}

	// 检查并刷新Cookie
	if err := CheckAndRefreshCookie(); err != nil {
		panic(fmt.Sprintf("登录状态检查失败: %v", err))
	}

	if roomId != -1 {
		Config.RoomId = roomId
	}
	if theme != -1 {
		Config.Theme = theme
	}
	if single_line != -1 {
		Config.SingleLine = single_line
	}
	if show_time != -1 {
		Config.ShowTime = show_time
	}
	if notify != -1 {
		Config.Notify = notify
	}
	if Config.TimeColor == "" {
		Config.TimeColor = "#bbbbbb"
	}
	if Config.NameColor == "" {
		Config.NameColor = "#bbbbbb"
	}
	if Config.ContentColor == "" {
		Config.ContentColor = "#bbbbbb"
	}
	if Config.TimeColor == "" {
		Config.TimeColor = "#bbbbbb"
	}
	if Config.NameColor == "" {
		Config.NameColor = "#bbbbbb"
	}
	if Config.ContentColor == "" {
		Config.ContentColor = "#bbbbbb"
	}
	if Config.InfoColor == "" {
		Config.InfoColor = "#bbbbbb"
	}
	if Config.RankColor == "" {
		Config.RankColor = "#bbbbbb"
	}
	if Config.FrameColor == "" {
		Config.FrameColor = "#bbbbbb"
	}
	if Config.Background == "" {
		Config.Background = "NONE"
	}
}
