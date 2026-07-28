package life

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifySkillTreeNodePrefersCharacteristicBranch(t *testing.T) {
	t.Parallel()
	_, byKey := cloneSkillTree()

	node := classifySkillTreeNode("Write architecture review", "WRI", byKey)
	require.NotNil(t, node)
	assert.Equal(t, "wri-draft", node.Key)

	node = classifySkillTreeNode("Architecture review", "INT", byKey)
	require.NotNil(t, node)
	assert.Equal(t, "int-systems", node.Key)
}

func TestSkillTreeStatusUsesCadenceWindows(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	active := skillTreeStatus([]SkillTreeEvidenceView{{
		Title:      "Daily journal",
		OccurredAt: now.Add(-6 * 24 * time.Hour),
		WindowDays: 7,
	}})
	assert.Equal(t, "Active", active)

	cooling := skillTreeStatus([]SkillTreeEvidenceView{{
		Title:      "Weekly run",
		OccurredAt: now.Add(-30 * 24 * time.Hour),
		WindowDays: 21,
	}})
	assert.Equal(t, "Cooling", cooling)

	quiet := skillTreeStatus([]SkillTreeEvidenceView{{
		Title:      "Monthly review",
		OccurredAt: now.Add(-120 * 24 * time.Hour),
		WindowDays: 45,
	}})
	assert.Equal(t, "Quiet", quiet)
}

func TestSkillTreeWindowDays(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 7, skillTreeWindowDays("daily"))
	assert.Equal(t, 21, skillTreeWindowDays("weekly"))
	assert.Equal(t, 45, skillTreeWindowDays("monthly"))
	assert.Equal(t, 14, skillTreeWindowDays(""))
}
