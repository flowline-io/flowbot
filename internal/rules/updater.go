package rules

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/github"
	"github.com/flowline-io/flowbot/pkg/rdb"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	rulesCurrentVersionKey = "rule_engine:rules:current_version"
)

func Updater(ctx context.Context) error {
	release, err := GetLatestRelease()
	if err != nil {
		return err
	}
	flog.Info("Latest rules release version: %v", *release.TagName)

	needsUpdate, _, err := CheckUpdates(ctx, *release.TagName)
	if err != nil {
		return err
	}
	if !needsUpdate {
		// Already up to date
		return nil
	}

	// Find the release asset named "rules.tar.gz"
	asset, ok := lo.Find(release.Assets, func(item *github.ReleaseAsset) bool {
		return *item.Name == "rules.tar.gz"
	})
	if !ok || asset == nil {
		flog.Warn("Asset rules.tar.gz not found in release assets")
		return nil
	}

	flog.Info("Downloading latest version...")

	// Create a temporary download file
	filename := "rules.tar.gz.tmp"
	err = utils.DownloadFile(*(*asset).BrowserDownloadURL, filename)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Ensure temporary file is deleted after download
	defer os.Remove(filename)

	// Target directory for rules
	rulesDir := config.App.RuleEngine.RulesPath

	// Create a temporary extraction directory
	tempExtractDir := rulesDir + "_new"
	if err := os.MkdirAll(tempExtractDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Extract file to temporary directory
	if err := extractTarGz(filename, tempExtractDir); err != nil {
		_ = os.RemoveAll(tempExtractDir) // Clean up temp directory
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Backup original directory (add .bak suffix)
	backupDir := rulesDir + ".bak"
	_ = os.RemoveAll(backupDir) // Clean up old backup
	if err := os.Rename(rulesDir, backupDir); err != nil && !os.IsNotExist(err) {
		_ = os.RemoveAll(tempExtractDir)
		return fmt.Errorf("backup failed: %w", err)
	}

	// Atomic directory replacement
	if err := os.Rename(tempExtractDir, rulesDir); err != nil {
		// Attempt to restore backup
		if renameErr := os.Rename(backupDir, rulesDir); renameErr != nil {
			flog.Error(fmt.Errorf("CRITICAL: Failed to restore backup after update failure: %v", renameErr))
		}
		return fmt.Errorf("directory replacement failed: %w", err)
	}

	// Clean up backup directory
	_ = os.RemoveAll(backupDir)

	flog.Info("Rules updated successfully to version %s", *release.TagName)
	return nil
}

func CheckUpdates(ctx context.Context, releaseTag string) (bool, string, error) {
	latestVersion, err := semver.NewVersion(releaseTag)
	if err != nil {
		return false, "", err
	}

	cu, err := rdb.Client.Get(ctx, rulesCurrentVersionKey).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return false, "", err
		}
	}
	if cu == "" {
		cu = "0.0.0"
	}

	currentVersion, err := semver.NewVersion(cu)
	if err != nil {
		return false, "", err
	}

	needsUpdate := currentVersion.LessThan(latestVersion)

	return needsUpdate, releaseTag, nil
}

func GetLatestRelease() (*github.RepositoryRelease, error) {
	// get owner/repo from github url
	regex := `github.com/([^/]+)/([^/]+)`
	r := regexp.MustCompile(regex)
	result := r.FindStringSubmatch(config.App.RuleEngine.GithubRulesRepo)
	if len(result) != 3 {
		return nil, fmt.Errorf("invalid github repo url: %s", config.App.RuleEngine.GithubRulesRepo)
	}
	owner := result[1]
	repo := result[2]

	client := github.NewGithub("", "", "", config.App.RuleEngine.GithubReleaseAccessToken)
	releases, err := client.GetReleases(owner, repo, 1, 1)
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	return releases[0], nil
}

// a utility function to safely extract a tar.gz file
func extractTarGz(src string, dest string) error {
	// Open the compressed file
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)
	baseDest, _ := filepath.Abs(dest) // used for security check
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Prevent path traversal attacks
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, baseDest) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}

		// Handle files according to type
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// Ensure the directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			// Create the file
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// Copy the contents
			if _, err := io.Copy(f, tr); err != nil {
				_ = f.Close()
				return err
			}
			_ = f.Close()
		default:
			flog.Info("skipped unsupported file type %v in %s", header.Typeflag, header.Name)
		}
	}
	return nil
}
