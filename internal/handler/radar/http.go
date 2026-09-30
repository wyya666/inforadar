package radar

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	radardto "inforadar/internal/api/dto/radar"
	"inforadar/internal/common"
	"inforadar/internal/constants"
	handlerinterface "inforadar/internal/handler/interface"
	serviceinterface "inforadar/internal/service/interface"
)

type Handler struct{ service serviceinterface.RadarService }

var _ handlerinterface.RadarHandler = (*Handler)(nil)

func New(service serviceinterface.RadarService) handlerinterface.RadarHandler {
	return &Handler{service: service}
}

func userID(c *gin.Context) uint64 { return c.MustGet(constants.UserIDContextKey).(uint64) }
func topicID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.WriteError(c, common.ErrParams)
		return 0, false
	}
	return id, true
}
func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		common.WriteBindError(c, err)
		return false
	}
	return true
}

func (h *Handler) GetSettings(c *gin.Context) {
	row, err := h.service.Settings(c.Request.Context(), userID(c))
	if err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, row)
}
func (h *Handler) PutSettings(c *gin.Context) {
	var req radardto.SettingsRequest
	if !bind(c, &req) {
		return
	}
	row, err := h.service.SaveSettings(c.Request.Context(), userID(c), req.ModelProvider, req.ModelID, req.SearchProvider, req.IntervalHours)
	if err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, row)
}
func (h *Handler) PutKey(c *gin.Context) {
	var req radardto.CredentialRequest
	if !bind(c, &req) {
		return
	}
	if err := h.service.SaveCredential(c.Request.Context(), userID(c), c.Param("provider"), req.APIKey); err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, nil)
}
func (h *Handler) DeleteKey(c *gin.Context) {
	if err := h.service.DeleteCredential(c.Request.Context(), userID(c), c.Param("provider")); err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, nil)
}
func (h *Handler) ListTopics(c *gin.Context) {
	rows, err := h.service.ListTopics(c.Request.Context(), userID(c))
	if err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, rows)
}
func (h *Handler) CreateTopic(c *gin.Context) {
	var req radardto.TopicRequest
	if !bind(c, &req) {
		return
	}
	row, err := h.service.CreateTopic(c.Request.Context(), userID(c), req.Description)
	if err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusCreated, row)
}
func (h *Handler) UpdateTopic(c *gin.Context) {
	id, ok := topicID(c)
	if !ok {
		return
	}
	var req radardto.TopicUpdateRequest
	if !bind(c, &req) {
		return
	}
	row, err := h.service.UpdateTopic(c.Request.Context(), userID(c), id, req.Description, req.Status)
	if err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, row)
}
func (h *Handler) DeleteTopic(c *gin.Context) {
	id, ok := topicID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTopic(c.Request.Context(), userID(c), id); err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, nil)
}
func (h *Handler) ScanTopic(c *gin.Context) {
	id, ok := topicID(c)
	if !ok {
		return
	}
	if err := h.service.ManualScan(c.Request.Context(), userID(c), id); err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusAccepted, nil)
}
func (h *Handler) ListRuns(c *gin.Context) {
	id, ok := topicID(c)
	if !ok {
		return
	}
	rows, err := h.service.ListRuns(c.Request.Context(), userID(c), id)
	if err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, rows)
}
func (h *Handler) ListFeed(c *gin.Context) {
	parse := func(name string) (uint64, bool) {
		value := strings.TrimSpace(c.Query(name))
		if value == "" {
			return 0, true
		}
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			common.WriteError(c, common.ErrParams)
			return 0, false
		}
		return parsed, true
	}
	topic, ok := parse("topicId")
	if !ok {
		return
	}
	before, ok := parse("before")
	if !ok {
		return
	}
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 50 {
			common.WriteError(c, common.ErrParams)
			return
		}
		limit = parsed
	}
	feed, err := h.service.ListFeed(c.Request.Context(), userID(c), topic, before, limit)
	if err != nil {
		common.WriteErrorFrom(c, err)
		return
	}
	common.WriteSuccess(c, http.StatusOK, feed)
}
