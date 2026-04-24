// 极简主题

package theme2

import (
	"bili/config"
	"bili/getter"
	"bili/sender"
	"bili/ui/common"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func draw(app *tview.Application, roomId int64, busChan chan getter.DanmuMsg, roomInfoChan chan getter.RoomInfo) *tview.Grid {
	chatGrid := tview.NewGrid().SetRows(0, 1).SetBorders(false)
	messagesView := tview.NewTextView().SetDynamicColors(true)
	messagesView.SetBackgroundColor(common.Bg)

	input := tview.NewInputField()
	input.SetFormAttributes(0, tcell.ColorDefault, common.Bg, tcell.ColorDefault, common.Bg)

	chatGrid.
		AddItem(messagesView, 0, 0, 1, 1, 0, 0, false).
		AddItem(input, 1, 0, 1, 1, 0, 0, true)

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

	return chatGrid
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
