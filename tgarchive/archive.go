package tgarchive

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"github.com/kittenbark/tg"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

func SendBy20MB(ctx context.Context, chatId int64, dir string, filename string, opt ...*tg.OptSendDocument) ([]*tg.Message, error) {
	return SendByN(ctx, chatId, dir, filename, 20_000_000, opt...)
}

func SendBy2GB(ctx context.Context, chatId int64, dir string, filename string, opt ...*tg.OptSendDocument) ([]*tg.Message, error) {
	return SendByN(ctx, chatId, dir, filename, 2_000_000_000, opt...)
}

func SendByN(ctx context.Context, chatId int64, dir string, filename string, n int64, opt ...*tg.OptSendDocument) (messages []*tg.Message, err error) {
	filename = strings.TrimSuffix(filename, ".tar")
	tmpdir, err := os.MkdirTemp("", "tgarchive_*")
	if err != nil {
		return nil, err
	}
	defer func(name string) {
		if e := os.RemoveAll(name); e != nil && err == nil {
			err = e
		}
	}(tmpdir)

	files := []*os.File{}
	file, err := os.OpenFile(path.Join(tmpdir, fmt.Sprintf("%s.tar", filename)), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return nil, err
	}
	files = append(files, file)
	defer func() {
		for _, f := range files {
			_ = f.Close()
			if e := os.Remove(f.Name()); e != nil && err == nil {
				err = e
			}
		}
	}()

	tarWriters := []*tar.Writer{}
	defer func() {
		for _, w := range tarWriters {
			_ = w.Close()
		}
	}()
	tarWriter := tar.NewWriter(file)
	tarWriters = append(tarWriters, tarWriter)
	tarSize := int64(0)
	iteration := 1

	fsys := os.DirFS(dir)
	err = fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}

		if !d.IsDir() && !info.Mode().IsRegular() {
			return errors.New("tgarchive: cannot add non-regular file")
		}
		if info.Size() > n {
			return fmt.Errorf("tgarchive: file too large (%d > %d)", info.Size(), n)
		}

		if tarSize+info.Size() <= n {
			tarSize += info.Size()
			h, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			h.Name = name
			if d.IsDir() {
				h.Name += "/"
			}
			if err := tarWriter.WriteHeader(h); err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			f, err := fsys.Open(name)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tarWriter, f)
			return err
		}

		if err := tarWriter.Flush(); err != nil {
			return err
		}
		msg, err := tg.SendDocument(ctx, chatId, tg.FromDisk(file.Name()), opt...)
		if err != nil {
			return err
		}
		messages = append(messages, msg)

		iteration += 1
		file, err = os.OpenFile(path.Join(tmpdir, fmt.Sprintf("%s_%02d.tar", filename, iteration)), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			return err
		}
		files = append(files, file)

		tarWriter = tar.NewWriter(file)
		tarWriters = append(tarWriters, tarWriter)
		tarSize = 0
		return nil
	})

	if tarSize != 0 {
		msg, err := tg.SendDocument(ctx, chatId, tg.FromDisk(file.Name()), opt...)
		if err != nil {
			return messages, err
		}
		messages = append(messages, msg)
	}

	return messages, err
}
