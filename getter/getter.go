package getter

import (
	"bili/config"
	"fmt"
	"time"

	"github.com/Akegarasu/blivedm-go/client"
	"github.com/Akegarasu/blivedm-go/message"
	"github.com/asmcos/requests"
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

var historied = false

func getHistory(busChan chan DanmuMsg, roomID int) {
	if historied {
		return
	}

	historyApi := fmt.Sprintf("https://api.live.bilibili.com/xlive/web-room/v1/dM/gethistory?roomid=%d", roomID)
	r, err := requests.Get(historyApi)
	if err != nil {
		return
	}

	histories := gjson.Get(r.Text(), "data.room").Array()
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

		busChan <- danmu
	}
	historied = true
}

func syncRoomInfo(roomID int, roomInfoChan chan RoomInfo) {
	for {
		roomInfoApi := fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%d", roomID)
		roomInfo := new(RoomInfo)
		roomInfo.OnlineRankUsers = make([]OnlineRankUser, 0)
		r1, err1 := requests.Get(roomInfoApi)
		if err1 == nil {
			roomInfo.RoomId = roomID
			roomInfo.Uid = int(gjson.Get(r1.Text(), "data.uid").Int())
			roomInfo.Title = gjson.Get(r1.Text(), "data.title").String()
			roomInfo.AreaName = gjson.Get(r1.Text(), "data.area_name").String()
			roomInfo.ParentAreaName = gjson.Get(r1.Text(), "data.parent_area_name").String()
			roomInfo.Online = gjson.Get(r1.Text(), "data.online").Int()
			roomInfo.Attention = gjson.Get(r1.Text(), "data.attention").Int()
			_time, _ := time.Parse("2006-01-02 15:04:05", gjson.Get(r1.Text(), "data.live_time").String())
			seconds := time.Now().Unix() - _time.Unix() + 8*60*60
			days := seconds / 86400
			hours := (seconds % 86400) / 3600
			minutes := (seconds % 3600) / 60
			if days > 0 {
				roomInfo.Time = fmt.Sprintf("%d天%d时%d分", days, hours, minutes)
			} else if hours > 0 {
				roomInfo.Time = fmt.Sprintf("%d时%d分", hours, minutes)
			} else {
				roomInfo.Time = fmt.Sprintf("%d分", minutes)
			}
		}

		onlineRankApi := fmt.Sprintf("https://api.live.bilibili.com/xlive/general-interface/v1/rank/getOnlineGoldRank?ruid=%d&roomId=%d&page=1&pageSize=50", roomInfo.Uid, roomID)
		r2, err2 := requests.Get(onlineRankApi)
		if err2 == nil {
			rawUsers := gjson.Get(r2.Text(), "data.OnlineRankItem").Array()
			for _, rawUser := range rawUsers {
				user := OnlineRankUser{
					Name:  rawUser.Get("name").String(),
					Score: rawUser.Get("score").Int(),
					Rank:  rawUser.Get("userRank").Int(),
				}
				roomInfo.OnlineRankUsers = append(roomInfo.OnlineRankUsers, user)
			}
		}

		roomInfoChan <- *roomInfo
		time.Sleep(30 * time.Second)
	}
}

func Run(busChan chan DanmuMsg, roomInfoChan chan RoomInfo) {
	roomID := int(config.Config.RoomId)

	// 启动房间信息同步
	go syncRoomInfo(roomID, roomInfoChan)

	// 获取历史弹幕
	go getHistory(busChan, roomID)

	// 创建 blivedm 客户端
	c := client.NewClient(roomID)
	c.SetCookie(config.Config.Cookie)

	// 弹幕事件
	c.OnDanmaku(func(danmaku *message.Danmaku) {
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
			msg.Content = fmt.Sprintf("[表情]%s", danmaku.Emoticon.Url)
		}
		busChan <- msg
		notifyDanmu(msg)
	})

	// SC 事件
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

	// 礼物事件
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

	// 上舰事件
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

	// 关注事件
	// c.OnFollower(func(follower *message.Follower) {
	// 	msg := DanmuMsg{
	// 		Author:  follower.Username,
	// 		Content: "关注了直播间",
	// 		Type:    "FOLLOWER",
	// 		Time:    time.Now(),
	// 	}
	// 	busChan <- msg
	// 	notifyDanmu(msg)
	// })

	// 进入房间事件（使用自定义事件处理器）
	c.RegisterCustomEventHandler("INTERACT_WORD", func(s string) {
		// 解析 JSON 获取用户名
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

	// 启动客户端（在 goroutine 中）
	go func() {
		err := c.Start()
		if err != nil {
			msg := DanmuMsg{
				Author:  "system",
				Content: fmt.Sprintf("连接失败: %v", err),
				Type:    "NOTICE_MSG",
				Time:    time.Now(),
			}
			busChan <- msg
			return
		}

		msg := DanmuMsg{
			Author:  "system",
			Content: "已连接到弹幕服务器",
			Type:    "NOTICE_MSG",
			Time:    time.Now(),
		}
		busChan <- msg

		// goroutine 会自动保持运行，blivedm-go 内部有 wsLoop 和 heartBeatLoop
	}()
}
