package tgarchive

import (
	"github.com/kittenbark/tg"
	"github.com/kittenbark/tgmedia/tgdir"
	"os"
	"strconv"
	"testing"
)

var (
	chat, _ = strconv.ParseInt(os.Getenv(tg.EnvTestingChat), 10, 64)
	bot     = tg.NewFromEnv().Scheduler()
)

func TestSendByN(t *testing.T) {
	t.Parallel()

	messages, err := SendByN(bot.Context(), chat, "./data", "frauromana.tar", 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(messages)
}

func TestSendByN_Bad(t *testing.T) {
	t.Parallel()

	_, err := SendByN(bot.Context(), chat, "./data", "frauromana.tar", 10<<20)
	if err == nil {
		t.Fatal("an error was expected, some files are too big")
	}
}

func TestTgdir(t *testing.T) {
	t.Parallel()

	t.Run("ungrouped", func(t *testing.T) {
		t.Parallel()
		_, err := tgdir.Send(bot.Context(), chat, "./data")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("grouped", func(t *testing.T) {
		t.Parallel()
		_, err := tgdir.SendGrouped(bot.Context(), chat, "./data")
		if err != nil {
			t.Fatal(err)
		}
	})
}
