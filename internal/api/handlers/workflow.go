package handlers

import (
	"net/http"
	"strconv"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/anunay/mentis/internal/core/ports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WorkflowHandler struct {
	workflowService ports.WorkflowService
}

func NewWorkflowHandler(workflowService ports.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{
		workflowService: workflowService,
	}
}

func (h *WorkflowHandler) RegisterRoutes(r *gin.RouterGroup) {
	workflow := r.Group("/workflow")
	{
		workflow.POST("/sessions", h.CreateSession)
		workflow.GET("/sessions/:id", h.GetSession)
		workflow.POST("/sessions/:id/complete", h.CompleteSession)
		workflow.POST("/sessions/:id/fail", h.FailSession)
		workflow.POST("/steps", h.ExecuteStep)
		workflow.POST("/steps/lookup", h.LookupStep)
	}
}

func (h *WorkflowHandler) CreateSession(c *gin.Context) {
	var req struct {
		Goal    string                 `json:"goal" binding:"required"`
		Context map[string]interface{} `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.workflowService.CreateSession(c.Request.Context(), req.Goal, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

func (h *WorkflowHandler) GetSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	session, err := h.workflowService.GetSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *WorkflowHandler) CompleteSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	err = h.workflowService.CompleteSession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session completed"})
}

func (h *WorkflowHandler) FailSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.workflowService.FailSession(c.Request.Context(), id, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session failed"})
}

func (h *WorkflowHandler) ExecuteStep(c *gin.Context) {
	var req domain.WorkflowStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.workflowService.ExecuteStep(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *WorkflowHandler) LookupStep(c *gin.Context) {
	var req domain.WorkflowLookupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.TopK == 0 {
		req.TopK = 10
	}

	response, err := h.workflowService.LookupStep(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Quick step lookup endpoint for GET requests
func (h *WorkflowHandler) QuickStepLookup(c *gin.Context) {
	sessionIDStr := c.Query("session_id")
	if sessionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id parameter is required"})
		return
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	stepType := c.Query("step_type")
	if stepType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "step_type parameter is required"})
		return
	}

	input := c.Query("input")
	if input == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input parameter is required"})
		return
	}

	topK := 10
	if topKStr := c.Query("top_k"); topKStr != "" {
		if k, err := strconv.Atoi(topKStr); err == nil {
			topK = k
		}
	}

	req := domain.WorkflowLookupRequest{
		SessionID: sessionID,
		StepType:  stepType,
		Input:     input,
		TopK:      topK,
	}

	response, err := h.workflowService.LookupStep(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}