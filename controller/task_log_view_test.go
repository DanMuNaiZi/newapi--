package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskLogDTOSeparatesUserAdminAndRootDetails(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_public",
		Platform: "document-parser",
		PrivateData: model.TaskPrivateData{
			Key:            "channel-secret-canary",
			UpstreamTaskID: "upstream-private",
			NodeName:       "node-a",
			Execution: &model.TaskExecutionSnapshot{
				RequestID:   "request-public",
				RequestPath: "/v1/documents",
				TaskPlugin: &model.TaskPluginSnapshot{
					Key:     "document-parser",
					Name:    "Document Parser",
					Version: "1.2.3",
					Author: &model.TaskPluginAuthorSnapshot{
						Name: "Community Author",
						URL:  "https://plugins.example/author",
					},
					APIVersion: 1,
					Generation: 42,
				},
			},
		},
	}

	userView := tasksToDto([]*model.Task{task}, false, common.RoleCommonUser)[0]
	assert.Nil(t, userView.AdminInfo)
	assert.Nil(t, userView.RootInfo)

	adminView := tasksToDto([]*model.Task{task}, false, common.RoleAdminUser)[0]
	require.NotNil(t, adminView.AdminInfo)
	require.NotNil(t, adminView.AdminInfo.TaskPlugin)
	assert.Equal(t, "document-parser", adminView.AdminInfo.TaskPlugin.Key)
	assert.Equal(t, "Document Parser", adminView.AdminInfo.TaskPlugin.Name)
	assert.Equal(t, "1.2.3", adminView.AdminInfo.TaskPlugin.Version)
	require.NotNil(t, adminView.AdminInfo.TaskPlugin.Author)
	assert.Equal(t, "Community Author", adminView.AdminInfo.TaskPlugin.Author.Name)
	assert.Equal(t, "https://plugins.example/author", adminView.AdminInfo.TaskPlugin.Author.URL)
	assert.Equal(t, "request-public", adminView.AdminInfo.RequestID)
	assert.Equal(t, "/v1/documents", adminView.AdminInfo.RequestPath)
	assert.Nil(t, adminView.RootInfo)

	rootView := tasksToDto([]*model.Task{task}, false, common.RoleRootUser)[0]
	require.NotNil(t, rootView.AdminInfo)
	require.NotNil(t, rootView.RootInfo)
	require.NotNil(t, rootView.RootInfo.TaskPlugin)
	assert.Equal(t, 1, rootView.RootInfo.TaskPlugin.APIVersion)
	assert.Equal(t, uint64(42), rootView.RootInfo.TaskPlugin.Generation)
	assert.Equal(t, "upstream-private", rootView.RootInfo.UpstreamTaskID)
	assert.Equal(t, "node-a", rootView.RootInfo.NodeName)

	adminJSON, err := common.Marshal(adminView)
	require.NoError(t, err)
	assert.NotContains(t, string(adminJSON), "channel-secret-canary")
	assert.NotContains(t, string(adminJSON), "upstream-private")

	rootJSON, err := common.Marshal(rootView)
	require.NoError(t, err)
	assert.NotContains(t, string(rootJSON), "channel-secret-canary")
	assert.Contains(t, string(rootJSON), "upstream-private")
}

func TestTaskLogDTODoesNotInventHistoricalPluginProvenance(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_without_snapshot",
		Platform: "document-parser",
	}

	adminView := tasksToDto([]*model.Task{task}, false, common.RoleAdminUser)[0]

	assert.Nil(t, adminView.AdminInfo)
	assert.Nil(t, adminView.RootInfo)
}

func TestTaskLogDTOReplacesLegacyVideoURLWithAvailabilityFlag(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_legacy_video",
		Platform:   "jimeng",
		Action:     constant.TaskActionTextToVideo,
		Status:     model.TaskStatusSuccess,
		FailReason: "https://private-upstream.invalid/video.mp4?signature=secret",
	}

	view := tasksToDto([]*model.Task{task}, false, common.RoleCommonUser)[0]
	assert.True(t, view.LegacyVideoAvailable)
	assert.Empty(t, view.ResultURL)
	assert.Empty(t, view.FailReason)
	encoded, err := common.Marshal(view)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "private-upstream.invalid")
	assert.NotContains(t, string(encoded), "result_url")
	assert.Contains(t, string(encoded), "legacy_video_available")
}

