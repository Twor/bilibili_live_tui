package config

import (
	"bili/login"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	// 登录相关字段
	LastLoginTime int64  // 最后登录时间（Unix时间戳）
	CookieExpires int64  // Cookie过期时间（Unix时间戳）
	RefreshToken  string // 用于刷新Cookie的token
}

var Config ConfigType
var ConfigFile string // 配置文件路径

func defaultCfgFile() (configFile string, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	cfgDir := filepath.Join(homeDir, ".config", "bili")
	if err = os.MkdirAll(cfgDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}
	configFile = filepath.Join(cfgDir, "config.toml")
	_, err = os.Stat(configFile)
	if os.IsNotExist(err) {
		config := defaultConfig()
		f, err := os.Create(configFile)
		if err != nil {
			return "", fmt.Errorf("创建配置文件失败: %w", err)
		}
		defer f.Close()
		if err = toml.NewEncoder(f).Encode(config); err != nil {
			return "", fmt.Errorf("写入配置文件失败: %w", err)
		}
		fmt.Println("配置文件已生成，请修改配置文件后再次运行，配置文件路径为：" + configFile)
		os.Exit(0)
	}
	return configFile, nil
}

func defaultConfig() ConfigType {
	return ConfigType{
		Cookie:       "",
		RoomId:       7777,
		Theme:        3,
		SingleLine:   1,
		ShowTime:     1,
		Notify:       1,
		TimeColor:    "#FFFFFF",
		NameColor:    "#FFFFFF",
		ContentColor: "#FFFFFF",
		FrameColor:   "#FFFFFF",
		InfoColor:    "#FFFFFF",
		RankColor:    "#FFFFFF",
		Background:   "NONE",
	}
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
	Config.Cookie = result.Cookie
	Config.LastLoginTime = time.Now().Unix()
	Config.CookieExpires = time.Now().Unix() + 30*24*60*60 // 30天后过期
	Config.RefreshToken = result.RefreshToken

	if err := SaveConfig(); err != nil {
		return fmt.Errorf("保存登录信息失败: %w", err)
	}
	fmt.Println("登录信息已保存！")
	return nil
}

// setColorDefault 如果颜色值为空则设置默认值
func setColorDefault(color *string, def string) {
	if *color == "" {
		*color = def
	}
}

// setDefaultColors 为所有颜色配置项设置默认值
func setDefaultColors() {
	setColorDefault(&Config.TimeColor, "#bbbbbb")
	setColorDefault(&Config.NameColor, "#bbbbbb")
	setColorDefault(&Config.ContentColor, "#bbbbbb")
	setColorDefault(&Config.InfoColor, "#bbbbbb")
	setColorDefault(&Config.RankColor, "#bbbbbb")
	setColorDefault(&Config.FrameColor, "#bbbbbb")
	if Config.Background == "" {
		Config.Background = "NONE"
	}
}

func Init() {
	var err error
	configFile := ""
	roomId := int64(-1)
	theme := int64(-1)
	singleLine := int64(-1)
	showTime := int64(-1)
	notify := int64(-1)
	doLogin := false

	flag.StringVar(&configFile, "c", "", "配置文件路径")
	flag.Int64Var(&roomId, "r", -1, "直播间ID")
	flag.Int64Var(&theme, "t", -1, "主题 (1-4)")
	flag.Int64Var(&singleLine, "l", -1, "是否开启单行 (0/1)")
	flag.Int64Var(&showTime, "s", -1, "是否显示时间 (0/1)")
	flag.Int64Var(&notify, "n", -1, "是否开启通知 (0/1)")
	flag.BoolVar(&doLogin, "login", false, "强制扫码登录")
	flag.Parse()

	if configFile == "" {
		configFile, err = defaultCfgFile()
		if err != nil {
			panic(err)
		}
	}
	ConfigFile = configFile

	if _, err := toml.DecodeFile(configFile, &Config); err != nil {
		fmt.Printf("Error decoding config.toml: %s\n", err)
	}

	// 强制重新登录
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

	// 命令行参数覆盖配置文件
	if roomId != -1 {
		Config.RoomId = roomId
	}
	if theme != -1 {
		Config.Theme = theme
	}
	if singleLine != -1 {
		Config.SingleLine = singleLine
	}
	if showTime != -1 {
		Config.ShowTime = showTime
	}
	if notify != -1 {
		Config.Notify = notify
	}

	// 统一设置颜色默认值
	setDefaultColors()
}
