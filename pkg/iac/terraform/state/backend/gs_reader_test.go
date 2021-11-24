package backend

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"cloud.google.com/go/storage"
	googletest "github.com/cloudskiff/driftctl/test/google"
	"github.com/stretchr/testify/assert"
)

func TestGSBackend_Read(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name        string
		args        args
		wantErr     error
		handlerFunc func(http.ResponseWriter, *http.Request)
		expected    string
	}{
		{
			name: "Should fail with bad status code",
			args: args{
				path: "bucket-1/terraform.tfstate",
			},
			wantErr: errors.New("error requesting HTTP(s) backend state: status code: 404"),
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/bucket-1/terraform.tfstate", r.RequestURI)

				_, _ = w.Write([]byte("test"))
			},
			expected: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := googletest.NewFakeStorageServer(tt.handlerFunc)
			if err != nil {
				t.Fatal(err)
			}

			reader, err := NewGSReader(client, tt.args.path)
			assert.NoError(t, err)

			got := make([]byte, len(tt.expected))
			_, err = reader.Read(got)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			} else {
				assert.NoError(t, err)
			}
			assert.NotNil(t, got)
			assert.Equal(t, tt.expected, string(got))
		})
	}
}

func TestGSBackend_Close(t *testing.T) {
	type fields struct {
		reader io.ReadCloser
		client *storage.Client
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should fail to close reader",
			fields: fields{
				reader: func() io.ReadCloser {
					return nil
				}(),
				client: &storage.Client{},
			},
			wantErr: true,
		},
		{
			name: "should close reader",
			fields: fields{
				reader: func() io.ReadCloser {
					m := &MockReaderMock{}
					m.On("Close").Return(nil)
					return m
				}(),
				client: &storage.Client{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &GSBackend{
				reader:   tt.fields.reader,
				GSClient: tt.fields.client,
			}
			if err := h.Close(); (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