func TestTaskLogDTOKeepsFailureReasonAndDoesNotMarkPluginTaskLegacy(t *testing.T) {
	failed := &model.Task{
		TaskID:     "task_failed",
		Platform:   "jimeng",
		Action:     constant.TaskActionTextToVideo,
		Status:     model.TaskStatusFailure,
		FailReason: "provider rejected the request",
	}
	failedView := tasksToDto([]*model.Task{failed}, false, common.RoleCommonUser)[0]
	assert.Equal(t, "provider rejected the request", failedView.FailReason)
	assert.False(t, failedView.LegacyVideoAvailable)

	pluginTask := &model.Task{
		TaskID:     "task_plugin_video",
		Platform:   "community-video",
		Action:     constant.TaskActionTextToVideo,
		Status:     model.TaskStatusSuccess,
		FailReason: "https://stale-upstream.invalid/plugin-video.mp4",
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://private-upstream.invalid/plugin-video.mp4",
			Execution: &model.TaskExecutionSnapshot{
				TaskPlugin: &model.TaskPluginSnapshot{Key: "community-video"},
			},
		},
	}
	pluginView := tasksToDto([]*model.Task{pluginTask}, false, common.RoleCommonUser)[0]
	assert.False(t, pluginView.LegacyVideoAvailable)
	assert.Empty(t, pluginView.ResultURL)
	assert.Empty(t, pluginView.FailReason)
}

func TestTaskFailureProjectionsProtectModelAndSecretsWithoutMutatingTask(t *testing.T) {
	task := &model.Task{
		TaskID: "task_failed", Status: model.TaskStatusFailure,
		Properties:  model.Properties{Input: "safe input", OriginModelName: "requested-model", UpstreamModelName: "private-model"},
		FailReason:  `private-model failed for channel-key-secret, password="private-password", body: {"private":"private-body"}`,
		Data:        []byte(`{"model":"private-model","error":{"message":"private-model failed for channel-key-secret"}}`),
		PrivateData: model.TaskPrivateData{Key: "channel-key-secret"},
	}
	original, err := common.Marshal(task)
	require.NoError(t, err)
	for _, tc := range []struct {
		name              string
		role              int
		actualModelAccess bool
	}{
		{"ordinary user", common.RoleCommonUser, false},
		{"ordinary user cannot elevate projection", common.RoleCommonUser, true},
		{"restricted administrator", common.RoleAuthorizedAdmin, false},
		{"restricted full administrator", common.RoleAdminUser, false},
		{"permitted authorized administrator", common.RoleAuthorizedAdmin, true},
		{"permitted administrator", common.RoleAdminUser, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			view := tasksToDto([]*model.Task{task}, false, tc.role, tc.actualModelAccess)[0]
			properties, ok := view.Properties.(model.Properties)
			require.True(t, ok)
			assert.Equal(t, "requested-model", properties.OriginModelName)
			assert.Equal(t, "safe input", properties.Input)
			if tc.role >= common.RoleAuthorizedAdmin && tc.actualModelAccess {
				assert.Equal(t, "private-model", properties.UpstreamModelName)
			} else {
				assert.Empty(t, properties.UpstreamModelName)
			}
			assert.Contains(t, view.FailReason, "requested-model")
			assert.NotContains(t, view.FailReason, "private-model")
			assert.NotContains(t, view.FailReason, "channel-key-secret")
			assert.NotContains(t, view.FailReason, "private-password")
			assert.NotContains(t, view.FailReason, "private-body")
			assert.Empty(t, view.ResultURL, "a failure reason is not a task artifact URL")
			assert.Empty(t, view.Data, "failed upstream bodies are private, even when an administrator may view the model field")
		})
	}
	native := relay.TaskModel2Dto(task)
	nativeJSON, err := common.Marshal(native)
	require.NoError(t, err)
	assert.NotContains(t, string(nativeJSON), "private-model")
	assert.NotContains(t, string(nativeJSON), "channel-key-secret")
	assert.NotContains(t, string(nativeJSON), "private-password")
	assert.NotContains(t, string(nativeJSON), "private-body")
	after, err := common.Marshal(task)
	require.NoError(t, err)
	assert.Equal(t, string(original), string(after))
	assert.Equal(t, "channel-key-secret", task.PrivateData.Key)
	t.Run("historical task without a requested model", func(t *testing.T) {
		historical := &model.Task{Status: model.TaskStatusFailure, Properties: model.Properties{UpstreamModelName: "private-model"}, FailReason: "private-model failed"}
		view := relay.TaskModel2Dto(historical)
		assert.Equal(t, "requested model failed", view.FailReason)
		assert.Empty(t, view.ResultURL)
		assert.Equal(t, "private-model failed", historical.FailReason)
	})
}

