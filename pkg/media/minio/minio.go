// Package minio implements media interface by storing media objects in Minio bucket.
package minio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	appConfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/media"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	defaultServeURL = "/v0/file/s/"
	handlerName     = "minio"
)

type config struct {
	AccessKeyId     string   `json:"access_key_id"`
	SecretAccessKey string   `json:"secret_access_key"`
	Region          string   `json:"region"`
	DisableSSL      bool     `json:"disable_ssl"`
	ForcePathStyle  bool     `json:"force_path_style"`
	Endpoint        string   `json:"endpoint"`
	BucketName      string   `json:"bucket"`
	CorsOrigins     []string `json:"cors_origins"`
	ServeURL        string   `json:"serve_url"`
}

type handler struct {
	svc  *minio.Client
	conf config
}

// Init initializes the media handler.
func (ah *handler) Init(jsconf string) error {
	var err error
	if err = sonic.Unmarshal([]byte(jsconf), &ah.conf); err != nil {
		return fmt.Errorf("error parsing config: %s", jsconf)
	}

	if ah.conf.AccessKeyId == "" {
		return errors.New("missing Access Key ID")
	}
	if ah.conf.SecretAccessKey == "" {
		return errors.New("missing Secret Access Key")
	}
	if ah.conf.BucketName == "" {
		return errors.New("missing Bucket")
	}

	if ah.conf.ServeURL == "" {
		ah.conf.ServeURL = defaultServeURL
	}

	minioClient, err := minio.New(ah.conf.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(ah.conf.AccessKeyId, ah.conf.SecretAccessKey, ""),
		Secure: !ah.conf.DisableSSL,
	})
	if err != nil {
		return fmt.Errorf("error connecting to minio, %w", err)
	}
	ah.svc = minioClient

	ctx := context.Background()
	exist, err := ah.svc.BucketExists(ctx, ah.conf.BucketName)
	if err != nil {
		return fmt.Errorf("error checking if bucket %s exists, %w", ah.conf.BucketName, err)
	}
	if !exist {
		err = ah.svc.MakeBucket(ctx, ah.conf.BucketName, minio.MakeBucketOptions{Region: ah.conf.Region})
		if err != nil {
			return fmt.Errorf("error creating bucket %s, %w", ah.conf.BucketName, err)
		}
	}
	return nil
}

// Headers redirects GET, HEAD requests to the AWS server.
func (ah *handler) Headers(req *http.Request, serve bool) (http.Header, int, error) {
	header, status := media.CORSHandler(req, ah.conf.CorsOrigins, serve)
	return header, status, nil
}

// Upload processes request for a file upload. The file is given as io.Reader.
func (ah *handler) Upload(fdef *types.FileDef, file io.ReadSeeker) (string, int64, error) {
	var err error

	if fdef.Id == "" {
		fdef.Id = types.Id()
	}

	if fdef.Location == "" {
		fdef.Location = "/"
	}

	if fdef.MimeType == "" {
		fdef.MimeType = "application/octet-stream"
	}

	if fdef.Size == 0 {
		return "", 0, errors.New("empty file")
	}
	if fdef.Size > appConfig.App.Media.MaxFileUploadSize {
		return "", 0, fmt.Errorf("error max file upload size, %d > %d", fdef.Size, appConfig.App.Media.MaxFileUploadSize)
	}

	fname := strings.TrimRight(fdef.Location, "/") + "/" + fdef.Id
	ext, _ := mime.ExtensionsByType(fdef.MimeType)
	if len(ext) > 0 {
		fname += ext[len(ext)-1]
	}
	fdef.Location = fname

	if err = store.Database.FileStartUpload(fdef); err != nil {
		flog.Warn("failed to create file record %v %v", fdef.Id, err)
		return "", 0, fmt.Errorf("failed to create file record %v, %w", fdef.Id, err)
	}

	info, err := ah.svc.PutObject(context.Background(), ah.conf.BucketName, fname, file, fdef.Size, minio.PutObjectOptions{
		ContentType: fdef.MimeType,
	})
	if err != nil {
		if _, finishErr := store.Database.FileFinishUpload(fdef, false, 0); finishErr != nil {
			flog.Warn("failed to update file record %v %v", fdef.Id, finishErr)
		}
		return "", 0, fmt.Errorf("error uploading file %s, %w", fname, err)
	}

	if _, err = store.Database.FileFinishUpload(fdef, true, info.Size); err != nil {
		flog.Warn("failed to update file record %v %v", fdef.Id, err)
		return "", 0, fmt.Errorf("failed to update file record %v, %w", fdef.Id, err)
	}

	presignedURL, err := ah.presignedURL(fdef)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get presigned url, %w", err)
	}

	return presignedURL, info.Size, nil
}

// Download processes request for file download.
// The returned ReadSeekCloser must be closed after use.
func (ah *handler) Download(fUrl string) (*types.FileDef, media.ReadSeekCloser, error) {
	fid := ah.GetIdFromUrl(fUrl)
	if fid.IsZero() {
		return nil, nil, protocol.ErrNotFound.New("fid not found")
	}

	fd, err := store.Database.FileGet(fid.String())
	if err != nil {
		return nil, nil, fmt.Errorf("file not found %v, %w", fid, err)
	}
	if fd == nil {
		return nil, nil, protocol.ErrNotFound.New("fid not found")
	}

	obj, err := ah.svc.GetObject(context.Background(), ah.conf.BucketName, fd.Location, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download file %v from minio, %w", fd.Location, err)
	}

	return fd, obj, nil
}

// Delete deletes files from minio by provided slice of locations.
func (ah *handler) Delete(locations []string) error {
	objectsCh := make(chan minio.ObjectInfo)

	// Send object names that are needed to be removed to objectsCh
	go func() {
		defer close(objectsCh)
		for _, location := range locations {
			objectsCh <- minio.ObjectInfo{Key: location}
		}
	}()

	// Call RemoveObjects API
	errorCh := ah.svc.RemoveObjects(context.Background(), ah.conf.BucketName, objectsCh, minio.RemoveObjectsOptions{})

	// Collect all errors from RemoveObjects API
	var errs []error
	for e := range errorCh {
		flog.Warn("failed to remove %s: %v", e.ObjectName, e.Err)
		errs = append(errs, fmt.Errorf("failed to remove %s: %w", e.ObjectName, e.Err))
	}

	return errors.Join(errs...)
}

// GetIdFromUrl converts an attahment URL to a file UID.
func (ah *handler) GetIdFromUrl(fUrl string) types.Uid {
	return media.GetIdFromUrl(fUrl, ah.conf.ServeURL)
}

func (ah *handler) presignedURL(fdef *types.FileDef) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=\"%s\"", fdef.Name))

	presignedURL, err := ah.svc.PresignedGetObject(context.Background(), ah.conf.BucketName, fdef.Location, time.Hour, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned url, %w", err)
	}

	return presignedURL.String(), nil
}

func Register() {
	store.RegisterMediaHandler(handlerName, &handler{})
}
