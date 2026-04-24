// simple

package theme3

import (
	"bili/config"
	"bili/getter"
	"bili/sender"
	"bili/ui/common"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func roomInfoHandler(app *tview.Application, roomInfoView *tview.TextView, roomInfoChan chan getter.RoomInfo) {
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
		roomInfoView.ScrollToBeginning()
		app.Draw()
	}
}

func draw(app *tview.Application, roomId int64, busChan chan getter.DanmuMsg, roomInfoChan chan getter.RoomInfo) *tview.Grid {
	grid := tview.NewGrid().SetRows(1, 1, 0, 1, 1).SetBorders(false)

	roomInfoView := tview.NewTextView().SetDynamicColors(true)
	roomInfoView.SetBackgroundColor(common.Bg)

	delimiter1 := tview.NewTextView().SetTextColor(tcell.GetColor(config.Config.FrameColor))
	delimiter2 := tview.NewTextView().SetTextColor(tcell.GetColor(config.Config.FrameColor))
	delimiter1.SetBackgroundColor(common.Bg).SetBorder(false)
	delimiter2.SetBackgroundColor(common.Bg).SetBorder(false)

	_, _, width, _ := grid.GetRect()
	str := ""
	for i := 0; i < width; i++ {
		str = str + "—"
	}
	delimiter1.SetText(str)
	delimiter2.SetText(str)

	messagesView := tview.NewTextView().SetDynamicColors(true)
	messagesView.SetBackgroundColor(common.Bg)

	input := tview.NewInputField()
	input.SetFormAttributes(0, tcell.ColorDefault, common.Bg, tcell.ColorDefault, common.Bg)

	grid.
		AddItem(roomInfoView, 0, 0, 1, 1, 0, 0, false).
		AddItem(delimiter1, 1, 0, 1, 1, 0, 0, false).
		AddItem(messagesView, 2, 0, 1, 1, 0, 0, false).
		AddItem(delimiter2, 3, 0, 1, 1, 0, 0, false).
		AddItem(input /*  */, 4, 0, 1, 1, 0, 0, true)

	go roomInfoHandler(app, roomInfoView, roomInfoChan)
	go common.DanmuHandler(app, messagesView, busChan)

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			go sender.SendMsg(roomId, input.GetText(), busChan)

			common.SubmitHistory = append(common.SubmitHistory, input.GetText())
			if len(common.SubmitHistory) > 10 {
				common.SubmitHistory = common.SubmitHistory[1:]
			}
			common.SubmitHistoryIndex = len(common.SubmitHistory)

			input.SetText("")
		}
	})

	grid.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		str := ""
		for i := 0; i < width; i++ {
			str = str + "—"
		}
		delimiter1.SetText(str)
		delimiter2.SetText(str)
		return x, y, width, height
	})

	return grid
}

func Run(busChan chan getter.DanmuMsg, roomInfoChan chan getter.RoomInfo) {
	if config.Config.Background != "NONE" {
		common.Bg = tcell.GetColor(config.Config.Background)
	}
	app := tview.NewApplication()
	if err := app.SetRoot(draw(app, config.Config.RoomId, busChan, roomInfoChan), true).EnableMouse(false).Run(); err != nil {
		panic(err)
	}
}
