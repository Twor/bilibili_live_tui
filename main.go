package main

import (
	"bili/config"
	"bili/getter"
	"bili/sender"
	"bili/ui"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// fixCharset 修正环境变量，避免亚洲语言区域设置导致终端显示问题
func fixCharset() {
	locale := os.Getenv("LANG")
	wideCharset := []string{"zh_", "jp_", "ko_", "ja_", "th_", "hi_"}
	for _, prefix := range wideCharset {
		if strings.HasPrefix(locale, prefix) {
			// 直接设置环境变量，无需重新 exec 自己
			os.Setenv("LANG", "C.UTF-8")
			return
		}
	}
}

func main() {
	fixCharset()
	config.Init()

	// 使用带缓冲的 channel，减少阻塞
	busChan := make(chan getter.DanmuMsg, 512)
	roomInfoChan := make(chan getter.RoomInfo, 32)

	getter.Run(busChan, roomInfoChan)

	if err := sender.Run(); err != nil {
		busChan <- getter.DanmuMsg{
			Author:  "system",
			Content: fmt.Sprintf("发送功能初始化失败: %v", err),
			Type:    "NOTICE_MSG",
		}
	}

	// 启动信号监听，实现优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		close(busChan)
		close(roomInfoChan)
		os.Exit(0)
	}()

	ui.Run(busChan, roomInfoChan)
}
