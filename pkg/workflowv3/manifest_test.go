package workflowv3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func manifestRef(schema, digestByte, locator string) ArtifactRef {
	return ArtifactRef{
		Schema: schema, Digest: "sha256:" + strings.Repeat(digestByte, 64),
		MediaType: "application/json", Size: 42, Locator: locator,
	}
}

func TestItemManifestCanonicalRoundTrip(t *testing.T) {
	manifest, err := NewItemManifest("customer/v1", []ManifestItem{
		{Key: "customer-0001", Value: manifestRef("customer/v1", "a", "cas://customer-0001")},
		{Key: "customer-0002", Value: manifestRef("customer/v1", "b", "cas://customer-0002")},
	})
	require.NoError(t, err)
	body, err := EncodeItemManifest(manifest)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"schema":"scraper-workflow-item-manifest/v1",
		"itemSchema":"customer/v1",
		"items":[
			{"key":"customer-0001","value":{"schema":"customer/v1","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","mediaType":"application/json","size":42,"locator":"cas://customer-0001"}},
			{"key":"customer-0002","value":{"schema":"customer/v1","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","mediaType":"application/json","size":42,"locator":"cas://customer-0002"}}
		]
	}`, string(body))
	decoded, err := DecodeItemManifest(body)
	require.NoError(t, err)
	require.Equal(t, manifest, decoded)
}

func TestItemManifestRejectsNonCanonicalOrUnsafeItems(t *testing.T) {
	tests := []struct {
		name  string
		items []ManifestItem
		want  string
	}{
		{
			name: "duplicate", want: "unique and strictly increasing",
			items: []ManifestItem{
				{Key: "same", Value: manifestRef("customer/v1", "a", "cas://one")},
				{Key: "same", Value: manifestRef("customer/v1", "b", "cas://two")},
			},
		},
		{
			name: "unsorted", want: "unique and strictly increasing",
			items: []ManifestItem{
				{Key: "z", Value: manifestRef("customer/v1", "a", "cas://one")},
				{Key: "a", Value: manifestRef("customer/v1", "b", "cas://two")},
			},
		},
		{
			name: "wrong schema", want: "does not match",
			items: []ManifestItem{{Key: "one", Value: manifestRef("wrong/v1", "a", "cas://one")}},
		},
		{
			name: "control", want: "control character",
			items: []ManifestItem{{Key: "bad\nkey", Value: manifestRef("customer/v1", "a", "cas://one")}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewItemManifest("customer/v1", test.items)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestDecodeItemManifestRejectsUnknownFieldsAndNullItems(t *testing.T) {
	_, err := DecodeItemManifest([]byte(`{"schema":"scraper-workflow-item-manifest/v1","itemSchema":"x/v1","items":[],"unknown":true}`))
	require.ErrorContains(t, err, "unknown field")
	_, err = DecodeItemManifest([]byte(`{"schema":"scraper-workflow-item-manifest/v1","itemSchema":"x/v1","items":null}`))
	require.ErrorContains(t, err, "must be an array")
}

func TestMapChildNodeKeyIsExactStableAndOpaque(t *testing.T) {
	_, err := MapChildNodeKey("normalize", "sha256:"+strings.Repeat("z", 64), "customer-0001")
	require.ErrorContains(t, err, "sha256 hex")

	sourceDigest := "sha256:" + strings.Repeat("b", 64)
	first, err := MapChildNodeKey("normalize", sourceDigest, "customer-0001")
	require.NoError(t, err)
	second, err := MapChildNodeKey("normalize", sourceDigest, "customer-0001")
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, NodeKey("map:762fdd8fbf74c3fab9d50ead448fcdb7ba311e5cbfe061f2677ff94deee84552"), first)
	require.NotContains(t, string(first), "customer-0001")

	other, err := MapChildNodeKey("normalize", sourceDigest, "customer-0002")
	require.NoError(t, err)
	require.NotEqual(t, first, other)
}
