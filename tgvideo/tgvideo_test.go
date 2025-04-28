package tgvideo

import (
	"github.com/kittenbark/tg"
	"os"
	"strconv"
	"testing"
)

var (
	chat, _ = strconv.ParseInt(os.Getenv(tg.EnvTestingChat), 10, 64)
	bot     = tg.NewFromEnv().Scheduler()
)

func TestSend(t *testing.T) {
	t.Parallel()

	if _, err := Send(bot.Context(), chat, "./video.mp4"); err != nil {
		t.Fatal(err)
	}
}

func TestSendH264(t *testing.T) {
	t.Parallel()

	msg, err := SendH264(bot.Context(), chat, "./video.mp4")
	if err != nil {
		t.Fatal(err)
	}

	if msg.Video.FileName != "video.mp4" {
		t.Fatal(msg.Video.FileName, " != video.mp4")
	}
}

func TestSendNew(t *testing.T) {
	t.Parallel()

	vid, cleanup, err := New("./video.mp4")
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	vidH64, cleanupH64, errH64 := NewH264("./video.mp4")
	defer cleanupH64()
	if errH64 != nil {
		t.Fatal(errH64)
	}

	if _, err := tg.SendMediaGroup(bot.Context(), chat, tg.Album{vid, vidH64}); err != nil {
		t.Fatal(err)
	}
}
