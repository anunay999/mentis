package handlers

import (
	"net/http"
	"strconv"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/anunay/mentis/internal/core/ports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CacheHandler struct {
	cacheService ports.CacheService
}

func NewCacheHandler(cacheService ports.CacheService) *CacheHandler {
	return &CacheHandler{
		cacheService: cacheService,
	}
}

func (h *CacheHandler) RegisterRoutes(r *gin.RouterGroup) {
	cache := r.Group("/cache")
	{
		cache.POST("/publish", h.Publish)
		cache.POST("/lookup", h.Lookup)
		cache.GET("/artifacts/:id", h.GetArtifact)
		cache.DELETE("/artifacts/:id", h.DeleteArtifact)
		cache.POST("/invalidate", h.Invalidate)
	}
}

func (h *CacheHandler) Publish(c *gin.Context) {
	var req domain.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.cacheService.Publish(c.Request.Context(), req.Objects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *CacheHandler) Lookup(c *gin.Context) {
	var req domain.LookupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.cacheService.Lookup(c.Request.Context(), req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *CacheHandler) GetArtifact(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artifact ID"})
		return
	}

	artifact, err := h.cacheService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if artifact == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "artifact not found"})
		return
	}

	c.JSON(http.StatusOK, artifact)
}

func (h *CacheHandler) DeleteArtifact(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artifact ID"})
		return
	}

	err = h.cacheService.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "artifact deleted"})
}

func (h *CacheHandler) Invalidate(c *gin.Context) {
	var req struct {
		SourceURL string `json:"source_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.Invalidate(c.Request.Context(), req.SourceURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "artifacts invalidated"})
}

// Quick lookup endpoint for GET requests
func (h *CacheHandler) QuickLookup(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	topK := 10
	if topKStr := c.Query("top_k"); topKStr != "" {
		if k, err := strconv.Atoi(topKStr); err == nil {
			topK = k
		}
	}

	minScore := float32(0.85)
	if minScoreStr := c.Query("min_score"); minScoreStr != "" {
		if score, err := strconv.ParseFloat(minScoreStr, 32); err == nil {
			minScore = float32(score)
		}
	}

	options := domain.LookupOptions{
		Query:           query,
		TopK:            topK,
		MinScore:        minScore,
		IncludeContent:  c.Query("include_content") == "true",
		IncludeEmbedding: c.Query("include_embedding") == "true",
		IncludeStale:    c.Query("include_stale") == "true",
	}

	if artifactType := c.Query("type"); artifactType != "" {
		options.ArtifactType = domain.ArtifactType(artifactType)
	}

	response, err := h.cacheService.Lookup(c.Request.Context(), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}