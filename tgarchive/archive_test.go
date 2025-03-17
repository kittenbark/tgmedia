package tgarchive

import (
	"github.com/kittenbark/tg"
	"os"
	"strconv"
	"testing"
)

var (
	chat, _ = strconv.ParseInt(os.Getenv(tg.EnvTestingChat), 10, 64)
)

func TestSendByN(t *testing.T) {
	bot := tg.NewFromEnv().Scheduler()

	messages, err := SendByN(bot.Context(), chat, "./data", "frauromana.tar", 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(messages)
}

func TestSendByN_Bad(t *testing.T) {
	bot := tg.NewFromEnv().Scheduler()

	_, err := SendByN(bot.Context(), chat, "./data", "frauromana.tar", 10<<20)
	if err == nil {
		t.Fatal("an error was expected, some files are too big")
	}
}
