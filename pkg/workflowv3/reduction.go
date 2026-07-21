package workflowv3

import "fmt"

type reductionMemberIdentity struct {
	Key    string `json:"key"`
	Digest string `json:"digest"`
}

type ReductionPartition struct {
	Schema       string         `json:"schema"`
	ReduceKey    string         `json:"reduceKey"`
	SourceDigest string         `json:"sourceDigest"`
	Level        int            `json:"level"`
	Ordinal      int            `json:"ordinal"`
	ItemSchema   string         `json:"itemSchema"`
	Members      []ManifestItem `json:"members"`
}

func NewReductionPartition(
	reduceKey, sourceDigest, itemSchema string,
	level, ordinal, maxFanIn int,
	members []ManifestItem,
) (ReductionPartition, error) {
	partition := ReductionPartition{
		Schema: ReductionPartitionSchemaV1, ReduceKey: reduceKey,
		SourceDigest: sourceDigest, Level: level, Ordinal: ordinal,
		ItemSchema: itemSchema, Members: append([]ManifestItem{}, members...),
	}
	if err := ValidateReductionPartition(partition, maxFanIn); err != nil {
		return ReductionPartition{}, err
	}
	return partition, nil
}

func ValidateReductionPartition(partition ReductionPartition, maxFanIn int) error {
	if partition.Schema != ReductionPartitionSchemaV1 {
		return fmt.Errorf("reduction partition schema must be %q", ReductionPartitionSchemaV1)
	}
	if partition.ReduceKey == "" {
		return fmt.Errorf("reduction key is required")
	}
	if err := validateSHA256Digest(partition.SourceDigest); err != nil {
		return fmt.Errorf("reduction source digest: %w", err)
	}
	if partition.Level < 0 || partition.Ordinal < 0 {
		return fmt.Errorf("reduction level and ordinal cannot be negative")
	}
	if partition.ItemSchema == "" {
		return fmt.Errorf("reduction item schema is required")
	}
	if maxFanIn < 1 || len(partition.Members) < 1 || len(partition.Members) > maxFanIn {
		return fmt.Errorf("reduction partition member count %d exceeds fan-in %d", len(partition.Members), maxFanIn)
	}
	previous := ""
	for index, member := range partition.Members {
		if err := validateItemKey(member.Key); err != nil {
			return fmt.Errorf("reduction member %d: %w", index, err)
		}
		if index > 0 && member.Key <= previous {
			return fmt.Errorf("reduction member keys must be unique and strictly increasing")
		}
		if err := ValidateArtifactRef(member.Value); err != nil {
			return fmt.Errorf("reduction member %q: %w", member.Key, err)
		}
		if member.Value.Schema != partition.ItemSchema {
			return fmt.Errorf("reduction member %q schema %q does not match %q", member.Key, member.Value.Schema, partition.ItemSchema)
		}
		previous = member.Key
	}
	return nil
}

func EncodeReductionPartition(partition ReductionPartition, maxFanIn int) ([]byte, error) {
	if err := ValidateReductionPartition(partition, maxFanIn); err != nil {
		return nil, err
	}
	return CanonicalJSON(partition)
}

func DecodeReductionPartition(body []byte, maxFanIn int) (ReductionPartition, error) {
	var partition ReductionPartition
	if err := StrictDecode(body, &partition); err != nil {
		return ReductionPartition{}, err
	}
	if err := ValidateReductionPartition(partition, maxFanIn); err != nil {
		return ReductionPartition{}, err
	}
	return partition, nil
}

func ReductionPartitionNodeKey(partition ReductionPartition, maxFanIn int) (NodeKey, error) {
	if err := ValidateReductionPartition(partition, maxFanIn); err != nil {
		return "", err
	}
	members := make([]reductionMemberIdentity, len(partition.Members))
	for index, member := range partition.Members {
		members[index].Key = member.Key
		members[index].Digest = member.Value.Digest
	}
	digest, err := Digest(struct {
		ReduceKey    string                    `json:"reduceKey"`
		SourceDigest string                    `json:"sourceDigest"`
		Level        int                       `json:"level"`
		Ordinal      int                       `json:"ordinal"`
		Members      []reductionMemberIdentity `json:"members"`
	}{
		ReduceKey: partition.ReduceKey, SourceDigest: partition.SourceDigest,
		Level: partition.Level, Ordinal: partition.Ordinal, Members: members,
	})
	if err != nil {
		return "", err
	}
	return NodeKey("reduce:" + digest[len("sha256:"):]), nil
}
