package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/domain"
	"github.com/hxuan190/stable_payment_gateway/internal/modules/compliance/repository"
)

type AMLRuleHandler struct {
	ruleRepo *repository.AMLRuleRepository
}

func NewAMLRuleHandler(ruleRepo *repository.AMLRuleRepository) *AMLRuleHandler {
	return &AMLRuleHandler{
		ruleRepo: ruleRepo,
	}
}

func (h *AMLRuleHandler) ListRules(c *gin.Context) {
	rules, err := h.ruleRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"total": len(rules),
	})
}

func (h *AMLRuleHandler) GetRule(c *gin.Context) {
	ruleID := c.Param("id")

	rule, err := h.ruleRepo.GetByID(c.Request.Context(), ruleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (h *AMLRuleHandler) GetRulesByCategory(c *gin.Context) {
	category := domain.AMLRuleCategory(c.Param("category"))

	rules, err := h.ruleRepo.GetEnabledByCategory(c.Request.Context(), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"rules":    rules,
		"total":    len(rules),
	})
}

func (h *AMLRuleHandler) CreateRule(c *gin.Context) {
	var rule domain.AMLRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.ruleRepo.Create(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

func (h *AMLRuleHandler) UpdateRule(c *gin.Context) {
	ruleID := c.Param("id")

	var rule domain.AMLRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = ruleID

	if err := h.ruleRepo.Update(c.Request.Context(), &rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (h *AMLRuleHandler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("id")

	if err := h.ruleRepo.Delete(c.Request.Context(), ruleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted successfully"})
}

func (h *AMLRuleHandler) ToggleRule(c *gin.Context) {
	ruleID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.ruleRepo.UpdateEnabled(c.Request.Context(), ruleID, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rule status updated",
		"enabled": req.Enabled,
	})
}
