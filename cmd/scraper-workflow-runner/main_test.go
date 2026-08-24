package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapacityFlagIsStrictAndRejectsDuplicates(t *testing.T) {
	var values capacityFlag
	require.NoError(t, values.Set("cpu.default=2"))
	require.Equal(t, 2, values["cpu.default"])
	require.ErrorContains(t, values.Set("cpu.default=3"), "more than once")
	for _, invalid := range []string{"", "cpu.default", "=2", "cpu.default=0", "cpu.default=nope"} {
		var current capacityFlag
		require.Error(t, current.Set(invalid), invalid)
	}
}

func TestTaskPackageFlagRejectsEmptyValue(t *testing.T) {
	var values stringListFlag
	require.Error(t, values.Set(""))
	require.NoError(t, values.Set("research-runner-fixture"))
	require.Equal(t, stringListFlag{"research-runner-fixture"}, values)
}
