package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenanabbak/notification-management-api/internal/domain"
)

type NotificationHandler struct {
	svc domain.NotificationService
}

func NewNotificationHandler(svc domain.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// Create godoc
// @Summary     Create a notification
// @Tags        notifications
// @Accept      json
// @Produce     json
// @Param       body  body      domain.CreateNotificationInput  true  "Notification input"
// @Success     201   {object}  domain.Notification
// @Failure     400   {object}  map[string]string
// @Failure     409   {object}  map[string]string
// @Router      /api/v1/notifications [post]
func (h *NotificationHandler) Create(c *gin.Context) {
	var input domain.CreateNotificationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n, err := h.svc.Create(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicate) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrContentTooLong) || errors.Is(err, domain.ErrInvalidRecipient) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrTemplateNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrContentRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, n)
}

// CreateBatch godoc
// @Summary     Create notifications in batch (up to 1000)
// @Tags        notifications
// @Accept      json
// @Produce     json
// @Param       body  body      domain.CreateBatchInput  true  "Batch input"
// @Success     201   {object}  domain.BatchResult
// @Failure     400   {object}  map[string]string
// @Router      /api/v1/notifications/batch [post]
func (h *NotificationHandler) CreateBatch(c *gin.Context) {
	var input domain.CreateBatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.CreateBatch(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrContentTooLong) || errors.Is(err, domain.ErrInvalidRecipient) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrTemplateNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrContentRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetByID godoc
// @Summary     Get notification by ID
// @Tags        notifications
// @Produce     json
// @Param       id   path      string  true  "Notification ID"
// @Success     200  {object}  domain.Notification
// @Failure     404  {object}  map[string]string
// @Router      /api/v1/notifications/{id} [get]
func (h *NotificationHandler) GetByID(c *gin.Context) {
	n, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, n)
}

// Cancel godoc
// @Summary     Cancel a pending/queued notification
// @Tags        notifications
// @Produce     json
// @Param       id   path      string  true  "Notification ID"
// @Success     200  {object}  map[string]string
// @Failure     404  {object}  map[string]string
// @Failure     409  {object}  map[string]string
// @Router      /api/v1/notifications/{id} [delete]
func (h *NotificationHandler) Cancel(c *gin.Context) {
	err := h.svc.Cancel(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		if errors.Is(err, domain.ErrCannotCancel) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification cancelled"})
}

// List godoc
// @Summary     List notifications with filters and pagination
// @Tags        notifications
// @Produce     json
// @Param       status     query     string  false  "Filter by status"
// @Param       channel    query     string  false  "Filter by channel"
// @Param       batch_id   query     string  false  "Filter by batch ID"
// @Param       date_from  query     string  false  "Filter from date (RFC3339)"
// @Param       date_to    query     string  false  "Filter to date (RFC3339)"
// @Param       page       query     int     false  "Page number (default 1)"
// @Param       page_size  query     int     false  "Page size (default 20, max 100)"
// @Success     200        {object}  map[string]any
// @Router      /api/v1/notifications [get]
func (h *NotificationHandler) List(c *gin.Context) {
	var filter domain.ListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	notifications, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      notifications,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	})
}
