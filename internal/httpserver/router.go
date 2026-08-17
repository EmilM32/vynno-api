package httpserver

import (
	"net/http"
	"time"

	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	sessionCookie  = "vynno_session"
	ctxUserID      = "userID"
	rememberMaxAge = int((30 * 24 * time.Hour) / time.Second)
)

type Options struct {
	SPAOrigins      []string
	CookieSecure    bool
	PublicAPIOrigin string
}

type Server struct {
	svc          *service.Service
	spaOrigins   map[string]struct{}
	cookieSecure bool
	publicOrigin string
}

func NewRouter(svc *service.Service, opts Options) *gin.Engine {
	origins := opts.SPAOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[o] = struct{}{}
	}

	publicOrigin := opts.PublicAPIOrigin
	if publicOrigin == "" {
		publicOrigin = "http://localhost:8080"
	}
	s := &Server{svc: svc, spaOrigins: allowed, cookieSecure: opts.CookieSecure, publicOrigin: publicOrigin}
	r := gin.New()
	r.MaxMultipartMemory = maxAvatarMultipart
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	_ = r.SetTrustedProxies(nil)
	r.GET("/healthz", handleHealth)

	v1 := r.Group("/v1")
	v1.POST("/auth/register", s.register)
	v1.POST("/auth/login", s.login)
	v1.GET("/avatars/:id", s.getAvatar)

	authed := v1.Group("")
	authed.Use(s.requireAuth())
	authed.POST("/auth/logout", s.logout)
	authed.GET("/me", s.getMe)
	authed.PATCH("/me", s.patchMe)
	authed.PUT("/me/avatar", s.putMeAvatar)
	authed.DELETE("/me/avatar", s.deleteMeAvatar)
	authed.GET("/projects", s.listProjects)
	authed.POST("/projects", s.createProject)
	authed.GET("/projects/:id", s.getProject)
	authed.PATCH("/projects/:id", s.updateProject)
	authed.DELETE("/projects/:id", s.deleteProject)
	authed.POST("/projects/:id/archive", s.archiveProject)
	authed.POST("/projects/:id/restore", s.restoreProject)
	authed.GET("/projects/:id/session-count", s.projectSessionCount)
	authed.GET("/sessions", s.listSessions)
	authed.POST("/sessions", s.startSession)
	authed.GET("/sessions/active", s.getActiveSession)
	authed.GET("/sessions/:id", s.getSession)
	authed.POST("/sessions/:id/pause", s.pauseSession)
	authed.POST("/sessions/:id/resume", s.resumeSession)
	authed.POST("/sessions/:id/stop", s.stopSession)
	return r
}

func (s *Server) userSvc(c *gin.Context) *service.Service {
	return s.svc.ForUser(c.MustGet(ctxUserID).(uuid.UUID))
}

func (s *Server) setSessionCookie(c *gin.Context, token string, remember bool) {
	maxAge := 0
	if remember {
		maxAge = rememberMaxAge
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, token, maxAge, "/", "", s.cookieSecure, true)
}

func (s *Server) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, "", -1, "/", "", s.cookieSecure, true)
}
