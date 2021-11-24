package backend

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
)

const BackendKeyGS = "gs"

type GSBackend struct {
	bucketName string
	path       string
	reader     io.ReadCloser
	GSClient   *storage.Client
}

func NewGSReader(storageClient *storage.Client, path string) (*GSBackend, error) {
	bucketPath := strings.Split(path, "/")
	if len(bucketPath) < 2 {
		return nil, errors.Errorf("Unable to parse Google Storage path: %s. Must be BUCKET_NAME/PATH/TO/OBJECT", path)
	}
	bucketName := bucketPath[0]
	key := strings.Join(bucketPath[1:], "/")

	return &GSBackend{
		bucketName: bucketName,
		path:       key,
		GSClient:   storageClient,
	}, nil
}

func (s *GSBackend) Read(p []byte) (int, error) {
	if s.reader == nil {
		ctx := context.Background()
		rc, err := s.GSClient.Bucket(s.bucketName).Object(s.path).NewReader(ctx)
		if err != nil {
			return 0, fmt.Errorf("Object(%q).NewReader: %v", s.path, err)
		}
		s.reader = rc
	}
	return s.reader.Read(p)
}

func (s *GSBackend) Close() error {
	if err := s.GSClient.Close(); err != nil {
		return err
	}
	if s.reader != nil {
		return s.reader.Close()
	}
	return errors.New("Unable to close reader as nothing was opened")
}
