package workflowv3

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const maxItemKeyBytes = 256

// ItemManifest is the immutable ordered data plane for lazy map expansion.
// Workflow control rows persist only its ArtifactRef and per-item compact refs.
type ItemManifest struct {
	Schema     string         `json:"schema"`
	ItemSchema string         `json:"itemSchema"`
	Items      []ManifestItem `json:"items"`
}

type ManifestItem struct {
	Key   string      `json:"key"`
	Value ArtifactRef `json:"value"`
}

func NewItemManifest(itemSchema string, items []ManifestItem) (ItemManifest, error) {
	manifest := ItemManifest{
		Schema: ItemManifestSchemaV1, ItemSchema: itemSchema,
		Items: append([]ManifestItem{}, items...),
	}
	if err := ValidateItemManifest(manifest); err != nil {
		return ItemManifest{}, err
	}
	return manifest, nil
}

func ValidateItemManifest(manifest ItemManifest) error {
	if manifest.Schema != ItemManifestSchemaV1 {
		return fmt.Errorf("item manifest schema must be %q", ItemManifestSchemaV1)
	}
	if strings.TrimSpace(manifest.ItemSchema) == "" {
		return fmt.Errorf("item manifest item schema is required")
	}
	if manifest.Items == nil {
		return fmt.Errorf("item manifest items must be an array")
	}
	previous := ""
	for index, item := range manifest.Items {
		if err := validateItemKey(item.Key); err != nil {
			return fmt.Errorf("item manifest item %d: %w", index, err)
		}
		if index > 0 && item.Key <= previous {
			return fmt.Errorf("item manifest keys must be unique and strictly increasing")
		}
		if err := ValidateArtifactRef(item.Value); err != nil {
			return fmt.Errorf("item manifest item %q: %w", item.Key, err)
		}
		if item.Value.Schema != manifest.ItemSchema {
			return fmt.Errorf("item manifest item %q schema %q does not match %q", item.Key, item.Value.Schema, manifest.ItemSchema)
		}
		previous = item.Key
	}
	return nil
}

func EncodeItemManifest(manifest ItemManifest) ([]byte, error) {
	if err := ValidateItemManifest(manifest); err != nil {
		return nil, err
	}
	return CanonicalJSON(manifest)
}

func DecodeItemManifest(body []byte) (ItemManifest, error) {
	var manifest ItemManifest
	if err := StrictDecode(body, &manifest); err != nil {
		return ItemManifest{}, err
	}
	if err := ValidateItemManifest(manifest); err != nil {
		return ItemManifest{}, err
	}
	return manifest, nil
}

// MapChildNodeKey derives stable child identity without exposing source item
// keys in durable node identifiers.
func MapChildNodeKey(mapKey, sourceDigest, itemKey string) (NodeKey, error) {
	if strings.TrimSpace(mapKey) == "" {
		return "", fmt.Errorf("map key is required")
	}
	if err := validateSHA256Digest(sourceDigest); err != nil {
		return "", fmt.Errorf("map source digest: %w", err)
	}
	if err := validateItemKey(itemKey); err != nil {
		return "", err
	}
	digest, err := Digest(struct {
		MapKey       string `json:"mapKey"`
		SourceDigest string `json:"sourceDigest"`
		ItemKey      string `json:"itemKey"`
	}{MapKey: mapKey, SourceDigest: sourceDigest, ItemKey: itemKey})
	if err != nil {
		return "", err
	}
	return NodeKey("map:" + strings.TrimPrefix(digest, "sha256:")), nil
}

func validateItemKey(key string) error {
	if strings.TrimSpace(key) == "" || key != strings.TrimSpace(key) {
		return fmt.Errorf("item key is required without surrounding whitespace")
	}
	if !utf8.ValidString(key) {
		return fmt.Errorf("item key must be valid UTF-8")
	}
	if len(key) > maxItemKeyBytes {
		return fmt.Errorf("item key exceeds %d bytes", maxItemKeyBytes)
	}
	for _, r := range key {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("item key contains a control character")
		}
	}
	return nil
}
