package getter

import (
	"bili/config"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Akegarasu/blivedm-go/client"
	"github.com/Akegarasu/blivedm-go/message"
	"github.com/tidwall/gjson"
)

type OnlineRankUser struct {
	Name  string
	Score int64
	Rank  int64
}

type RoomInfo struct {
	RoomId          int
	Uid             int
	Title           string
	ParentAreaName  string
	AreaName        string
	Online          int64
	Attention       int64
	Time            string
	OnlineRankUsers []OnlineRankUser
}

type DanmuMsg struct {
	Author     string
	Content    string
	Type       string
	Time       time.Time
	MedalLevel int    // 粉丝灯牌等级
	MedalName  string // 粉丝灯牌名称
	GiftPrice  int64  // 礼物/SC金额（金瓜子/元）
	GiftNum    int64  // 礼物数量
	GiftName   string // 礼物名称
	CoinType   string // 币种类型 gold/silver
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// fetchJSON 发起 HTTP GET 请求并返回 JSON 字符串
func fetchJSON(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}
	return string(body), nil
}

// getHistory 使用 sync.Once 确保只执行一次
var getHistoryOnce sync.Once

func getHistory(busChan chan DanmuMsg, roomID int) {
	getHistoryOnce.Do(func() {
		historyApi := fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/dM/gethistory?roomid=%d", roomID)
		body, err := fetchJSON(historyApi)
		if err != nil {
			return
		}

		histories := gjson.Get(body, "data.room").Array()
		for _, history := range histories {
			t, _ := time.Parse("2006-01-02 15:04:05", history.Get("timeline").String())
			danmu := DanmuMsg{
				Author:  history.Get("nickname").String(),
				Content: history.Get("text").String(),
				Type:    "DANMU_MSG",
				Time:    t,
			}
			if history.Get("medal").Exists() {
				danmu.MedalLevel = int(history.Get("medal.0").Int())
				danmu.MedalName = history.Get("medal.1").String()
			}

			select {
			case busChan <- danmu:
			default:
				// channel 已满，丢弃该历史消息
			}
		}
	})
}

// buildLiveDuration 计算直播持续时长字符串
func buildLiveDuration(liveTimeStr string) string {
	liveTime, err := time.Parse("2006-01-02 15:04:05", liveTimeStr)
	if err != nil {
		return "未知"
	}
	seconds := time.Now().Unix() - liveTime.Unix() + 8*60*60
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%d天%d时%d分", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d时%d分", hours, minutes)
	}
	return fmt.Sprintf("%d分", minutes)
}

// fetchRoomInfo 获取房间信息
func fetchRoomInfo(roomID int) (*RoomInfo, error) {
	roomInfoApi := fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%d", roomID)
	body, err := fetchJSON(roomInfoApi)
	if err != nil {
		return nil, err
	}

	data := gjson.Get(body, "data")
	info := &RoomInfo{
		RoomId:         roomID,
		Uid:            int(data.Get("uid").Int()),
		Title:          data.Get("title").String(),
		AreaName:       data.Get("area_name").String(),
		ParentAreaName: data.Get("parent_area_name").String(),
		Online:         data.Get("online").Int(),
		Attention:      data.Get("attention").Int(),
		Time:           buildLiveDuration(data.Get("live_time").String()),
	}
	return info, nil
}

// fetchOnlineRank 获取在线排行榜
func fetchOnlineRank(uid int, roomID int) ([]OnlineRankUser, error) {
	apiURL := fmt.Sprintf(
		"https://api.live.bilibili.com/xlive/general-interface/v1/rank/getOnlineGoldRank?ruid=%d&roomId=%d&page=1&pageSize=50",
		uid, roomID,
	)
	body, err := fetchJSON(apiURL)
	if err != nil {
		return nil, err
	}

	rawUsers := gjson.Get(body, "data.OnlineRankItem").Array()
	users := make([]OnlineRankUser, 0, len(rawUsers))
	for _, rawUser := range rawUsers {
		user := OnlineRankUser{
			Name:  rawUser.Get("name").String(),
			Score: rawUser.Get("score").Int(),
			Rank:  rawUser.Get("userRank").Int(),
		}
		users = append(users, user)
	}
	return users, nil
}

// syncRoomInfo 定时同步房间信息和排行榜，每30秒刷新一次
func syncRoomInfo(roomID int, roomInfoChan chan RoomInfo) {
	// 立即执行第一次同步
	fetchAndSendRoomInfo(roomID, roomInfoChan)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fetchAndSendRoomInfo(roomID, roomInfoChan)
	}
}

