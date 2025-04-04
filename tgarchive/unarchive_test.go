package tgarchive

import (
	"fmt"
	"github.com/kittenbark/tgmedia/tgdir"
	"testing"
)

func TestSendUnpacked(t *testing.T) {
	t.Parallel()

	for _, filename := range []string{"archive.tar", "archive.tar.gz", "archive.zip"} {
		t.Run(fmt.Sprintf("unpacked_%s", filename), func(t *testing.T) {
			t.Parallel()
			messages, err := SendUnpacked(bot.Context(), chat, "./archive.tar", &tgdir.Opt{Caption: filename})
			if err != nil {
				t.Fatal(err)
			}
			t.Log(messages)
		})
	}
}
