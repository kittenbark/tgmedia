package tgvideo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kittenbark/tg"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

var (
	Ffprobe = "ffprobe"
	Ffmpeg  = "ffmpeg"
	Preset  = "medium"
)

func Send(ctx context.Context, chatId int64, filename string, opts ...*tg.OptSendVideo) (*tg.Message, error) {
	return send(ctx, chatId, filename, filename, opts...)
}

func SendH264(ctx context.Context, chatId int64, filename string, opts ...*tg.OptSendVideo) (*tg.Message, error) {
	converted, err := os.CreateTemp("", "kittenbark_tgmedia_*.mp4")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer time.AfterFunc(time.Second*5, func() {
		_ = converted.Close()
		_ = os.Remove(converted.Name())
	})
	if err := convertH264(filename, converted); err != nil {
		return nil, err
	}

	return send(ctx, chatId, converted.Name(), filepath.Base(filename), opts...)
}

func New(filename string) (video *tg.Video, cleanup func(), err error) {
	temporaryFiles := []string{}
	cleanup = func() {
		wg := &sync.WaitGroup{}
		for _, file := range temporaryFiles {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = os.Remove(file)
			}()
		}
	}

	thumbnailFile, err := os.CreateTemp("", "kittenbark_tgmedia_*.jpg")
	if err != nil {
		return nil, cleanup, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func(thumbnailFile *os.File) { _ = thumbnailFile.Close() }(thumbnailFile)
	temporaryFiles = append(temporaryFiles, thumbnailFile.Name())

	thumbnail, err := buildThumbnail(filename, thumbnailFile)
	if err != nil {
		defer cleanup()
		return nil, cleanup, err
	}

	meta, err := getFileMetadata(filename)
	if err != nil {
		defer cleanup()
		return nil, cleanup, fmt.Errorf("failed to get file metadata: %w", err)
	}

	return &tg.Video{
		Media:             tg.FromDisk(filename),
		Thumbnail:         thumbnail,
		Width:             meta.Width,
		Height:            meta.Height,
		Duration:          meta.Duration,
		SupportsStreaming: true,
	}, cleanup, nil
}

func NewH264(filename string) (*tg.Video, func(), error) {
	converted, err := os.CreateTemp("", "kittenbark_tgmedia_*.mp4")
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to create temporary file: %w", err)
	}
	if err := convertH264(filename, converted); err != nil {
		_ = converted.Close()
		_ = os.Remove(converted.Name())
		return nil, func() {}, err
	}

	video, cleanup, err := New(converted.Name())
	wrappedCleanup := func() {
		defer cleanup()
		_ = converted.Close()
		_ = os.Remove(converted.Name())
	}
	return video, wrappedCleanup, nil
}

func send(ctx context.Context, chatId int64, filename string, name string, opts ...*tg.OptSendVideo) (*tg.Message, error) {
	thumbnailFile, err := os.CreateTemp("", "kittenbark_tgmedia_*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer time.AfterFunc(time.Second, func() {
		_ = thumbnailFile.Close()
		_ = os.Remove(thumbnailFile.Name())
	})
	thumbnail, err := buildThumbnail(filename, thumbnailFile)
	if err != nil {
		return nil, err
	}

	meta, err := getFileMetadata(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	opts = append(opts, &tg.OptSendVideo{
		Thumbnail:         thumbnail,
		Width:             meta.Width,
		Height:            meta.Height,
		Duration:          meta.Duration,
		SupportsStreaming: true,
	})
	return tg.SendVideo(ctx, chatId, tg.FromDisk(filename, name), opts...)
}

func convertH264(filename string, converted *os.File) error {
	convertCmd := exec.Command(
		Ffmpeg,
		"-y",
		"-i", filename,
		"-c:v", "libx264",
		"-preset", Preset,
		"-c:a", "aac",
		"-strict", "experimental",
		converted.Name(),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	convertCmd.Stdout = &stdout
	convertCmd.Stderr = &stderr
	if err := convertCmd.Run(); err != nil {
		return fmt.Errorf("failed to convert video to H264: %w (stdout: %s, stderr: %s)", err, stdout.String(), stderr.String())
	}
	return nil
}

type metadata struct {
	Width, Height int64
	Duration      int64
}

func getFileMetadata(filename string) (*metadata, error) {
	type fileMetadata struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
		Format struct {
			Filename string `json:"filename"`
			Duration string `json:"duration"`
		} `json:"format"`
	}

	output, err := exec.Command(Ffprobe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height",
		"-of", "json", "-show_format", filename).Output()
	if err != nil {
		return nil, fmt.Errorf("%v\n%s", err, string(output))
	}

	var ffprobeMetadata fileMetadata
	err = json.Unmarshal(output, &ffprobeMetadata)
	if err != nil {
		return nil, fmt.Errorf("%v\n%s", err, string(output))
	}

	result := &metadata{}
	duration, _ := strconv.ParseFloat(ffprobeMetadata.Format.Duration, 64)
	result.Duration = int64(duration)
	if len(ffprobeMetadata.Streams) > 0 {
		result.Width = int64(ffprobeMetadata.Streams[0].Width)
		result.Height = int64(ffprobeMetadata.Streams[0].Height)
	}
	return result, nil
}

func buildThumbnail(filename string, thumbnail *os.File) (tg.InputFile, error) {
	thumbnailCmd := exec.Command(
		Ffmpeg,
		"-y", "-i", filename,
		"-c:v", "mjpeg",
		"-pix_fmt", "yuvj420p",
		"-q:v", "2",
		"-vframes", "1",
		"-vf", "scale=if(gte(iw\\,ih)\\,min(320\\,iw)\\,-2):if(lt(iw\\,ih)\\,min(320\\,ih)\\,-2)",
		thumbnail.Name(),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	thumbnailCmd.Stdout = &stdout
	thumbnailCmd.Stderr = &stderr
	if err := thumbnailCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w (stdout: %s, stderr: %s)", err, stdout.String(), stderr.String())
	}
	return tg.FromDisk(thumbnail.Name()), nil
}
