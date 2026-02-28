// Package fs implements github.com/flowline-io/flowbot/media interface by storing media objects in a single
// directory in the file system.
// This module won't perform well with tens of thousand of files because it stores all files in a single directory.
package fs

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	appConfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

const (
	defaultServeURL = "/v0/file/s/"
	handlerName     = "fs"
)

type configType struct {
	FileUploadDirectory string   `json:"upload_dir"`
	ServeURL            string   `json:"serve_url"`
	CorsOrigins         []string `json:"cors_origins"`
}

type fshandler struct {
	// In case of a cluster fileUploadLocation must be accessible to all cluster members.
	fileUploadLocation string
	serveURL           string
	corsOrigins        []string
}

func (fh *fshandler) Init(jsconf string) error {
	var err error
	var config configType

	if err = sonic.Unmarshal([]byte(jsconf), &config); err != nil {
		return fmt.Errorf("error parsing config: %s, %w", jsconf, err)
	}

	fh.fileUploadLocation = config.FileUploadDirectory
	if fh.fileUploadLocation == "" {
		return errors.New("missing upload location")
	}

	fh.serveURL = config.ServeURL
	if fh.serveURL == "" {
		fh.serveURL = defaultServeURL
	}

	// Make sure the upload directory exists.
	return os.MkdirAll(fh.fileUploadLocation, 0750)
}

// Headers is used for serving CORS headers.
func (fh *fshandler) Headers(req *http.Request, serve bool) (http.Header, int, error) {
	header, status := media.CORSHandler(req, fh.corsOrigins, serve)
	return header, status, nil
}

// Upload processes request for file upload. The file is given as io.Reader.
func (fh *fshandler) Upload(fdef *types.FileDef, file io.ReadSeeker) (string, int64, error) {
	// Ensure file ID is set.
	if fdef.Id == "" {
		fdef.Id = types.Id()
	}

	// FIXME: create two-three levels of nested directories. Serving from a single directory
	// with tens of thousands of files in it will not perform well.

	// Generate a unique file name and attach it to path. Using base32 instead of base64 to avoid possible
	// file name collisions on Windows due to case-insensitive file names there.
	fdef.Location = filepath.Join(fh.fileUploadLocation, fdef.Uid().String())

	if fdef.Size > appConfig.App.Media.MaxFileUploadSize {
		return "", 0, fmt.Errorf("error max file upload size, %d > %d", fdef.Size, appConfig.App.Media.MaxFileUploadSize)
	}

	outfile, err := os.Create(fdef.Location)
	if err != nil {
		flog.Warn("Upload: failed to create file %v %v", fdef.Location, err)
		return "", 0, fmt.Errorf("failed to create file %v, %w", fdef.Location, err)
	}

	if err = store.Database.FileStartUpload(fdef); err != nil {
		_ = outfile.Close()
		_ = os.Remove(fdef.Location)
		flog.Warn("failed to create file record %v %v", fdef.Id, err)
		return "", 0, fmt.Errorf("failed to create file record %v, %w", fdef.Id, err)
	}

	size, err := io.Copy(outfile, file)
	_ = outfile.Close()
	if err != nil {
		_ = os.Remove(fdef.Location)
		if _, finishErr := store.Database.FileFinishUpload(fdef, false, 0); finishErr != nil {
			flog.Warn("failed to update file record %v %v", fdef.Id, finishErr)
		}
		return "", 0, fmt.Errorf("failed to upload file %v, %w", fdef.Location, err)
	}

	if _, err = store.Database.FileFinishUpload(fdef, true, size); err != nil {
		flog.Warn("failed to update file record %v %v", fdef.Id, err)
		return "", 0, fmt.Errorf("failed to update file record %v, %w", fdef.Id, err)
	}

	fname := fdef.Id
	ext, _ := mime.ExtensionsByType(fdef.MimeType)
	if len(ext) > 0 {
		fname += ext[0]
	}

	return fh.serveURL + fname, size, nil
}

// Download processes request for file download.
// The returned ReadSeekCloser must be closed after use.
func (fh *fshandler) Download(url string) (*types.FileDef, media.ReadSeekCloser, error) {
	fid := fh.GetIdFromUrl(url)
	if fid.IsZero() {
		return nil, nil, protocol.ErrNotFound.New("fid not found")
	}

	fd, err := fh.getFileRecord(fid)
	if err != nil {
		flog.Warn("Download: file not found %v", fid)
		return nil, nil, fmt.Errorf("file not found %v, %w", fid, err)
	}

	file, err := os.Open(fd.Location)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file is not found, send 404 instead of the default 500
			err = protocol.ErrNotFound.New("file not found")
		}
		return nil, nil, fmt.Errorf("failed to open file %v, %w", fd.Location, err)
	}

	return fd, file, nil
}

// Delete deletes files from storage by provided slice of locations.
func (fh *fshandler) Delete(locations []string) error {
	var errs []error
	for _, loc := range locations {
		err := os.Remove(loc)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// File already gone, not an error.
				continue
			}
			flog.Warn("fs: error deleting file %v %v", loc, err)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// GetIdFromUrl converts an attahment URL to a file UID.
func (fh *fshandler) GetIdFromUrl(url string) types.Uid {
	return media.GetIdFromUrl(url, fh.serveURL)
}

// getFileRecord given file ID reads file record from the database.
func (fh *fshandler) getFileRecord(fid types.Uid) (*types.FileDef, error) {
	fd, err := store.Database.FileGet(fid.String())
	if err != nil {
		return nil, fmt.Errorf("file not found %v, %w", fid, err)
	}
	if fd == nil {
		return nil, protocol.ErrNotFound.New("fid not found")
	}
	return fd, nil
}

func Register() {
	store.RegisterMediaHandler(handlerName, &fshandler{})
}