// fetchAndSendRoomInfo 获取房间信息并发送到 channel
func fetchAndSendRoomInfo(roomID int, roomInfoChan chan RoomInfo) {
	info, err := fetchRoomInfo(roomID)
	if err != nil {
		return
	}

	rankUsers, err := fetchOnlineRank(info.Uid, roomID)
	if err == nil {
		info.OnlineRankUsers = rankUsers
	}

	select {
	case roomInfoChan <- *info:
	default:
		// channel 已满，丢弃旧数据
	}
}

// buildDanmuMsg 构建弹幕消息
func buildDanmuMsg(danmaku *message.Danmaku) DanmuMsg {
	msg := DanmuMsg{
		Author:  danmaku.Sender.Uname,
		Content: danmaku.Content,
		Type:    "DANMU_MSG",
		Time:    time.Now(),
	}
	if danmaku.Sender.Medal != nil {
		msg.MedalLevel = danmaku.Sender.Medal.Level
		msg.MedalName = danmaku.Sender.Medal.Name
	}
	if danmaku.Type == message.EmoticonDanmaku {
		emoticonStr := fmt.Sprintf("[%v]", danmaku.Extra.Content)
		// msg.Content = strings.TrimPrefix(emoticonStr, "upower_")
		msg.Content = emoticonStr
	}
	return msg
}

// setupEventHandlers 注册所有事件处理器
func setupEventHandlers(c *client.Client, busChan chan DanmuMsg) {
	c.OnDanmaku(func(danmaku *message.Danmaku) {
		msg := buildDanmuMsg(danmaku)
		busChan <- msg
		notifyDanmu(msg)
	})

	c.OnSuperChat(func(superChat *message.SuperChat) {
		msg := DanmuMsg{
			Author:    superChat.UserInfo.Uname,
			Content:   superChat.Message,
			Type:      "SUPER_CHAT",
			Time:      time.Now(),
			GiftPrice: int64(superChat.Price),
		}
		busChan <- msg
		notifyDanmu(msg)
	})

	c.OnGift(func(gift *message.Gift) {
		msg := DanmuMsg{
			Author:    gift.Uname,
			Content:   fmt.Sprintf("投喂了 %d 个 %s", gift.Num, gift.GiftName),
			Type:      "SEND_GIFT",
			Time:      time.Now(),
			GiftPrice: int64(gift.Price),
			GiftNum:   int64(gift.Num),
			GiftName:  gift.GiftName,
			CoinType:  gift.CoinType,
		}
		busChan <- msg
		notifyDanmu(msg)
	})

	c.OnGuardBuy(func(guardBuy *message.GuardBuy) {
		msg := DanmuMsg{
			Author:    guardBuy.Username,
			Content:   fmt.Sprintf("购买了 %s", guardBuy.GiftName),
			Type:      "GUARD_BUY",
			Time:      time.Now(),
			GiftPrice: int64(guardBuy.Price),
			GiftName:  guardBuy.GiftName,
		}
		busChan <- msg
		notifyDanmu(msg)
	})

	// 进入房间事件
	c.RegisterCustomEventHandler("INTERACT_WORD", func(s string) {
		result := gjson.Parse(s)
		uname := result.Get("data.uname").String()
		if uname != "" {
			msg := DanmuMsg{
				Author:  uname,
				Content: "进入了房间",
				Type:    "INTERACT_WORD",
				Time:    time.Now(),
			}
			busChan <- msg
			notifyDanmu(msg)
		}
	})
}

// startBlivedmClient 启动 blivedm 客户端（goroutine 会自动保持运行）
func startBlivedmClient(roomID int, busChan chan DanmuMsg) {
	c := client.NewClient(roomID)
	c.SetCookie(config.Config.Cookie)

	setupEventHandlers(c, busChan)

	go func() {
		busChan <- DanmuMsg{
			Author:  "system",
			Content: "正在连接弹幕服务器...",
			Type:    "NOTICE_MSG",
			Time:    time.Now(),
		}

		if err := c.Start(); err != nil {
			busChan <- DanmuMsg{
				Author:  "system",
				Content: fmt.Sprintf("连接失败: %v", err),
				Type:    "NOTICE_MSG",
				Time:    time.Now(),
			}
			return
		}

		busChan <- DanmuMsg{
			Author:  "system",
			Content: "已连接到弹幕服务器",
			Type:    "NOTICE_MSG",
			Time:    time.Now(),
		}

		// blivedm-go 内部有 wsLoop 和 heartBeatLoop，goroutine 自动保持运行
	}()
}

func Run(busChan chan DanmuMsg, roomInfoChan chan RoomInfo) {
	roomID := int(config.Config.RoomId)

	// 启动房间信息同步（每30秒刷新）
	go syncRoomInfo(roomID, roomInfoChan)

	// 获取历史弹幕（仅执行一次）
	go getHistory(busChan, roomID)

	// 启动 blivedm 客户端
	startBlivedmClient(roomID, busChan)
}
