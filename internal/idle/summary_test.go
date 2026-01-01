package idle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewSummaryGenerator(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	generator := NewSummaryGenerator(nil, nil, logger)

	require.NotNil(t, generator)
	assert.Nil(t, generator.runner)
	assert.Nil(t, generator.historyManager)
	assert.NotNil(t, generator.logger)
}

func TestSummaryGenerator_Struct(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	generator := &SummaryGenerator{
		runner:         nil,
		historyManager: nil,
		logger:         logger,
	}

	assert.Nil(t, generator.runner)
	assert.Nil(t, generator.historyManager)
	assert.NotNil(t, generator.logger)
}
