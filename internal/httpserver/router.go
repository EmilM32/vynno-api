package httpserver

import (
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	svc *service.Service
}

func NewRouter(svc *service.Service) *gin.Engine {
	s := &Server{svc: svc}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.Default())
	_ = r.SetTrustedProxies(nil)
	r.GET("/healthz", handleHealth)

	v1 := r.Group("/v1")
	v1.GET("/me", s.getMe)
	v1.GET("/projects", s.listProjects)
	v1.POST("/projects", s.createProject)
	v1.GET("/projects/:id", s.getProject)
	v1.PATCH("/projects/:id", s.updateProject)
	v1.DELETE("/projects/:id", s.deleteProject)
	v1.POST("/projects/:id/archive", s.archiveProject)
	v1.POST("/projects/:id/restore", s.restoreProject)
	v1.GET("/projects/:id/session-count", s.projectSessionCount)
	v1.GET("/sessions", s.listSessions)
	v1.POST("/sessions", s.startSession)
	v1.GET("/sessions/active", s.getActiveSession)
	v1.GET("/sessions/:id", s.getSession)
	v1.POST("/sessions/:id/pause", s.pauseSession)
	v1.POST("/sessions/:id/resume", s.resumeSession)
	v1.POST("/sessions/:id/stop", s.stopSession)
	return r
}
