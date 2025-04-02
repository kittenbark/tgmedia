package tgvideo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kittenbark/tg"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var (
	Ffprobe = "ffprobe"
	Ffmpeg  = "ffmpeg"
	Preset  = "medium"
)

func Send(ctx context.Context, chatId int64, filename string, opts ...*tg.OptSendVideo) (*tg.Message, error) {
	thumbnail, err := os.CreateTemp("", "kittenbark_tgmedia_*.jpg")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer time.AfterFunc(time.Second*5, func() {
		_ = thumbnail.Close()
		_ = os.Remove(thumbnail.Name())
	})

	thumbnailCmd := exec.Command(
		Ffmpeg,
		"-y", "-i", filename,
		"-c:v", "mjpeg",
		"-pix_fmt", "yuvj420p",
		"-q:v", "2",
		"-vframes", "1",
		thumbnail.Name(),
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	thumbnailCmd.Stdout = &stdout
	thumbnailCmd.Stderr = &stderr
	if err := thumbnailCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w (stdout: %s, stderr: %s)", err, stdout.String(), stderr.String())
	}

	meta, err := getFileMetadata(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	opts = append(opts, &tg.OptSendVideo{
		Thumbnail:         tg.FromDisk(thumbnail.Name()),
		Width:             meta.Width,
		Height:            meta.Height,
		Duration:          meta.Duration,
		SupportsStreaming: true,
	})
	return tg.SendVideo(ctx, chatId, tg.FromDisk(filename), opts...)
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
		return nil, fmt.Errorf("failed to convert video to H264: %w (stdout: %s, stderr: %s)", err, stdout.String(), stderr.String())
	}

	return Send(ctx, chatId, converted.Name(), opts...)
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
