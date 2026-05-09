package router

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	_ "github.com/kenanabbak/notification-management-api/docs"
	"github.com/kenanabbak/notification-management-api/internal/api/handler"
	"github.com/kenanabbak/notification-management-api/internal/api/middleware"
	"github.com/kenanabbak/notification-management-api/internal/config"
	"github.com/kenanabbak/notification-management-api/internal/telemetry"
)

func New(cfg *config.Config, tracerCfg *telemetry.TracerConfig, log *zap.Logger, nh *handler.NotificationHandler, mh *handler.MetricsHandler, hh *handler.HealthHandler, th *handler.TemplateHandler, wh *handler.WSHandler) *gin.Engine {
	r := gin.New()
	r.Use(otelgin.Middleware(tracerCfg.ServiceName))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/index.html")
	})
	r.GET("/health", hh.Health)
	r.GET("/metrics", mh.Metrics)
	r.GET("/ws", wh.ServeWS)
	r.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	v1 := r.Group("/api/v1")
	{
		n := v1.Group("/notifications")
		n.POST("", nh.Create)
		n.POST("/batch", nh.CreateBatch)
		n.GET("", nh.List)
		n.GET("/:id", nh.GetByID)
		n.DELETE("/:id", nh.Cancel)

		t := v1.Group("/templates")
		t.POST("", th.Create)
		t.GET("", th.List)
		t.GET("/:id", th.GetByID)
		t.DELETE("/:id", th.Delete)
	}

	return r
}

func RegisterLifecycle(lc fx.Lifecycle, cfg *config.Config, r *gin.Engine, log *zap.Logger) {
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("starting HTTP server", zap.String("addr", srv.Addr))
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatal("server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("shutting down HTTP server")
			return srv.Shutdown(ctx)
		},
	})
}
