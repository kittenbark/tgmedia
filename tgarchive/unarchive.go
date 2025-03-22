package tgarchive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/kittenbark/tg"
	"github.com/kittenbark/tgmedia/tgdir"
	"io"
	"os"
	"path"
	"path/filepath"
)

func SendUnpacked(ctx context.Context, chatId int64, filename string, opts ...*tgdir.Opt) ([]*tg.Message, error) {
	dir, err := unpack(filename)
	if err != nil {
		return nil, err
	}

	return tgdir.SendDocs(ctx, chatId, dir, opts...)
}

func unpack(filename string) (dir string, err error) {
	dir, err = os.MkdirTemp("", "tgmedia_*")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(dir)
		}
	}()

	switch filepath.Ext(filename) {
	case ".tar":
		return dir, unpackTar(filename, dir)
	case ".tar.gz":
		return dir, unpackTarGz(filename, dir)
	case ".zip":
		return dir, unpackZip(filename, dir)
	default:
		return "", errors.New("file type unsupported (.tar/.zip only)")
	}
}

func unpackZip(filename string, dir string) error {
	reader, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, fileZip := range reader.File {
		if fileZip.FileInfo().IsDir() {
			continue
		}

		unpackedPath := path.Join(dir, path.Clean(fileZip.Name))
		if err := os.MkdirAll(path.Dir(unpackedPath), os.ModePerm); err != nil {
			return err
		}

		fileData, err := fileZip.Open()
		if err != nil {
			return fmt.Errorf("bad file %s: %w (open)", fileZip.Name, err)
		}
		defer fileData.Close()

		fileUnpacked, err := os.Create(unpackedPath)
		if err != nil {
			return fmt.Errorf("bad file %s: %w (unpack create)", fileZip.Name, err)
		}
		defer fileUnpacked.Close()

		if _, err := io.Copy(fileUnpacked, fileData); err != nil {
			return fmt.Errorf("bad file %s: %w (unpack copy)", fileZip.Name, err)
		}
	}

	return nil
}

func unpackTar(filename string, dir string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := tar.NewReader(file)
	return unpackTarInternal(reader, dir)
}

func unpackTarGz(filename string, dir string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	uncompressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	reader := tar.NewReader(uncompressed)
	return unpackTarInternal(reader, dir)
}

func unpackTarInternal(reader *tar.Reader, dir string) error {
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if header.FileInfo().IsDir() {
			continue
		}

		unpackedPath := path.Join(dir, path.Clean(header.Name))
		if err := os.MkdirAll(path.Dir(unpackedPath), os.ModePerm); err != nil {
			return err
		}

		fileUnpacked, err := os.Create(unpackedPath)
		if err != nil {
			return fmt.Errorf("bad file %s: %w (unpack create)", header.Name, err)
		}
		defer fileUnpacked.Close()

		if _, err := io.Copy(fileUnpacked, reader); err != nil {
			return fmt.Errorf("bad file %s: %w (unpack copy)", header.Name, err)
		}
	}
}
