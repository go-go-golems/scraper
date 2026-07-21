package workflowv3

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func CanonicalJSON(value any) ([]byte, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical JSON: %w", err)
	}
	return body, nil
}

func Digest(value any) (string, error) {
	body, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func StrictDecode(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode strict JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode strict JSON: trailing value")
		}
		return fmt.Errorf("decode strict JSON trailing value: %w", err)
	}
	return nil
}

func ValidateArtifactRef(ref ArtifactRef) error {
	if strings.TrimSpace(ref.Schema) == "" {
		return fmt.Errorf("artifact schema is required")
	}
	if !strings.HasPrefix(ref.Digest, "sha256:") || len(ref.Digest) != 71 {
		return fmt.Errorf("artifact digest must be sha256 hex")
	}
	if strings.TrimSpace(ref.MediaType) == "" {
		return fmt.Errorf("artifact media type is required")
	}
	if ref.Size < 0 {
		return fmt.Errorf("artifact size cannot be negative")
	}
	if strings.TrimSpace(ref.Locator) == "" {
		return fmt.Errorf("artifact locator is required")
	}
	if len(ref.Locator) > 256 {
		return fmt.Errorf("artifact locator exceeds 256 bytes")
	}
	return nil
}
