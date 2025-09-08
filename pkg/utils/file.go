package utils

import (
	"io"
	"net/http"
	"os"

	"github.com/schollz/progressbar/v3"
)

func FileExist(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func DownloadFile(url, filename string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		return err
	}

	return nil
}
