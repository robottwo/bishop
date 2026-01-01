package system

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResources_Struct(t *testing.T) {
	now := time.Now()
	res := &Resources{
		CPUPercent: 50.5,
		RAMUsed:    1024 * 1024 * 1024,      // 1 GB
		RAMTotal:   16 * 1024 * 1024 * 1024, // 16 GB
		VRAMUsed:   512 * 1024 * 1024,       // 512 MB
		VRAMTotal:  8 * 1024 * 1024 * 1024,  // 8 GB
		Timestamp:  now,
	}

	assert.Equal(t, 50.5, res.CPUPercent)
	assert.Equal(t, uint64(1024*1024*1024), res.RAMUsed)
	assert.Equal(t, uint64(16*1024*1024*1024), res.RAMTotal)
	assert.Equal(t, uint64(512*1024*1024), res.VRAMUsed)
	assert.Equal(t, uint64(8*1024*1024*1024), res.VRAMTotal)
	assert.Equal(t, now, res.Timestamp)
}

func TestGetResources_ReturnsNonNil(t *testing.T) {
	res := GetResources()
	require.NotNil(t, res)
}

func TestGetResources_HasTimestamp(t *testing.T) {
	before := time.Now()
	res := GetResources()
	after := time.Now()

	require.NotNil(t, res)
	assert.False(t, res.Timestamp.IsZero())
	assert.True(t, res.Timestamp.After(before) || res.Timestamp.Equal(before))
	assert.True(t, res.Timestamp.Before(after) || res.Timestamp.Equal(after))
}

func TestGetResources_CPUPercentInValidRange(t *testing.T) {
	// Call twice to get a valid CPU reading (first call establishes baseline)
	_ = GetResources()
	time.Sleep(10 * time.Millisecond)
	res := GetResources()

	require.NotNil(t, res)
	// CPU percent should be between 0 and 100
	assert.GreaterOrEqual(t, res.CPUPercent, 0.0)
	assert.LessOrEqual(t, res.CPUPercent, 100.0)
}

func TestGetResources_RAMValuesReasonable(t *testing.T) {
	res := GetResources()

	require.NotNil(t, res)

	// RAM total should be non-zero on any running system
	// Skip this assertion if we couldn't get RAM info
	if res.RAMTotal > 0 {
		// RAM used should be less than or equal to total
		assert.LessOrEqual(t, res.RAMUsed, res.RAMTotal)
		// RAM total should be at least 1 MB (unrealistic minimum, but sanity check)
		assert.GreaterOrEqual(t, res.RAMTotal, uint64(1024*1024))
	}
}

func TestGetResources_MultipleCallsSucceed(t *testing.T) {
	// Ensure multiple calls don't panic or fail
	for i := 0; i < 5; i++ {
		res := GetResources()
		require.NotNil(t, res)
	}
}
