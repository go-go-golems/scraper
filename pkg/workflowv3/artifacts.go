package workflowv3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxArtifactBytes int64 = 64 << 20

type ArtifactStore interface {
	Put(context.Context, string, string, []byte) (ArtifactRef, error)
	Open(context.Context, ArtifactRef) (io.ReadCloser, error)
}

type FileArtifactStore struct {
	root     string
	maxBytes int64
}

func NewFileArtifactStore(root string, maxBytes int64) (*FileArtifactStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("artifact root is required")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxArtifactBytes
	}
	if err := os.MkdirAll(filepath.Join(root, "objects"), 0o755); err != nil {
		return nil, fmt.Errorf("create artifact root: %w", err)
	}
	return &FileArtifactStore{root: root, maxBytes: maxBytes}, nil
}

func (s *FileArtifactStore) Put(ctx context.Context, schema, mediaType string, body []byte) (ArtifactRef, error) {
	if err := ctx.Err(); err != nil {
		return ArtifactRef{}, err
	}
	if strings.TrimSpace(schema) == "" || strings.TrimSpace(mediaType) == "" {
		return ArtifactRef{}, fmt.Errorf("artifact schema and media type are required")
	}
	if int64(len(body)) > s.maxBytes {
		return ArtifactRef{}, fmt.Errorf("artifact size %d exceeds %d", len(body), s.maxBytes)
	}
	sum := sha256.Sum256(body)
	hexDigest := hex.EncodeToString(sum[:])
	locator := filepath.ToSlash(filepath.Join("objects", hexDigest))
	target := filepath.Join(s.root, filepath.FromSlash(locator))
	if _, err := os.Stat(target); err == nil {
		return ArtifactRef{Schema: schema, Digest: "sha256:" + hexDigest, MediaType: mediaType, Size: int64(len(body)), Locator: locator}, nil
	} else if !os.IsNotExist(err) {
		return ArtifactRef{}, fmt.Errorf("stat artifact: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".artifact-*")
	if err != nil {
		return ArtifactRef{}, fmt.Errorf("create artifact temporary file: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(body); err != nil {
		_ = temporary.Close()
		return ArtifactRef{}, fmt.Errorf("write artifact: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return ArtifactRef{}, fmt.Errorf("sync artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return ArtifactRef{}, fmt.Errorf("close artifact: %w", err)
	}
	if err := os.Rename(temporaryName, target); err != nil {
		if _, statErr := os.Stat(target); statErr != nil {
			return ArtifactRef{}, fmt.Errorf("publish artifact: %w", err)
		}
	}
	return ArtifactRef{Schema: schema, Digest: "sha256:" + hexDigest, MediaType: mediaType, Size: int64(len(body)), Locator: locator}, nil
}

func (s *FileArtifactStore) PutJSON(ctx context.Context, schema string, value any) (ArtifactRef, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return ArtifactRef{}, fmt.Errorf("marshal artifact JSON: %w", err)
	}
	return s.Put(ctx, schema, "application/json", body)
}

func (s *FileArtifactStore) Open(ctx context.Context, ref ArtifactRef) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateArtifactRef(ref); err != nil {
		return nil, err
	}
	clean := filepath.Clean(filepath.FromSlash(ref.Locator))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("artifact locator is outside the store")
	}
	body, err := os.ReadFile(filepath.Join(s.root, clean))
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}
	if int64(len(body)) != ref.Size {
		return nil, fmt.Errorf("artifact size mismatch: got %d want %d", len(body), ref.Size)
	}
	sum := sha256.Sum256(body)
	actual := "sha256:" + hex.EncodeToString(sum[:])
	if actual != ref.Digest {
		return nil, fmt.Errorf("artifact digest mismatch: got %s want %s", actual, ref.Digest)
	}
	return io.NopCloser(bytes.NewReader(body)), nil
}

func ReadArtifact(ctx context.Context, store ArtifactStore, ref ArtifactRef) ([]byte, error) {
	reader, err := store.Open(ctx, ref)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read artifact body: %w", err)
	}
	return body, nil
}
