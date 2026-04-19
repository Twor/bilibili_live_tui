package getter

import (
	"bili/config"
	"fmt"
	"os/exec"
)

// NotifySend 通过 notify-send 发送桌面通知
func NotifySend(title, body string) {
	if config.Config.Notify != 1 {
		return
	}
	// 非阻塞执行 notify-send，忽略错误（如果系统没有安装 notify-send）
	cmd := exec.Command("notify-send", title, body)
	_ = cmd.Start()
}

// notifyDanmu 根据弹幕消息类型发送桌面通知
func notifyDanmu(msg DanmuMsg) {
	if config.Config.Notify != 1 {
		return
	}

	switch msg.Type {
	case "DANMU_MSG":
		if msg.MedalLevel > 0 {
			NotifySend("弹幕", fmt.Sprintf("[%s%d] %s: %s", msg.MedalName, msg.MedalLevel, msg.Author, msg.Content))
		} else {
			NotifySend("弹幕", fmt.Sprintf("%s: %s", msg.Author, msg.Content))
		}
	case "SUPER_CHAT":
		NotifySend("SC", fmt.Sprintf("¥%d %s: %s", msg.GiftPrice, msg.Author, msg.Content))
	case "SEND_GIFT":
		if msg.CoinType == "gold" {
			NotifySend("🎁 礼物", fmt.Sprintf("%s: 投喂了 %d 个 %s（¥%.1f）", msg.Author, msg.GiftNum, msg.GiftName, float64(msg.GiftPrice)/1000.0))
		} else {
			NotifySend("🎁 礼物", fmt.Sprintf("%s: 投喂了 %d 个 %s", msg.Author, msg.GiftNum, msg.GiftName))
		}
	case "GUARD_BUY":
		NotifySend("🚢 舰长", fmt.Sprintf("%s: 购买了 %s（¥%d）", msg.Author, msg.GiftName, msg.GiftPrice))
	// case "INTERACT_WORD":
	// 	NotifySend("进入", fmt.Sprintf("%s 进入了直播间", msg.Author))
	case "FOLLOWER": // 关注事件
		NotifySend("💚 关注", fmt.Sprintf("%s 关注了直播间", msg.Author))
	case "NOTICE_MSG":
		NotifySend("📢 系统消息", msg.Content)
	}
}
