package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	pingDB    func(ctx context.Context) error
	pingRedis func(ctx context.Context) error
}

func NewHealthHandler(db *sqlx.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{
		pingDB:    db.PingContext,
		pingRedis: func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
	}
}

// Health godoc
// @Summary     Health check
// @Tags        observability
// @Produce     json
// @Success     200  {object}  map[string]any
// @Failure     503  {object}  map[string]any
// @Router      /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	dbErr := h.pingDB(ctx)
	redisErr := h.pingRedis(ctx)

	checks := gin.H{
		"postgres": statusString(dbErr),
		"redis":    statusString(redisErr),
	}

	if dbErr != nil || redisErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "checks": checks})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "checks": checks})
}

func statusString(err error) string {
	if err == nil {
		return "ok"
	}
	return "error"
}
