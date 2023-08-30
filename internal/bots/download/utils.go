package download

import (
	"errors"
	"fmt"
	"github.com/cavaliergopher/grab/v3"
	"github.com/flowline-io/flowbot/internal/types"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func fileDownload(fullUrlFile string) (string, string, error) {
	fileUrl, err := url.Parse(fullUrlFile)
	if err != nil {
		return "", "", err
	}

	segments := strings.Split(fileUrl.Path, "/")
	originalFileName := segments[len(segments)-1]
	ext := filepath.Ext(originalFileName)
	if ext == "" {
		return "", "", errors.New("ext error")
	}
	downloadPath := os.Getenv("DOWNLOAD_PATH")
	if downloadPath == "" {
		return "", "", errors.New("download path error")
	}

	newFileName := fmt.Sprintf("%s%s", types.Id(), ext)
	fullDownloadFileName := fmt.Sprintf("%s/%s", downloadPath, newFileName)

	client := grab.NewClient()
	req, err := grab.NewRequest(fullDownloadFileName, fullUrlFile)
	if err != nil {
		return "", "", err
	}

	resp := client.Do(req)
	if resp == nil {
		return "", "", errors.New("download error")
	}
	if resp.HTTPResponse == nil {
		return "", "", errors.New("download error")
	}
	if resp.HTTPResponse.StatusCode != http.StatusOK {
		return "", "", errors.New(resp.HTTPResponse.Status)
	}

	return originalFileName, newFileName, resp.Err()
}
