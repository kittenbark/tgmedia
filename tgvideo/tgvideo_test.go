package tgvideo

import (
	"github.com/kittenbark/tg"
	"os"
	"strconv"
	"testing"
)

var (
	chat, _ = strconv.ParseInt(os.Getenv(tg.EnvTestingChat), 10, 64)
)

func TestSend(t *testing.T) {
	bot := tg.NewFromEnv()
	if _, err := Send(bot.Context(), chat, "./video.mp4"); err != nil {
		t.Fatal(err)
	}
}

func TestSendH264(t *testing.T) {
	bot := tg.NewFromEnv()
	if _, err := SendH264(bot.Context(), chat, "./video.mp4"); err != nil {
		t.Fatal(err)
	}
}
