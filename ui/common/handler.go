package common

import (
	"bili/config"
	"bili/getter"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	colorDanmuMedal = "#FFD700"
	colorSuperChat  = "#0000FF"
	colorGift       = "#FF0000"
)

var (
	Bg                 = tcell.ColorDefault
	SubmitHistory      = []string{}
	SubmitHistoryIndex = 0
)

func SetBoxAttr(box *tview.Box, title string) {
	box.SetBorder(true)
	box.SetTitleAlign(tview.AlignLeft)
	box.SetTitle(title)
	box.SetBackgroundColor(Bg)
	box.SetBorderColor(tcell.GetColor(config.Config.FrameColor))
	box.SetTitleColor(tcell.GetColor(config.Config.FrameColor))
}

func RoomInfoHandler(app *tview.Application, roomInfoView *tview.TextView, rankUsersView *tview.TextView, roomInfoChan chan getter.RoomInfo) {
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

func DanmuHandler(app *tview.Application, messages *tview.TextView, busChan chan getter.DanmuMsg) {
	for msg := range busChan {
		if strings.Trim(msg.Content, " ") == "" {
			continue
		}

		str := ""

		timeStr := msg.Time.Format("15:04")
		if config.Config.ShowTime == 0 {
			timeStr = ""
		}

		switch msg.Type {
		case "DANMU_MSG":
			if msg.MedalLevel > 0 {
				str += fmt.Sprintf("[%s]%s [%s][%s%d] [%s]%s[%s]: %s", config.Config.TimeColor, timeStr, colorDanmuMedal, msg.MedalName, msg.MedalLevel, config.Config.NameColor, msg.Author, config.Config.ContentColor, msg.Content)
			} else {
				str += fmt.Sprintf("[%s]%s [%s]%s[%s]: %s", config.Config.TimeColor, timeStr, config.Config.NameColor, msg.Author, config.Config.ContentColor, msg.Content)
			}
		case "SUPER_CHAT":
			str += fmt.Sprintf("[%s]%s [%s]SC ¥%d [%s]%s[%s]: %s", config.Config.TimeColor, timeStr, colorSuperChat, msg.GiftPrice, colorSuperChat, msg.Author, colorSuperChat, msg.Content)
		case "SEND_GIFT":
			if msg.CoinType == "gold" {
				str += fmt.Sprintf("[%s]%s [%s]%s: [%s] 投喂了 %d 个 %s（¥%.1f）", config.Config.TimeColor, timeStr, colorGift, msg.Author, colorGift, msg.GiftNum, msg.GiftName, float64(msg.GiftPrice)/1000.0)
			} else {
				str += fmt.Sprintf("[%s]%s [%s]%s: [%s] 投喂了 %d 个 %s", config.Config.TimeColor, timeStr, colorGift, msg.Author, colorGift, msg.GiftNum, msg.GiftName)
			}
		case "GUARD_BUY":
			str += fmt.Sprintf("[%s]%s [%s]%s: [%s] 购买了 %s（¥%d）", config.Config.TimeColor, timeStr, colorGift, msg.Author, colorGift, msg.GiftName, msg.GiftPrice)
		case "INTERACT_WORD":
			str += fmt.Sprintf("[%s]%s [%s]%s[%s]%s", config.Config.TimeColor, timeStr, config.Config.NameColor, msg.Author, config.Config.ContentColor, msg.Content)
		case "NOTICE_MSG":
			str += fmt.Sprintf("[%s]%s", config.Config.ContentColor, msg.Content)
		default:
			str += fmt.Sprintf("[%s]%s [%s]%s[%s]: %s", config.Config.TimeColor, timeStr, config.Config.NameColor, msg.Author, config.Config.ContentColor, msg.Content)
		}

		fmt.Fprintf(messages, "%s\n", str)
		app.Draw()
	}
}
