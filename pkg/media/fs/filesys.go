// Package fs implements github.com/flowline-io/flowbot/media interface by storing media objects in a single
// directory in the file system.
// This module won't perform well with tens of thousand of files because it stores all files in a single directory.
package fs

import (
	"encoding/json"
	"errors"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/flowline-io/flowbot/pkg/media"
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

	if err = json.Unmarshal([]byte(jsconf), &config); err != nil {
		return errors.New("failed to parse config: " + err.Error())
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
	return os.MkdirAll(fh.fileUploadLocation, 0777)
}

// Headers is used for serving CORS headers.
func (fh *fshandler) Headers(req *http.Request, serve bool) (http.Header, int, error) {
	header, status := media.CORSHandler(req, fh.corsOrigins, serve)
	return header, status, nil
}

// Upload processes request for file upload. The file is given as io.Reader.
func (fh *fshandler) Upload(fdef *types.FileDef, file io.ReadSeeker) (string, int64, error) {
	// FIXME: create two-three levels of nested directories. Serving from a single directory
	// with tens of thousands of files in it will not perform well.

	// Generate a unique file name and attach it to path. Using base32 instead of base64 to avoid possible
	// file name collisions on Windows due to case-insensitive file names there.
	fdef.Location = filepath.Join(fh.fileUploadLocation, fdef.Uid().String32())

	outfile, err := os.Create(fdef.Location)
	if err != nil {
		flog.Warn("Upload: failed to create file %v %v", fdef.Location, err)
		return "", 0, err
	}

	if err = store.Chatbot.FileStartUpload(fdef); err != nil {
		outfile.Close()
		os.Remove(fdef.Location)
		flog.Warn("failed to create file record %v %v", fdef.Id, err)
		return "", 0, err
	}

	size, err := io.Copy(outfile, file)
	outfile.Close()
	if err != nil {
		os.Remove(fdef.Location)
		return "", 0, err
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
		return nil, nil, types.ErrNotFound
	}

	fd, err := fh.getFileRecord(fid)
	if err != nil {
		flog.Warn("Download: file not found %v", fid)
		return nil, nil, err
	}

	file, err := os.Open(fd.Location)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file is not found, send 404 instead of the default 500
			err = types.ErrNotFound
		}
		return nil, nil, err
	}

	return fd, file, nil
}

// Delete deletes files from storage by provided slice of locations.
func (fh *fshandler) Delete(locations []string) error {
	for _, loc := range locations {
		if err, _ := os.Remove(loc).(*os.PathError); err != nil {
			if err != os.ErrNotExist {
				flog.Warn("fs: error deleting file %v %v", loc, err)
			}
		}
	}
	return nil
}

// GetIdFromUrl converts an attahment URL to a file UID.
func (fh *fshandler) GetIdFromUrl(url string) types.Uid {
	return media.GetIdFromUrl(url, fh.serveURL)
}

// getFileRecord given file ID reads file record from the database.
func (fh *fshandler) getFileRecord(fid types.Uid) (*types.FileDef, error) {
	fd, err := store.Chatbot.FileGet(fid.String())
	if err != nil {
		return nil, err
	}
	if fd == nil {
		return nil, types.ErrNotFound
	}
	return fd, nil
}

func init() {
	store.RegisterMediaHandler(handlerName, &fshandler{})
}
