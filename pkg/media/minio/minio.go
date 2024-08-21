// Package minio implements media interface by storing media objects in Minio bucket.
package minio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/internal/types/protocol"
	appConfig "github.com/flowline-io/flowbot/pkg/config"
	"github.com/flowline-io/flowbot/pkg/media"
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
	if err = json.Unmarshal([]byte(jsconf), &ah.conf); err != nil {
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

	if fdef.Size > appConfig.App.Media.MaxFileUploadSize {
		return "", 0, fmt.Errorf("error max file upload size, %d > %d", fdef.Size, appConfig.App.Media.MaxFileUploadSize)
	}

	fname := strings.TrimRight(fdef.Location, "/") + "/" + fdef.Id
	ext, _ := mime.ExtensionsByType(fdef.MimeType)
	if len(ext) > 0 {
		fname += ext[len(ext)-1]
	}

	info, err := ah.svc.PutObject(context.Background(), ah.conf.BucketName, fname, file, fdef.Size, minio.PutObjectOptions{
		ContentType: fdef.MimeType,
	})
	if err != nil {
		return "", 0, fmt.Errorf("error uploading file %s, %w", fname, err)
	}

	return ah.conf.ServeURL + fname, info.Size, nil
}

// Download processes request for file download.
// The returned ReadSeekCloser must be closed after use.
func (ah *handler) Download(_ string) (*types.FileDef, media.ReadSeekCloser, error) {
	return nil, nil, protocol.ErrUnsupported
}

// Delete deletes files from aws by provided slice of locations.
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

	// Print errors received from RemoveObjects API
	for e := range errorCh {
		return errors.New("Failed to remove " + e.ObjectName + ", error: " + e.Err.Error())
	}

	return nil
}

// GetIdFromUrl converts an attahment URL to a file UID.
func (ah *handler) GetIdFromUrl(url string) types.Uid {
	return media.GetIdFromUrl(url, ah.conf.ServeURL)
}

func init() {
	store.RegisterMediaHandler(handlerName, &handler{})
}
