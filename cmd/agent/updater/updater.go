package updater

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"github.com/google/go-github/v66/github"
	"github.com/minio/selfupdate"
)

var p *tea.Program

func CheckUpdates() (bool, string, error) {
	release, err := GetLatestRelease()
	if err != nil {
		return false, "", err
	}
	flog.Info("release latest version: %v", *release.TagName)

	latestVersion, err := semver.NewVersion(*release.TagName)
	if err != nil {
		return false, "", err
	}
	currentVersion, err := semver.NewVersion(version.Buildtags)
	if err != nil {
		return false, "", err
	}

	needsUpdate := currentVersion.LessThan(latestVersion)

	return needsUpdate, *release.TagName, nil
}

func UpdateSelf() (bool, error) {
	release, err := GetLatestRelease()
	if err != nil {
		return false, err
	}

	asset := utils.FindOne(release.Assets, func(asset **github.ReleaseAsset) bool {
		return *(*asset).Name == execName()
	})
	if asset == nil {
		return false, nil
	}

	flog.Info("Downloading latest version...")
	filename := execName() + ".tmp"
	err = DownloadFile(*(*asset).BrowserDownloadURL, filename)
	if err != nil {
		return false, err
	}

	flog.Info("Verifying checksum...")
	checksumAsset := utils.FindOne(release.Assets, func(asset **github.ReleaseAsset) bool {
		return *(*asset).Name == checksumsName()
	})
	if checksumAsset == nil {
		return false, nil
	}

	resp, err := http.Get(*(*checksumAsset).BrowserDownloadURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	checksumBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return false, err
	}
	if ok := findChecksum(string(checksumBytes), fmt.Sprintf("%x", h.Sum(nil))); !ok {
		return false, fmt.Errorf("checksum mismatch. expected: %s, got: %x", checksumBytes, h.Sum(nil))
	}
	_ = file.Close()

	flog.Info("Applying update...")
	file, err = os.Open(filename)
	if err != nil {
		return false, err
	}
	err = selfupdate.Apply(file, selfupdate.Options{})
	_ = file.Close()
	_ = os.Remove(filename)
	if err != nil {
		return false, err
	}

	return true, nil
}

func DownloadFile(url, filename string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	file, err := os.Create(filename)
	if err != nil {
		_, _ = fmt.Println("could not create file:", err)
		os.Exit(1)
	}
	defer file.Close()

	pw := &progressWriter{
		total:  int(res.ContentLength),
		file:   file,
		reader: res.Body,
		onProgress: func(ratio float64) {
			p.Send(progressMsg(ratio))
		},
	}

	m := model{
		pw:       pw,
		progress: progress.New(progress.WithDefaultGradient()),
	}
	// Start Bubble Tea
	p = tea.NewProgram(m)

	// Start the download
	go pw.Start()

	if _, err := p.Run(); err != nil {
		_, _ = fmt.Println("error running program:", err)
		os.Exit(1)
	}

	return nil
}

func GetLatestRelease() (*github.RepositoryRelease, error) {
	client := github.NewClient(nil)
	releases, _, err := client.Repositories.ListReleases(context.Background(), "flowline-io", "flowbot", nil)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	return releases[0], nil
}

type progressWriter struct {
	total      int
	downloaded int
	file       *os.File
	reader     io.Reader
	onProgress func(float64)
}

func (pw *progressWriter) Start() {
	// TeeReader calls pw.Write() each time a new response is received
	_, err := io.Copy(pw.file, io.TeeReader(pw.reader, pw))
	if err != nil {
		p.Send(progressErrMsg{err})
	}
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	pw.downloaded += len(p)
	if pw.total > 0 && pw.onProgress != nil {
		pw.onProgress(float64(pw.downloaded) / float64(pw.total))
	}
	return len(p), nil
}

func execName() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("flowbot-agent_%s_%s.exe", runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("flowbot-agent_%s_%s", runtime.GOOS, runtime.GOARCH)
}

func checksumsName() string {
	return "flowbot-agent_checksums.txt"
}

func findChecksum(text string, hash string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		arr := strings.Split(line, "  ")
		if len(arr) == 2 {
			if arr[0] == hash {
				return true
			}
		}
	}
	return false
}
