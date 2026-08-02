package group

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	connect "connectrpc.com/connect"
	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/active"
	storyEngineServer "github.com/grapery/grapery/pkg/story"
	"github.com/grapery/grapery/utils/convert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	heatmapRangeSecondsHeader = "X-Heatmap-Range-Seconds"
	defaultHeatmapDetailRange = 24 * time.Hour
	maxHeatmapDetailRange     = 180 * 24 * time.Hour
)

func (s *StoryBoardService) QueryTaskStatus(ctx context.Context, req *connect.Request[api.QueryTaskStatusRequest]) (*connect.Response[api.QueryTaskStatusResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("QueryTaskStatus called",
		zap.String("traceId", traceId),
		zap.Int64("userId", msg.GetUserId()),
		zap.String("taskId", msg.GetTaskId()),
		zap.Int64("storyId", msg.GetStoryId()),
		zap.Int64("boardId", msg.GetBoardId()))

	if msg.GetUserId() <= 0 {
		err := fmt.Errorf("invalid user_id")
		zap.L().Warn("QueryTaskStatus invalid user",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	taskID := strings.TrimSpace(msg.GetTaskId())
	if taskID == "" {
		err := fmt.Errorf("task_id is required")
		zap.L().Warn("QueryTaskStatus missing taskId",
			zap.String("traceId", traceId))
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	video, err := models.GetVideoGenByTaskID(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("QueryTaskStatus task not found",
				zap.String("traceId", traceId),
				zap.String("taskId", taskID))
			resp := &api.QueryTaskStatusResponse{
				Code:    int32(api.ResponseCode_OPERATION_FAILED),
				Message: "task not found",
			}
			return connect.NewResponse(resp), nil
		}
		zap.L().Error("QueryTaskStatus query failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if video.UserID != 0 && video.UserID != msg.GetUserId() {
		zap.L().Warn("QueryTaskStatus user mismatch",
			zap.String("traceId", traceId),
			zap.Int64("requestUserId", msg.GetUserId()),
			zap.Int64("taskUserId", video.UserID))
		resp := &api.QueryTaskStatusResponse{
			Code:    int32(api.ResponseCode_OPERATION_FAILED),
			Message: "task not found",
		}
		return connect.NewResponse(resp), nil
	}

	status := api.StoryGenStatus(video.GenStatus)
	stage := mapStoryGenStatusToStage(status)
	dashStatus := mapStoryGenStatusToDashStatus(status)

	data := &api.QueryTaskStatusResponse_Data{
		Stage:               stage,
		DashscopeTaskStatus: dashStatus,
		RenderStoryDetail:   buildRenderStoryDetail(video, msg),
	}

	if video.RoleId > 0 {
		if role, err := models.GetStoryRoleByID(ctx, video.RoleId); err == nil && role != nil {
			data.RenderStoryRole = convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("QueryTaskStatus load role failed",
				zap.String("traceId", traceId),
				zap.Int64("roleId", video.RoleId),
				zap.Error(err))
		}
	}

	resp := &api.QueryTaskStatusResponse{
		Code:    int32(api.ResponseCode_OK),
		Message: "OK",
		Data:    data,
	}

	zap.L().Info("QueryTaskStatus success",
		zap.String("traceId", traceId),
		zap.String("taskId", taskID),
		zap.Any("stage", stage),
		zap.Any("dashStatus", dashStatus))

	return connect.NewResponse(resp), nil
}

func mapStoryGenStatusToStage(status api.StoryGenStatus) api.StoryboardStage {
	switch status {
	case api.StoryGenStatus_STORY_GEN_STATUS_INIT, api.StoryGenStatus_STORY_GEN_STATUS_RUNNING:
		return api.StoryboardStage_STORYBOARD_STAGE_GEN_VIDEO
	case api.StoryGenStatus_STORY_GEN_STATUS_FINISHED:
		return api.StoryboardStage_STORYBOARD_STAGE_FINISHED
	case api.StoryGenStatus_STORY_GEN_STATUS_ERROR:
		return api.StoryboardStage_STORYBOARD_STAGE_FAILED
	default:
		return api.StoryboardStage_STORYBOARD_STAGE_UNSPECIFIED
	}
}

func mapStoryGenStatusToDashStatus(status api.StoryGenStatus) api.DashScopeTaskStatus {
	switch status {
	case api.StoryGenStatus_STORY_GEN_STATUS_INIT:
		return api.DashScopeTaskStatus_DashScopeTaskStatusPending
	case api.StoryGenStatus_STORY_GEN_STATUS_RUNNING:
		return api.DashScopeTaskStatus_DashScopeTaskStatusRunning
	case api.StoryGenStatus_STORY_GEN_STATUS_FINISHED:
		return api.DashScopeTaskStatus_DashScopeTaskStatusSucceeded
	case api.StoryGenStatus_STORY_GEN_STATUS_ERROR:
		return api.DashScopeTaskStatus_DashScopeTaskStatusFailed
	default:
		return api.DashScopeTaskStatus_DashScopeTaskStatusUnknown
	}
}

func buildRenderStoryDetail(video *models.VideoGen, req *api.QueryTaskStatusRequest) *api.RenderStoryDetail {
	if video == nil {
		return nil
	}

	urls := make([]string, 0, 4)
	if v := strings.TrimSpace(video.VideoUrl); v != "" {
		urls = append(urls, v)
	}
	if v := strings.TrimSpace(video.FisrtFrame); v != "" {
		urls = append(urls, v)
	}
	if v := strings.TrimSpace(video.EndFrame); v != "" {
		urls = append(urls, v)
	}
	if refs := strings.TrimSpace(video.RefImages); refs != "" {
		for _, item := range strings.Split(refs, ",") {
			if v := strings.TrimSpace(item); v != "" {
				urls = append(urls, v)
			}
		}
	}

	renderType := api.RenderType(video.TaskType)
	if renderType == api.RenderType_RENDER_TYPE_TEXT_UNSPECIFIED && req != nil {
		renderType = req.GetRenderType()
	}
	if renderType == api.RenderType_RENDER_TYPE_TEXT_UNSPECIFIED {
		renderType = api.RenderType_RENDER_TYPE_STORYBOARD
	}

	storyID := video.StoryId
	if storyID == 0 && req != nil {
		storyID = req.GetStoryId()
	}

	boardID := video.BoardId
	if boardID == 0 && req != nil {
		boardID = req.GetBoardId()
	}

	userID := video.UserID
	if userID == 0 && req != nil {
		userID = req.GetUserId()
	}

	var timecost int32
	if video.StartTime > 0 && video.EndTime > video.StartTime {
		diff := video.EndTime - video.StartTime
		if diff < 0 {
			diff = 0
		}
		if diff > math.MaxInt32 {
			timecost = math.MaxInt32
		} else {
			timecost = int32(diff)
		}
	}

	detail := &api.RenderStoryDetail{
		Text:       video.Prompt,
		Status:     int32(video.GenStatus),
		Urls:       urls,
		StoryId:    storyID,
		BoardId:    boardID,
		UserId:     userID,
		RenderType: renderType,
		Timecost:   timecost,
	}

	return detail
}

func (s *StoryBoardService) UserStoryboardDraftlist(ctx context.Context, req *connect.Request[api.UserStoryboardDraftlistRequest]) (*connect.Response[api.UserStoryboardDraftlistResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("UserStoryboardDraftlist called",
		zap.String("traceId", traceId),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int64("offset", msg.GetOffset()),
		zap.Int64("pageSize", msg.GetPageSize()),
		zap.Int64("storyId", msg.GetStoryId()))

	resp, err := storyEngineServer.GetStoryEngine().UserStoryboardDraftlist(ctx, msg)
	if err != nil {
		zap.L().Error("UserStoryboardDraftlist failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("UserStoryboardDraftlist success",
		zap.String("traceId", traceId),
		zap.Int("draftCount", len(resp.GetDrafts())),
		zap.Int64("total", resp.GetTotal()),
		zap.Bool("haveMore", resp.GetHaveMore()),
		zap.Int32("code", int32(resp.GetCode())))

	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) UserStoryboardDraftDetail(ctx context.Context, req *connect.Request[api.UserDraftStoryboardDetailRequest]) (*connect.Response[api.UserDraftStoryboardDetailResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("UserStoryboardDraftDetail called",
		zap.String("traceId", traceId),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int64("draftId", msg.GetDraftId()))

	resp, err := storyEngineServer.GetStoryEngine().UserDraftStoryboardDetail(ctx, msg)
	if err != nil {
		zap.L().Error("UserStoryboardDraftDetail failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("UserStoryboardDraftDetail success",
		zap.String("traceId", traceId),
		zap.Int32("code", int32(resp.GetCode())))

	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) DeleteUserStoryboardDraft(ctx context.Context, req *connect.Request[api.DeleteUserStoryboardDraftRequest]) (*connect.Response[api.DeleteUserStoryboardDraftResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("DeleteUserStoryboardDraft called",
		zap.String("traceId", traceId),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int64("draftId", msg.GetDraftId()),
		zap.Int64("storyId", msg.GetStoryId()))

	resp, err := storyEngineServer.GetStoryEngine().DeleteUserStoryboardDraft(ctx, msg)
	if err != nil {
		zap.L().Error("DeleteUserStoryboardDraft failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("DeleteUserStoryboardDraft finished",
		zap.String("traceId", traceId),
		zap.Int32("code", int32(resp.GetCode())),
		zap.String("message", resp.GetMessage()))

	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) UserActiveHeatmap(ctx context.Context, req *connect.Request[api.UserActiveHeamapRequest]) (*connect.Response[api.UserActiveHeamapResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("UserActiveHeatmap called",
		zap.String("traceId", traceId),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int64("startTime", msg.GetStartTime()),
		zap.Int64("endTime", msg.GetEndTime()))

	if msg.GetUserId() <= 0 {
		err := fmt.Errorf("invalid user_id")
		zap.L().Warn("UserActiveHeatmap invalid arguments",
			zap.String("traceId", traceId),
			zap.Int64("userId", msg.GetUserId()),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := active.BuildUserHeatmap(ctx, msg.GetUserId(), msg.GetStartTime(), msg.GetEndTime())
	if err != nil {
		code := connect.CodeInternal
		if errors.Is(err, active.ErrStartAfterEnd) || errors.Is(err, active.ErrUnsupportedSpan) {
			code = connect.CodeInvalidArgument
		}
		zap.L().Error("UserActiveHeatmap failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, connect.NewError(code, err)
	}

	resp := &api.UserActiveHeamapResponse{
		Code:       api.ResponseCode_OK,
		Message:    "OK",
		Data:       result.Items,
		TotalCount: result.TotalCount,
	}

	zap.L().Info("UserActiveHeatmap success",
		zap.String("traceId", traceId),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int("days", result.Days),
		zap.Int64("totalCount", result.TotalCount))

	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GroupActiveHeatmap(ctx context.Context, req *connect.Request[api.GroupActiveHeamapRequest]) (*connect.Response[api.GroupActiveHeamapResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("GroupActiveHeatmap called",
		zap.String("traceId", traceId),
		zap.Int64("groupId", msg.GetGroupId()),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int64("startTime", msg.GetStartTime()),
		zap.Int64("endTime", msg.GetEndTime()))

	if msg.GetGroupId() <= 0 {
		err := fmt.Errorf("invalid group_id")
		zap.L().Warn("GroupActiveHeatmap invalid arguments",
			zap.String("traceId", traceId),
			zap.Int64("groupId", msg.GetGroupId()),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := active.BuildGroupHeatmap(ctx, msg.GetGroupId(), msg.GetStartTime(), msg.GetEndTime())
	if err != nil {
		code := connect.CodeInternal
		if errors.Is(err, active.ErrStartAfterEnd) || errors.Is(err, active.ErrUnsupportedSpan) {
			code = connect.CodeInvalidArgument
		}
		zap.L().Error("GroupActiveHeatmap failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, connect.NewError(code, err)
	}

	resp := &api.GroupActiveHeamapResponse{
		Code:        api.ResponseCode_OK,
		Message:     "OK",
		Data:        result.Items,
		TotalCount:  result.TotalCount,
		MemberCount: result.MemberCount,
	}

	zap.L().Info("GroupActiveHeatmap success",
		zap.String("traceId", traceId),
		zap.Int64("groupId", msg.GetGroupId()),
		zap.Int("days", result.Days),
		zap.Int64("totalCount", result.TotalCount),
		zap.Int64("memberCount", result.MemberCount))

	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) UpdateStoryboardForkAble(ctx context.Context, req *connect.Request[api.UpdateStoryboardForkAbleRequest]) (*connect.Response[api.UpdateStoryboardForkAbleResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("UpdateStoryboardForkAble called",
		zap.String("traceId", traceId),
		zap.Int64("storyboardId", msg.GetStoryboardId()),
		zap.Int64("userId", msg.GetUserId()),
		zap.Bool("forkAble", msg.GetForkAble()))

	if msg.GetStoryboardId() <= 0 {
		err := fmt.Errorf("invalid storyboard_id")
		zap.L().Warn("UpdateStoryboardForkAble invalid arguments",
			zap.String("traceId", traceId),
			zap.Int64("storyboardId", msg.GetStoryboardId()),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	board, err := models.GetStoryboard(ctx, msg.GetStoryboardId())
	if err != nil {
		zap.L().Error("UpdateStoryboardForkAble query storyboard failed",
			zap.String("traceId", traceId),
			zap.Int64("storyboardId", msg.GetStoryboardId()),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if board == nil {
		err := fmt.Errorf("storyboard not found")
		zap.L().Warn("UpdateStoryboardForkAble storyboard not found",
			zap.String("traceId", traceId),
			zap.Int64("storyboardId", msg.GetStoryboardId()))
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if msg.GetUserId() != 0 && board.CreatorID != msg.GetUserId() {
		err := fmt.Errorf("permission denied")
		zap.L().Warn("UpdateStoryboardForkAble permission denied",
			zap.String("traceId", traceId),
			zap.Int64("storyboardId", msg.GetStoryboardId()),
			zap.Int64("requestUserId", msg.GetUserId()),
			zap.Int64("creatorId", board.CreatorID))
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	if board.ForkAble == msg.GetForkAble() {
		zap.L().Info("UpdateStoryboardForkAble noop",
			zap.String("traceId", traceId),
			zap.Int64("storyboardId", msg.GetStoryboardId()),
			zap.Bool("forkAble", msg.GetForkAble()))
		return connect.NewResponse(&api.UpdateStoryboardForkAbleResponse{
			Code:    api.ResponseCode_OK,
			Message: "OK",
		}), nil
	}

	if err := models.UpdateStoryBoardForkAble(ctx, msg.GetStoryboardId(), msg.GetForkAble()); err != nil {
		zap.L().Error("UpdateStoryboardForkAble update failed",
			zap.String("traceId", traceId),
			zap.Int64("storyboardId", msg.GetStoryboardId()),
			zap.Bool("forkAble", msg.GetForkAble()),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	zap.L().Info("UpdateStoryboardForkAble success",
		zap.String("traceId", traceId),
		zap.Int64("storyboardId", msg.GetStoryboardId()),
		zap.Bool("forkAble", msg.GetForkAble()))

	return connect.NewResponse(&api.UpdateStoryboardForkAbleResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
	}), nil
}

func (s *StoryBoardService) GetStoryboardGenerationRoadmap(ctx context.Context, req *connect.Request[api.GetStoryboardGenerationRoadmapRequest]) (*connect.Response[api.GetStoryboardGenerationRoadmapResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("GetStoryboardGenerationRoadmap called",
		zap.String("traceId", traceId),
		zap.Int64("storyId", msg.GetStoryId()),
		zap.Int64("storyboardId", msg.GetStoryboardId()),
		zap.Int64("userId", msg.GetUserId()))

	resp, err := storyEngineServer.GetStoryEngine().GetStoryboardGenerationRoadmap(ctx, msg)
	if err != nil {
		zap.L().Error("GetStoryboardGenerationRoadmap failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, err
	}
	respJson, _ := json.Marshal(resp)
	zap.L().Info("GetStoryboardGenerationRoadmap success",
		zap.String("traceId", traceId),
		zap.Int32("code", int32(resp.GetCode())),
		zap.String("data", string(respJson)))

	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetActiveHeatmapDetails(ctx context.Context, req *connect.Request[api.GetActiveHeatmapDetailsRequest]) (*connect.Response[api.GetActiveHeatmapDetailsResponse], error) {
	traceId := getTraceID(ctx)
	msg := req.Msg

	zap.L().Info("GetActiveHeatmapDetails called",
		zap.String("traceId", traceId),
		zap.Int64("groupId", msg.GetGroupId()),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int64("timestamp", msg.GetTimestamp()),
		zap.String("rangeHeader", req.Header().Get(heatmapRangeSecondsHeader)))

	if msg.GetTimestamp() <= 0 {
		err := fmt.Errorf("invalid timestamp")
		zap.L().Warn("GetActiveHeatmapDetails invalid timestamp",
			zap.String("traceId", traceId),
			zap.Int64("timestamp", msg.GetTimestamp()),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if msg.GetGroupId() <= 0 && msg.GetUserId() <= 0 {
		err := fmt.Errorf("group_id or user_id is required")
		zap.L().Warn("GetActiveHeatmapDetails missing subject",
			zap.String("traceId", traceId),
			zap.Int64("groupId", msg.GetGroupId()),
			zap.Int64("userId", msg.GetUserId()),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	targetTime := time.Unix(msg.GetTimestamp(), 0).In(time.Local)
	startTime := time.Date(targetTime.Year(), targetTime.Month(), targetTime.Day(), 0, 0, 0, 0, targetTime.Location())
	rangeDuration := defaultHeatmapDetailRange

	if header := strings.TrimSpace(req.Header().Get(heatmapRangeSecondsHeader)); header != "" {
		if secs, err := strconv.ParseInt(header, 10, 64); err != nil {
			zap.L().Warn("GetActiveHeatmapDetails invalid range header",
				zap.String("traceId", traceId),
				zap.String("rangeHeader", header),
				zap.Error(err))
		} else if secs > 0 {
			duration := time.Duration(secs) * time.Second
			if duration > maxHeatmapDetailRange {
				duration = maxHeatmapDetailRange
			}
			rangeDuration = duration
			startTime = targetTime
		}
	}

	endTime := startTime.Add(rangeDuration)
	actives, err := fetchHeatmapActives(ctx, msg.GetGroupId(), msg.GetUserId(), startTime, endTime)
	if err != nil {
		zap.L().Error("GetActiveHeatmapDetails query actives failed",
			zap.String("traceId", traceId),
			zap.Int64("groupId", msg.GetGroupId()),
			zap.Int64("userId", msg.GetUserId()),
			zap.Time("startTime", startTime),
			zap.Time("endTime", endTime),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(actives) == 0 {
		resp := &api.GetActiveHeatmapDetailsResponse{
			Code:       api.ResponseCode_OK,
			Message:    "OK",
			Data:       []*api.ActiveHeatmapDetail{},
			TotalCount: 0,
		}
		zap.L().Info("GetActiveHeatmapDetails no data",
			zap.String("traceId", traceId),
			zap.Int64("groupId", msg.GetGroupId()),
			zap.Int64("userId", msg.GetUserId()),
			zap.Time("startTime", startTime),
			zap.Time("endTime", endTime))
		return connect.NewResponse(resp), nil
	}

	infoList, err := buildActiveHeatmapInfos(ctx, actives)
	if err != nil {
		zap.L().Error("GetActiveHeatmapDetails build infos failed",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	heatmapItems, totalCount := buildHeatmapSummary(actives, startTime)
	details := make([]*api.ActiveHeatmapDetail, 0, len(actives))
	dateStr := startTime.Format("2006-01-02")
	for idx, activeItem := range actives {
		dataItem, ok := heatmapItems[activeItem.GroupId]
		if !ok || dataItem == nil {
			dataItem = &api.HeatmapDataItem{
				Date: dateStr,
			}
		} else {
			dataItem = &api.HeatmapDataItem{
				Date:  dataItem.Date,
				Count: dataItem.Count,
				Level: dataItem.Level,
			}
		}

		details = append(details, &api.ActiveHeatmapDetail{
			GroupId: activeItem.GroupId,
			Data:    dataItem,
			Info:    infoList[idx],
		})
	}

	resp := &api.GetActiveHeatmapDetailsResponse{
		Code:       api.ResponseCode_OK,
		Message:    "OK",
		Data:       details,
		TotalCount: totalCount,
	}

	zap.L().Info("GetActiveHeatmapDetails success",
		zap.String("traceId", traceId),
		zap.Int64("groupId", msg.GetGroupId()),
		zap.Int64("userId", msg.GetUserId()),
		zap.Int("detailCount", len(details)),
		zap.Int64("totalCount", totalCount),
		zap.Time("startTime", startTime),
		zap.Time("endTime", endTime))

	return connect.NewResponse(resp), nil
}

func fetchHeatmapActives(ctx context.Context, groupID, userID int64, startTime, endTime time.Time) ([]*models.Active, error) {
	query := models.DataBase().WithContext(ctx).
		Model(&models.Active{}).
		Where("deleted = 0").
		Where("create_at >= ? AND create_at < ?", startTime, endTime)

	if groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	var actives []*models.Active
	if err := query.Order("create_at desc").Find(&actives).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []*models.Active{}, nil
		}
		return nil, err
	}
	return actives, nil
}

func buildActiveHeatmapInfos(ctx context.Context, actives []*models.Active) ([]*api.ActiveInfo, error) {
	if len(actives) == 0 {
		return []*api.ActiveInfo{}, nil
	}

	userIDs := make([]int64, 0, len(actives))
	groupIDs := make([]int64, 0, len(actives))
	storyIDs := make([]int64, 0, len(actives))
	roleIDs := make([]int64, 0, len(actives))
	boardIDs := make([]int64, 0, len(actives))

	for _, item := range actives {
		if item == nil {
			continue
		}
		if item.UserId > 0 {
			userIDs = append(userIDs, item.UserId)
		}
		if item.GroupId > 0 {
			groupIDs = append(groupIDs, item.GroupId)
		}
		if item.StoryId > 0 {
			storyIDs = append(storyIDs, item.StoryId)
		}
		if item.StoryRoleId > 0 {
			roleIDs = append(roleIDs, item.StoryRoleId)
		}
		if item.StoryBoardId > 0 {
			boardIDs = append(boardIDs, item.StoryBoardId)
		}
	}

	userMap := make(map[int64]*models.User, len(userIDs))
	if len(userIDs) > 0 {
		users, err := models.GetUsersByIds(ctx, userIDs)
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			if user != nil {
				userMap[int64(user.ID)] = user
			}
		}
	}

	groupMap := make(map[int64]*models.Group, len(groupIDs))
	if len(groupIDs) > 0 {
		groups, err := models.GetGroupsByIds(groupIDs)
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			if group != nil {
				groupMap[int64(group.ID)] = group
			}
		}
	}

	storyMap := make(map[int64]*models.Story, len(storyIDs))
	if len(storyIDs) > 0 {
		stories, err := models.GetStoriesByIDs(ctx, storyIDs)
		if err != nil {
			return nil, err
		}
		for _, story := range stories {
			if story != nil {
				storyMap[int64(story.ID)] = story
			}
		}
	}

	roleMap := make(map[int64]*models.StoryRole, len(roleIDs))
	if len(roleIDs) > 0 {
		roles, err := models.GetStoryRolesByIDs(ctx, roleIDs)
		if err != nil {
			return nil, err
		}
		for _, role := range roles {
			if role != nil {
				roleMap[int64(role.ID)] = role
			}
		}
	}

	boardMap := make(map[int64]*models.StoryBoard, len(boardIDs))
	if len(boardIDs) > 0 {
		boards, err := models.GetStoryBoardsByIds(ctx, boardIDs)
		if err != nil {
			return nil, err
		}
		for _, board := range boards {
			if board != nil {
				boardMap[int64(board.ID)] = board
			}
		}
	}

	infos := make([]*api.ActiveInfo, 0, len(actives))
	for _, item := range actives {
		if item == nil {
			continue
		}

		info := &api.ActiveInfo{
			ActiveId:   int64(item.ID),
			ActiveType: item.ActiveType,
			Content:    item.Content,
			Ctime:      item.CreateAt.Unix(),
			Mtime:      item.UpdateAt.Unix(),
		}

		if item.UserId > 0 {
			if user := userMap[item.UserId]; user != nil {
				info.User = convert.ConvertUserToApiUser(user)
			}
		}
		if item.GroupId > 0 {
			if group := groupMap[item.GroupId]; group != nil {
				info.GroupInfo = convert.ConvertGroupToApiGroupInfo(group)
			}
		}
		if item.StoryId > 0 {
			if story := storyMap[item.StoryId]; story != nil {
				info.StoryInfo = convert.ConvertStoryToApiStory(story)
			}
		}
		if item.StoryRoleId > 0 {
			if role := roleMap[item.StoryRoleId]; role != nil {
				info.RoleInfo = convert.ConvertStoryRoleToApiStoryRoleInfo(role)
			}
		}
		if item.StoryBoardId > 0 {
			if board := boardMap[item.StoryBoardId]; board != nil {
				info.BoardInfo = convert.ConvertStoryBoardToApiStoryBoard(board)
			}
		}

		infos = append(infos, info)
	}

	return infos, nil
}

func buildHeatmapSummary(actives []*models.Active, start time.Time) (map[int64]*api.HeatmapDataItem, int64) {
	counts := make(map[int64]int64)
	var total int64
	for _, item := range actives {
		if item == nil {
			continue
		}
		counts[item.GroupId]++
		total++
	}

	var maxCount int64
	for _, cnt := range counts {
		if cnt > maxCount {
			maxCount = cnt
		}
	}

	items := make(map[int64]*api.HeatmapDataItem, len(counts))
	dateStr := start.Format("2006-01-02")
	for groupID, cnt := range counts {
		items[groupID] = &api.HeatmapDataItem{
			Date:  dateStr,
			Count: cnt,
			Level: calculateHeatmapLevel(cnt, maxCount),
		}
	}

	if total == 0 {
		total = int64(len(actives))
	}
	return items, total
}

func calculateHeatmapLevel(count, max int64) int64 {
	if count <= 0 || max <= 0 {
		return 0
	}

	ratio := float64(count) / float64(max)

	switch {
	case ratio >= 0.8:
		return 4
	case ratio >= 0.6:
		return 3
	case ratio >= 0.4:
		return 2
	case ratio > 0:
		return 1
	default:
		return 0
	}
}