func TestTaskUserPreviewUsesSanitizedTargetProjection(t *testing.T) {
	actor, _ := setupSecurityEnrollmentTest(t)
	require.NoError(t, model.DB.Model(actor).Update("role", common.RoleRootUser).Error)
	require.NoError(t, model.PublishUserAuthCache(actor.Id))
	bundle, err := service.CreateLoginSession(actor.Id, "password", "127.0.0.1", "task-preview-test")
	require.NoError(t, err)
	target := &model.User{Username: "task-preview-target", Password: "placeholder", AffCode: "task-preview-target", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AuthVersion: 1, Group: "default"}
	require.NoError(t, model.DB.Create(target).Error)
	require.NoError(t, model.PublishUserAuthCache(target.Id))
	require.NoError(t, model.DB.AutoMigrate(&model.Task{}))
	task := &model.Task{
		TaskID: "task_preview_failed", UserId: target.Id, Status: model.TaskStatusFailure,
		Properties: model.Properties{OriginModelName: "requested-model", UpstreamModelName: "private-model"},
		FailReason: `private-model failed, token="private-token", body: {"private":"private-body"}`,
	}
	require.NoError(t, model.DB.Create(task).Error)
	preview, _, err := service.IssueUserPreviewToken(actor.Id, target.Id, time.Now())
	require.NoError(t, err)
	engine := gin.New()
	engine.GET("/api/task/self", middleware.UserAuth(), GetUserTask)
	mutations := 0
	engine.POST("/api/task/self", middleware.UserAuth(), func(c *gin.Context) { mutations++; c.Status(http.StatusOK) })
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		request := httptest.NewRequest(method, "/api/task/self", nil)
		request.Header.Set("Authorization", "Bearer "+bundle.AccessToken)
		request.Header.Set("New-Api-User-Preview", preview)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		if method == http.MethodPost {
			assert.Equal(t, http.StatusForbidden, response.Code)
			continue
		}
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), task.TaskID)
		assert.Contains(t, response.Body.String(), "requested-model")
		assert.NotContains(t, response.Body.String(), "private-model")
		assert.NotContains(t, response.Body.String(), "private-token")
		assert.NotContains(t, response.Body.String(), "private-body")
	}
	assert.Zero(t, mutations)
	var persisted model.Task
	require.NoError(t, model.DB.First(&persisted, task.ID).Error)
	assert.Equal(t, task.Properties, persisted.Properties)
	assert.Equal(t, task.FailReason, persisted.FailReason)
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Set("id", target.Id)
	c.Params = gin.Params{{Key: "key", Value: task.TaskID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/tasks/"+task.TaskID, nil)
	GetTask(c)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "requested-model")
	assert.NotContains(t, response.Body.String(), "private-model")
	assert.NotContains(t, response.Body.String(), "private-token")
	assert.NotContains(t, response.Body.String(), "private-body")
}
