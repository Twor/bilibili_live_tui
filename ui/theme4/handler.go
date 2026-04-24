package theme4

import (
	"bili/config"
	"bili/getter"
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

const (
	colorDanmuMedal = "#FFD700"
	colorSuperChat  = "#0000FF"
	colorGift       = "#FF0000"
)

// danmuHandler 处理弹幕消息，按类型分流到不同视图
// messages: 弹幕和SC消息, access: 进入房间消息, gift: 礼物和上舰消息
func danmuHandler(app *tview.Application, messages *tview.TextView, access *tview.TextView, gift *tview.TextView, busChan chan getter.DanmuMsg) {
	for msg := range busChan {
		if strings.Trim(msg.Content, " ") == "" {
			continue
		}

		timeStr := msg.Time.Format("15:04")
		if config.Config.ShowTime == 0 {
			timeStr = ""
		}

		switch msg.Type {
		case "DANMU_MSG":
			var str string
			if msg.MedalLevel > 0 {
				str = fmt.Sprintf("[%s]%s [%s][%s%d] [%s]%s[%s]: %s",
					config.Config.TimeColor, timeStr,
					colorDanmuMedal, msg.MedalName, msg.MedalLevel,
					config.Config.NameColor, msg.Author,
					config.Config.ContentColor, msg.Content)
			} else {
				str = fmt.Sprintf("[%s]%s [%s]%s[%s]: %s",
					config.Config.TimeColor, timeStr,
					config.Config.NameColor, msg.Author,
					config.Config.ContentColor, msg.Content)
			}
			fmt.Fprintf(messages, "%s\n", str)

		case "SUPER_CHAT":
			str := fmt.Sprintf("[%s]%s [%s]SC ¥%d [%s]%s[%s]: %s",
				config.Config.TimeColor, timeStr,
				colorSuperChat, msg.GiftPrice,
				colorSuperChat, msg.Author,
				colorSuperChat, msg.Content)
			fmt.Fprintf(messages, "%s\n", str)

		case "SEND_GIFT":
			var str string
			if msg.CoinType == "gold" {
				str = fmt.Sprintf("[%s]%s [%s]%s: [%s] 投喂了 %d 个 %s（¥%.1f）",
					config.Config.TimeColor, timeStr,
					colorGift, msg.Author,
					colorGift, msg.GiftNum, msg.GiftName,
					float64(msg.GiftPrice)/1000.0)
			} else {
				str = fmt.Sprintf("[%s]%s [%s]%s: [%s] 投喂了 %d 个 %s",
					config.Config.TimeColor, timeStr,
					colorGift, msg.Author,
					colorGift, msg.GiftNum, msg.GiftName)
			}
			fmt.Fprintf(gift, "%s\n", str)

		case "GUARD_BUY":
			str := fmt.Sprintf("[%s]%s [%s]%s: [%s] 购买了 %s（¥%d）",
				config.Config.TimeColor, timeStr,
				colorGift, msg.Author,
				colorGift, msg.GiftName, msg.GiftPrice)
			fmt.Fprintf(gift, "%s\n", str)

		case "INTERACT_WORD":
			str := fmt.Sprintf("[%s]%s [%s]%s[%s]%s",
				config.Config.TimeColor, timeStr,
				config.Config.NameColor, msg.Author,
				config.Config.ContentColor, msg.Content)
			fmt.Fprintf(access, "%s\n", str)

		case "NOTICE_MSG":
			str := fmt.Sprintf("[%s]%s",
				config.Config.ContentColor, msg.Content)
			fmt.Fprintf(messages, "%s\n", str)

		default:
			str := fmt.Sprintf("[%s]%s [%s]%s[%s]: %s",
				config.Config.TimeColor, timeStr,
				config.Config.NameColor, msg.Author,
				config.Config.ContentColor, msg.Content)
			fmt.Fprintf(messages, "%s\n", str)
		}

		app.Draw()
	}
}
