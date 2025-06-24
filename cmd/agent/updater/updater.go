package updater

import (
	"crypto/sha256"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"github.com/minio/selfupdate"
	"github.com/samber/lo"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

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

	asset, ok := lo.Find(release.Assets, func(item *github.ReleaseAsset) bool {
		return *item.Name == execName()
	})
	if !ok || asset == nil {
		return false, nil
	}

	flog.Info("Downloading latest version...")
	filename := execName() + ".tmp"
	err = utils.DownloadFile(*asset.BrowserDownloadURL, filename)
	if err != nil {
		return false, err
	}

	flog.Info("Verifying checksum...")
	checksumAsset, ok := lo.Find(release.Assets, func(asset *github.ReleaseAsset) bool {
		return *asset.Name == checksumsName()
	})
	if !ok || checksumAsset == nil {
		return false, nil
	}

	resp, err := http.Get(*checksumAsset.BrowserDownloadURL)
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

func GetLatestRelease() (*github.RepositoryRelease, error) {
	client := github.NewGithub("", "", "", "")
	releases, err := client.GetReleases("flowline-io", "flowbot", 1, 1)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	return releases[0], nil
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
