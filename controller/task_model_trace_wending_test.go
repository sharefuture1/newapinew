package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WendingAI: the mapped upstream model of a task is root-only.
func TestTaskDTOUpstreamModelNameIsRootOnly(t *testing.T) {
	task := &model.Task{
		TaskID: "task_trace",
		Properties: model.Properties{
			Input:             "prompt",
			UpstreamModelName: "upstream-x",
			OriginModelName:   "requested-model",
		},
	}

	for _, role := range []int{common.RoleCommonUser, common.RoleAdminUser} {
		encoded, err := common.Marshal(tasksToDto([]*model.Task{task}, false, role)[0])
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "upstream-x")
		assert.Contains(t, string(encoded), "requested-model")
	}

	encoded, err := common.Marshal(tasksToDto([]*model.Task{task}, false, common.RoleRootUser)[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), "upstream-x")
	// The stored task itself is not modified.
	assert.Equal(t, "upstream-x", task.Properties.UpstreamModelName)
}
