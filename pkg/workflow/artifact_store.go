package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactObject is a blob that should be stored outside the engine result row.
type ArtifactObject struct {
	ID          string
	Name        string
	Kind        string
	ContentType string
	Metadata    map[string]string
	Body        []byte
}

// ArtifactRef points to an artifact stored by an ArtifactStore.
type ArtifactRef struct {
	ID          string            `json:"id"`
	URI         string            `json:"uri"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	ContentType string            `json:"contentType"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Size        int               `json:"size"`
}

// ArtifactStore stores large artifact bytes outside the engine DB.
type ArtifactStore interface {
	Put(ctx context.Context, artifact ArtifactObject) (ArtifactRef, error)
	Open(ctx context.Context, id string) (io.ReadCloser, ArtifactRef, error)
}

// FileArtifactStore stores artifacts under a local filesystem root. It is the
// first external artifact backend for embedded/local workflow runtimes.
type FileArtifactStore struct {
	root string
}

func NewFileArtifactStore(root string) *FileArtifactStore {
	return &FileArtifactStore{root: root}
}

func (s *FileArtifactStore) Put(ctx context.Context, artifact ArtifactObject) (ArtifactRef, error) {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return ArtifactRef{}, fmt.Errorf("file artifact store root is required")
	}
	id := strings.TrimSpace(artifact.ID)
	if id == "" {
		return ArtifactRef{}, fmt.Errorf("artifact id is required")
	}
	select {
	case <-ctx.Done():
		return ArtifactRef{}, ctx.Err()
	default:
	}
	path := filepath.Join(s.root, safeArtifactPath(id))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ArtifactRef{}, err
	}
	if err := os.WriteFile(path, artifact.Body, 0o644); err != nil {
		return ArtifactRef{}, err
	}
	ref := ArtifactRef{
		ID:          id,
		URI:         "file://" + path,
		Name:        artifact.Name,
		Kind:        artifact.Kind,
		ContentType: artifact.ContentType,
		Metadata:    cloneStringMap(artifact.Metadata),
		Size:        len(artifact.Body),
	}
	metaPath := path + ".json"
	if body, err := json.MarshalIndent(ref, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, body, 0o644)
	}
	return ref, nil
}

func (s *FileArtifactStore) Open(ctx context.Context, id string) (io.ReadCloser, ArtifactRef, error) {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return nil, ArtifactRef{}, fmt.Errorf("file artifact store root is required")
	}
	select {
	case <-ctx.Done():
		return nil, ArtifactRef{}, ctx.Err()
	default:
	}
	path := filepath.Join(s.root, safeArtifactPath(id))
	file, err := os.Open(path)
	if err != nil {
		return nil, ArtifactRef{}, err
	}
	ref := ArtifactRef{ID: id, URI: "file://" + path}
	if body, err := os.ReadFile(path + ".json"); err == nil {
		_ = json.Unmarshal(body, &ref)
	}
	return file, ref, nil
}

func safeArtifactPath(id string) string {
	id = strings.TrimSpace(id)
	id = strings.ReplaceAll(id, "..", "_")
	id = strings.TrimLeft(id, string(filepath.Separator))
	parts := strings.FieldsFunc(id, func(r rune) bool {
		switch r {
		case ':', '\\':
			return true
		default:
			return false
		}
	})
	for i, part := range parts {
		parts[i] = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
				return r
			}
			return '_'
		}, part)
	}
	if len(parts) == 0 {
		return "artifact"
	}
	return filepath.Join(parts...)
}
