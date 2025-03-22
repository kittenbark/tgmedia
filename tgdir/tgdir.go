package tgdir

import (
	"context"
	"fmt"
	"github.com/kittenbark/tg"
	"github.com/kittenbark/tgmedia/tgvideo"
	"io/fs"
	"os"
	"path/filepath"
)

type Opt = tg.OptSendVideo

func Send(ctx context.Context, chatId int64, dir string, opts ...*Opt) ([]*tg.Message, error) {
	return SendDocumentsVerbose(ctx, chatId, dir, false, optsToPhoto(opts), optsToVideo(opts), optsToDocs(opts))
}

func SendDocs(ctx context.Context, chatId int64, dir string, opts ...*Opt) ([]*tg.Message, error) {
	return SendDocumentsVerbose(ctx, chatId, dir, true, optsToPhoto(opts), optsToVideo(opts), optsToDocs(opts))
}

func SendDocumentsVerbose(
	ctx context.Context,
	chatId int64,
	dir string,
	sendPhotosAsDocs bool,
	optPhoto *tg.OptSendPhoto,
	optVideo *tg.OptSendVideo,
	optDocument *tg.OptSendDocument,
) ([]*tg.Message, error) {
	if optPhoto == nil {
		optPhoto = &tg.OptSendPhoto{}
	}
	if optVideo == nil {
		optVideo = &tg.OptSendVideo{}
	}
	if optDocument == nil {
		optDocument = &tg.OptSendDocument{}
	}

	result := []*tg.Message{}
	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		switch filepath.Ext(path) {
		case ".mp4", ".mov":
			msg, err := tgvideo.Send(ctx, chatId, path, optVideo)
			if err != nil {
				return fmt.Errorf("send video %s: %w", path, err)
			}
			result = append(result, msg)
		case ".webm":
			msg, err := tgvideo.SendH264(ctx, chatId, path, optVideo)
			if err != nil {
				return fmt.Errorf("send video %s: %w", path, err)
			}
			result = append(result, msg)
		case ".png", ".jpg", ".jpeg":
			if !sendPhotosAsDocs {
				msg, err := tg.SendPhoto(ctx, chatId, tg.FromDisk(path), optPhoto)
				if err != nil {
					return fmt.Errorf("send picture %s: %w", path, err)
				}
				result = append(result, msg)
				break
			}
			fallthrough
		default:
			msg, err := tg.SendDocument(ctx, chatId, tg.FromDisk(path), optDocument)
			if err != nil {
				return fmt.Errorf("send document %s: %w", path, err)
			}
			result = append(result, msg)
		}

		return nil
	})

	return result, err
}

func optsToPhoto(opts []*Opt) *tg.OptSendPhoto {
	if len(opts) == 0 {
		return &tg.OptSendPhoto{}
	}

	return &tg.OptSendPhoto{
		BusinessConnectionId:  opts[0].BusinessConnectionId,
		MessageThreadId:       opts[0].MessageThreadId,
		Caption:               opts[0].Caption,
		ParseMode:             opts[0].ParseMode,
		CaptionEntities:       opts[0].CaptionEntities,
		ShowCaptionAboveMedia: opts[0].ShowCaptionAboveMedia,
		HasSpoiler:            opts[0].HasSpoiler,
		DisableNotification:   opts[0].DisableNotification,
		ProtectContent:        opts[0].ProtectContent,
		AllowPaidBroadcast:    opts[0].AllowPaidBroadcast,
		MessageEffectId:       opts[0].MessageEffectId,
		ReplyParameters:       opts[0].ReplyParameters,
		ReplyMarkup:           opts[0].ReplyMarkup,
	}
}

func optsToVideo(opts []*Opt) *tg.OptSendVideo {
	if len(opts) == 0 {
		return &tg.OptSendVideo{}
	}
	return opts[0]
}

func optsToDocs(opts []*Opt) *tg.OptSendDocument {
	if len(opts) == 0 {
		return &tg.OptSendDocument{}
	}
	return &tg.OptSendDocument{
		BusinessConnectionId: opts[0].BusinessConnectionId,
		MessageThreadId:      opts[0].MessageThreadId,
		Caption:              opts[0].Caption,
		ParseMode:            opts[0].ParseMode,
		CaptionEntities:      opts[0].CaptionEntities,
		DisableNotification:  opts[0].DisableNotification,
		ProtectContent:       opts[0].ProtectContent,
		AllowPaidBroadcast:   opts[0].AllowPaidBroadcast,
		MessageEffectId:      opts[0].MessageEffectId,
		ReplyParameters:      opts[0].ReplyParameters,
		ReplyMarkup:          opts[0].ReplyMarkup,
	}
}
