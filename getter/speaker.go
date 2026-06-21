package getter

import (
	"bili/config"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// 两个朗读队列：高优先级（SC/礼物/上舰）先消费
var (
	priorityChan = make(chan string, 8)
	normalChan   = make(chan string, 16)
	edgeTTSPath  string
	tmpDir       string
)

func init() {
	tmpDir = filepath.Join(os.TempDir(), "bili_speak")
	os.MkdirAll(tmpDir, 0755)
	edgeTTSPath = findEdgeTTS()
	go speakLoop()
}

// findEdgeTTS 查找 edge-tts 可执行文件路径
func findEdgeTTS() string {
	if p, err := exec.LookPath("edge-tts"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Library/Python/3.9/bin/edge-tts"),
		filepath.Join(home, "Library/Python/3.10/bin/edge-tts"),
		filepath.Join(home, "Library/Python/3.11/bin/edge-tts"),
		filepath.Join(home, "Library/Python/3.12/bin/edge-tts"),
		filepath.Join(home, "Library/Python/3.13/bin/edge-tts"),
		filepath.Join(home, ".local/bin/edge-tts"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// speakLoop 串行消费朗读队列，优先消费高优先级消息
func speakLoop() {
	for {
		var text string
		select {
		case text = <-priorityChan:
		case text = <-normalChan:
		}
		if config.Config.Speak != 1 {
			continue
		}
		if err := speakText(text); err != nil {
			fmt.Fprintf(os.Stderr, "speak error: %v\n", err)
		}
	}
}

// speakText 使用 edge-tts 生成并播放语音
func speakText(text string) error {
	if edgeTTSPath == "" {
		return fmt.Errorf("edge-tts not found, install with: pip install edge-tts")
	}

	mp3 := filepath.Join(tmpDir, "speech.mp3")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, edgeTTSPath,
		"--voice", "zh-CN-XiaoxiaoNeural",
		"--text", text,
		"--write-media", mp3,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("edge-tts failed: %w", err)
	}

	return playMP3(mp3)
}

// playMP3 跨平台播放 MP3 文件
func playMP3(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("afplay", path).Run()
	case "linux":
		if err := exec.Command("ffplay", "-nodisp", "-autoexit", path).Run(); err == nil {
			return nil
		}
		return exec.Command("mpg123", path).Run()
	case "windows":
		return exec.Command("powershell", "-c",
			fmt.Sprintf("(New-Object Media.SoundPlayer '%s').PlaySync()", path)).Run()
	}
	return fmt.Errorf("unsupported platform for MP3 playback")
}

// QueueSpeak 将弹幕消息加入朗读队列
// SC/礼物/上舰 送高优先级队列，普通弹幕送低优先级队列
func QueueSpeak(msg DanmuMsg) {
	if config.Config.Speak != 1 {
		return
	}

	switch msg.Type {
	case "DANMU_MSG":
		text := msg.Author + "说：" + msg.Content
		// 超过 5 条堆积时，丢弃旧弹幕，只读最新一条
		if len(normalChan) >= 5 {
			for {
				select {
				case <-normalChan:
				default:
					goto SEND_NORMAL
				}
			}
		}
SEND_NORMAL:
		select {
		case normalChan <- text:
		default:
		}
	case "SUPER_CHAT", "SEND_GIFT", "GUARD_BUY":
		var text string
		switch msg.Type {
		case "SUPER_CHAT":
			text = fmt.Sprintf("Super Chat %.1f元，来自%s：%s", float64(msg.GiftPrice), msg.Author, msg.Content)
		case "SEND_GIFT":
			text = fmt.Sprintf("%s 投喂了 %d 个 %s", msg.Author, msg.GiftNum, msg.GiftName)
		case "GUARD_BUY":
			text = fmt.Sprintf("%s 购买了 %s", msg.Author, msg.GiftName)
		}
		select {
		case priorityChan <- text:
		default:
		}
	}
}
