package theme4

import (
	"bili/config"
	"bili/getter"
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

var lastMsg = getter.DanmuMsg{}
var lastLine = ""

func roomInfoHandler(app *tview.Application, roomInfoView *tview.TextView, rankUsersView *tview.TextView, roomInfoChan chan getter.RoomInfo) {
	for roomInfo := range roomInfoChan {
		roomInfoView.SetText(
			"[" + config.Config.InfoColor + "]" +
				roomInfo.Title + "\n" +
				fmt.Sprintf("ID: %d", roomInfo.RoomId) + "\n" +
				fmt.Sprintf("分区: %s/%s", roomInfo.ParentAreaName, roomInfo.AreaName) + "\n" +
				fmt.Sprintf("👀: %d", roomInfo.Online) + "\n" +
				fmt.Sprintf("❤️: %d", roomInfo.Attention) + "\n" +
				fmt.Sprintf("🕒: %s", roomInfo.Time) + "\n",
		)
		rankUsersView.SetTitle(fmt.Sprintf("Rank(%d)", len(roomInfo.OnlineRankUsers)))

		rankUserStr := ""
		spec := []string{"👑 ", "🥈 ", "🥉 "}
		for idx, rankUser := range roomInfo.OnlineRankUsers {
			rankUserStr += "[" + config.Config.RankColor + "]"
			if idx < 3 {
				rankUserStr += spec[idx] + rankUser.Name + "\n"
			} else {
				rankUserStr += "   " + rankUser.Name + "\n"
			}
		}
		strings.TrimRight(rankUserStr, "\n")
		rankUsersView.SetText(rankUserStr)
		// 滚动到顶部 避免过长显示下半部分
		roomInfoView.ScrollToBeginning()
		rankUsersView.ScrollToBeginning()
		app.Draw()
	}
}

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
				str = fmt.Sprintf("[%s]%s [#FFD700][%s%d] [%s]%s[%s] %s", config.Config.TimeColor, timeStr, msg.MedalName, msg.MedalLevel, config.Config.NameColor, msg.Author, config.Config.ContentColor, msg.Content)
			} else {
				str = fmt.Sprintf("[%s]%s [%s]%s[%s] %s", config.Config.TimeColor, timeStr, config.Config.NameColor, msg.Author, config.Config.ContentColor, msg.Content)
			}
			fmt.Fprintf(messages, "%s\n", str)
		case "SUPER_CHAT":
			var str string
			str = fmt.Sprintf("[%s]%s [#0000FF]SC ¥%d [#0000FF]%s[#0000FF]: %s", config.Config.TimeColor, timeStr, msg.GiftPrice, msg.Author, msg.Content)
			fmt.Fprintf(messages, "%s\n", str)
		case "SEND_GIFT":
			var str string
			if msg.CoinType == "gold" {
				str = fmt.Sprintf("[%s]%s [#FF0000]%s[#FF0000] 投喂了 %d 个 %s（¥%.1f）", config.Config.TimeColor, timeStr, msg.Author, msg.GiftNum, msg.GiftName, float64(msg.GiftPrice)/1000.0)
			} else {
				str = fmt.Sprintf("[%s]%s [#FF0000]%s[#FF0000] 投喂了 %d 个 %s", config.Config.TimeColor, timeStr, msg.Author, msg.GiftNum, msg.GiftName)
			}
			fmt.Fprintf(gift, "%s\n", str)
		case "GUARD_BUY":
			var str string
			str = fmt.Sprintf("[%s]%s [#FF0000]%s[#FF0000] 购买了 %s（¥%d）", config.Config.TimeColor, timeStr, msg.Author, msg.GiftName, msg.GiftPrice)
			fmt.Fprintf(gift, "%s\n", str)
		case "INTERACT_WORD":
			var str string
			str = fmt.Sprintf("[%s]%s [%s]%s[%s] 进入了直播间", config.Config.TimeColor, timeStr, config.Config.NameColor, msg.Author, config.Config.ContentColor)
			fmt.Fprintf(access, "%s\n", str)
		case "NOTICE_MSG":
			var str string
			str = fmt.Sprintf("[%s]%s", config.Config.ContentColor, msg.Content)
			fmt.Fprintf(messages, "%s\n", str)
		default:
			var str string
			str = fmt.Sprintf("[%s]%s [%s]%s[%s] %s", config.Config.TimeColor, timeStr, config.Config.NameColor, msg.Author, config.Config.ContentColor, msg.Content)
			fmt.Fprintf(messages, "%s\n", str)
		}

		lastMsg = msg
		app.Draw()
	}
}