package tgdir

import (
	"context"
	"fmt"
	"github.com/kittenbark/tg"
	"github.com/kittenbark/tgmedia/tgvideo"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
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

		path = filepath.Join(dir, path)
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

func SendGrouped(ctx context.Context, chatId int64, dir string, opts ...*Opt) ([]*tg.Message, error) {
	optMediaGroup := optsToMediaGroup(opts)
	optDocument := optsToDocs(opts)

	result := []*tg.Message{}
	albumBuff := tg.Album{}
	cleanups := []func(){}
	defer func() {
		wg := sync.WaitGroup{}
		for _, cleanup := range cleanups {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cleanup()
			}()
		}
		wg.Wait()
	}()

	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if len(albumBuff) == 10 {
			messages, err := tg.SendMediaGroup(ctx, chatId, albumBuff, optMediaGroup)
			if err != nil {
				return err
			}
			result = append(result, messages...)
			albumBuff = tg.Album{}
		}

		path = filepath.Join(dir, path)
		switch filepath.Ext(path) {
		case ".mp4", ".mov":
			vid, cleanup, err := tgvideo.New(path)
			if err != nil {
				return fmt.Errorf("failed to create video %s: %w", path, err)
			}
			cleanups = append(cleanups, cleanup)
			albumBuff = append(albumBuff, vid)

		case ".webm":
			vid, cleanup, err := tgvideo.NewH264(path)
			if err != nil {
				return fmt.Errorf("failed to create video %s: %w", path, err)
			}
			cleanups = append(cleanups, cleanup)
			albumBuff = append(albumBuff, vid)

		case ".png", ".jpg", ".jpeg":
			albumBuff = append(albumBuff, &tg.Photo{Media: tg.FromDisk(path)})

		default:
			msg, err := tg.SendDocument(ctx, chatId, tg.FromDisk(path), optDocument)
			if err != nil {
				return fmt.Errorf("send document %s: %w", path, err)
			}
			result = append(result, msg)
		}

		return nil
	})

	if len(albumBuff) > 0 {
		messages, err := tg.SendMediaGroup(ctx, chatId, albumBuff, optMediaGroup)
		if err != nil {
			return result, err
		}
		result = append(result, messages...)
	}

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
func optsToMediaGroup(opts []*Opt) *tg.OptSendMediaGroup {
	if len(opts) == 0 {
		return &tg.OptSendMediaGroup{}
	}
	return &tg.OptSendMediaGroup{
		BusinessConnectionId: opts[0].BusinessConnectionId,
		MessageThreadId:      opts[0].MessageThreadId,
		DisableNotification:  opts[0].DisableNotification,
		ProtectContent:       opts[0].ProtectContent,
		AllowPaidBroadcast:   opts[0].AllowPaidBroadcast,
		MessageEffectId:      opts[0].MessageEffectId,
		ReplyParameters:      opts[0].ReplyParameters,
	}
}
