package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
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
	TrustedProxies  []string
	Ready           func(context.Context) error
}

type Server struct {
	svc          *service.Service
	spaOrigins   map[string]struct{}
	cookieSecure bool
	publicOrigin string
	ready        func(context.Context) error
	ops          []documentedOp
	specJSON     []byte
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
	s := &Server{
		svc: svc, spaOrigins: allowed, cookieSecure: opts.CookieSecure,
		publicOrigin: publicOrigin, ready: opts.Ready,
	}
	r := gin.New()
	r.MaxMultipartMemory = maxAvatarMultipart
	r.Use(requestID())
	r.Use(requestLog())
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsAllowOrigins(origins, publicOrigin),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", requestIDHeader},
		ExposeHeaders:    []string{requestIDHeader},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	proxies := opts.TrustedProxies
	if len(proxies) == 0 {
		proxies = []string{"127.0.0.1", "::1"}
	}
	if err := r.SetTrustedProxies(proxies); err != nil {
		panic("trusted proxies: " + err.Error())
	}
	r.HandleMethodNotAllowed = true

	s.route(r, http.MethodGet, "/healthz", handleHealth, op{
		Summary: "Liveness",
		Tags:    []string{"Ops"},
		Public:  true,
		Success: healthResponse{},
	})
	s.route(r, http.MethodGet, "/readyz", s.handleReady, op{
		Summary: "Readiness (Postgres ping)",
		Tags:    []string{"Ops"},
		Public:  true,
		Success: healthResponse{},
	})

	v1 := r.Group("/v1")
	s.route(v1, http.MethodPost, "/auth/register/code", s.requestRegisterCode, op{
		Summary:     "Request register confirmation code",
		Description: "Emails a 6-digit code when the address is free. 204 empty. Account is not created yet.",
		Tags:        []string{"Auth"},
		Public:      true,
		Body:        registerCodeBody{},
		Empty:       true,
		SuccessCode: http.StatusNoContent,
		Errors:      []string{domain.CodeEmailInUse, domain.CodeRateLimited},
	})
	s.route(v1, http.MethodPost, "/auth/register", s.register, op{
		Summary:     "Register",
		Description: "Requires a valid unused register code. Creates an account, signs in, and sets cookie vynno_session. JSON body is { profile } only.",
		Tags:        []string{"Auth"},
		Public:      true,
		Body:        registerBody{},
		Success:     authResponse{},
		SuccessCode: http.StatusCreated,
		Errors:      []string{domain.CodeEmailInUse, domain.CodeInvalidCode, domain.CodeRateLimited},
		SetCookie:   true,
	})
	s.route(v1, http.MethodPost, "/auth/login", s.login, op{
		Summary:     "Login",
		Description: "Sets cookie vynno_session. JSON body is { profile } only.",
		Tags:        []string{"Auth"},
		Public:      true,
		Body:        loginBody{},
		Success:     authResponse{},
		Errors:      []string{domain.CodeInvalidCredentials, domain.CodeRateLimited},
		SetCookie:   true,
	})
	s.route(v1, http.MethodPost, "/auth/password/forgot", s.requestPasswordReset, op{
		Summary:     "Request password reset code",
		Description: "Always 204 for a well-formed email. Sends a 6-digit code only when the account exists.",
		Tags:        []string{"Auth"},
		Public:      true,
		Body:        passwordForgotBody{},
		Empty:       true,
		SuccessCode: http.StatusNoContent,
		Errors:      []string{domain.CodeRateLimited},
	})
	s.route(v1, http.MethodPost, "/auth/password/reset", s.resetPassword, op{
		Summary:     "Reset password",
		Description: "Sets a new password and revokes every session for that user. 204 empty. No cookie; log in afterwards.",
		Tags:        []string{"Auth"},
		Public:      true,
		Body:        resetPasswordBody{},
		Empty:       true,
		SuccessCode: http.StatusNoContent,
		Errors:      []string{domain.CodeInvalidCode, domain.CodeRateLimited},
	})
	s.route(v1, http.MethodGet, "/avatars/:id", s.getAvatar, op{
		Summary:     "Avatar bytes",
		Description: "Public. Success is raw image bytes with Cache-Control: public, max-age=31536000, immutable.",
		Tags:        []string{"Profile"},
		Public:      true,
		Binary:      "image/*",
		Errors:      []string{domain.CodeNotFound},
	})

	authed := v1.Group("")
	authed.Use(s.requireAuth())
	s.route(authed, http.MethodPost, "/auth/logout", s.logout, op{
		Summary:     "Logout",
		Tags:        []string{"Auth"},
		Empty:       true,
		ClearCookie: true,
	})
	s.route(authed, http.MethodPost, "/auth/password/change", s.changePassword, op{
		Summary:     "Change password",
		Description: "Signed in. Checks currentPassword, sets newPassword, and signs out every other session. Wrong current password counts toward the login failure caps. 204 empty.",
		Tags:        []string{"Auth"},
		Body:        changePasswordBody{},
		Empty:       true,
		SuccessCode: http.StatusNoContent,
		Errors:      []string{domain.CodeInvalidCredentials, domain.CodeRateLimited},
	})
	s.route(authed, http.MethodPost, "/auth/email/code", s.requestEmailChange, op{
		Summary:     "Request email change code",
		Description: "Signed in. Checks password, then mails a 6-digit code to the new address. 204 empty. The email does not change yet.",
		Tags:        []string{"Auth"},
		Body:        emailChangeCodeBody{},
		Empty:       true,
		SuccessCode: http.StatusNoContent,
		Errors:      []string{domain.CodeInvalidCredentials, domain.CodeEmailInUse, domain.CodeRateLimited},
	})
	s.route(authed, http.MethodPost, "/auth/email/change", s.changeEmail, op{
		Summary:     "Change email",
		Description: "Signed in. Requires the code sent to the new address. Switches the sign-in email, signs out every other session, and notifies the old address.",
		Tags:        []string{"Auth"},
		Body:        changeEmailBody{},
		Success:     profileDTO{},
		Errors:      []string{domain.CodeInvalidCode, domain.CodeEmailInUse, domain.CodeRateLimited},
	})
	s.route(authed, http.MethodGet, "/me", s.getMe, op{
		Summary: "Current profile",
		Tags:    []string{"Profile"},
		Success: profileDTO{},
	})
	s.route(authed, http.MethodPatch, "/me", s.patchMe, op{
		Summary:     "Update profile",
		Description: "All fields optional. Omit = leave unchanged. Do not send email or avatarUrl.",
		Tags:        []string{"Profile"},
		Body:        updateProfileBody{},
		Success:     profileDTO{},
	})
	s.route(authed, http.MethodPut, "/me/avatar", s.putMeAvatar, op{
		Summary:     "Upload avatar",
		Description: "multipart field file. jpeg, png, or webp. Max 1 MiB (magic bytes).",
		Tags:        []string{"Profile"},
		Multipart:   "file",
		Success:     profileDTO{},
		Errors:      []string{domain.CodeInvalidBody},
	})
	s.route(authed, http.MethodDelete, "/me/avatar", s.deleteMeAvatar, op{
		Summary: "Remove avatar",
		Tags:    []string{"Profile"},
		Success: profileDTO{},
	})
	s.route(authed, http.MethodGet, "/me/prefs", s.getPrefs, op{
		Summary:     "Account preferences",
		Description: "Daily target and default project. Unset values are null; the SPA applies its own defaults.",
		Tags:        []string{"Profile"},
		Success:     prefsDTO{},
	})
	s.route(authed, http.MethodPatch, "/me/prefs", s.patchPrefs, op{
		Summary:     "Update account preferences",
		Description: "All fields optional. Omit = leave unchanged. null clears. dailyTargetMs is 60000 to 86400000. defaultProjectId must be a project you own (archived is allowed).",
		Tags:        []string{"Profile"},
		Body:        updatePrefsBody{},
		Success:     prefsDTO{},
		Errors:      []string{domain.CodeNotFound},
	})

	s.route(authed, http.MethodGet, "/projects", s.listProjects, op{
		Summary:     "List projects",
		Description: "Default omits archived. Pass includeArchived=true for management UI.",
		Tags:        []string{"Projects"},
		Success:     listDTO[projectDTO]{},
		Query: []queryParam{{
			Name: "includeArchived", Type: "boolean",
			Description: "When true, include archived projects.",
		}},
		Errors: []string{domain.CodeInvalidQuery},
	})
	s.route(authed, http.MethodPost, "/projects", s.createProject, op{
		Summary:     "Create project",
		Tags:        []string{"Projects"},
		Body:        createProjectBody{},
		Success:     projectDTO{},
		SuccessCode: http.StatusCreated,
		Errors:      []string{domain.CodeCodeInUse},
	})
	s.route(authed, http.MethodGet, "/projects/:id", s.getProject, op{
		Summary: "Get project",
		Tags:    []string{"Projects"},
		Success: projectDTO{},
		Errors:  []string{domain.CodeNotFound},
	})
	s.route(authed, http.MethodPatch, "/projects/:id", s.updateProject, op{
		Summary:     "Update project",
		Description: "All fields optional. code: null clears the chip. progressPercent: null clears the dashboard bar.",
		Tags:        []string{"Projects"},
		Body:        updateProjectBody{},
		Success:     projectDTO{},
		Errors:      []string{domain.CodeNotFound, domain.CodeCodeInUse},
	})
	s.route(authed, http.MethodDelete, "/projects/:id", s.deleteProject, op{
		Summary: "Hard-delete project",
		Tags:    []string{"Projects"},
		Empty:   true,
		Errors:  []string{domain.CodeNotFound, domain.CodeLastActiveProject, domain.CodeProjectHasSessions},
	})
	s.route(authed, http.MethodPost, "/projects/:id/archive", s.archiveProject, op{
		Summary: "Archive project",
		Tags:    []string{"Projects"},
		Success: projectDTO{},
		Errors:  []string{domain.CodeNotFound, domain.CodeLastActiveProject, domain.CodeInvalidTransition},
	})
	s.route(authed, http.MethodPost, "/projects/:id/restore", s.restoreProject, op{
		Summary: "Restore project",
		Tags:    []string{"Projects"},
		Success: projectDTO{},
		Errors:  []string{domain.CodeNotFound, domain.CodeInvalidTransition},
	})
	s.route(authed, http.MethodGet, "/projects/:id/session-count", s.projectSessionCount, op{
		Summary: "Session count for project",
		Tags:    []string{"Projects"},
		Success: countDTO{},
		Errors:  []string{domain.CodeNotFound},
	})

	s.route(authed, http.MethodGet, "/activity-types", s.listActivityTypes, op{
		Summary: "List activity types",
		Tags:    []string{"Activity types"},
		Success: listDTO[activityTypeDTO]{},
	})
	s.route(authed, http.MethodPost, "/activity-types", s.createActivityType, op{
		Summary:     "Create activity type",
		Tags:        []string{"Activity types"},
		Body:        createActivityTypeBody{},
		Success:     activityTypeDTO{},
		SuccessCode: http.StatusCreated,
		Errors:      []string{domain.CodeNameInUse},
	})
	s.route(authed, http.MethodGet, "/activity-types/:id", s.getActivityType, op{
		Summary: "Get activity type",
		Tags:    []string{"Activity types"},
		Success: activityTypeDTO{},
		Errors:  []string{domain.CodeNotFound},
	})
	s.route(authed, http.MethodPatch, "/activity-types/:id", s.updateActivityType, op{
		Summary: "Update activity type",
		Tags:    []string{"Activity types"},
		Body:    updateActivityTypeBody{},
		Success: activityTypeDTO{},
		Errors:  []string{domain.CodeNotFound, domain.CodeNameInUse},
	})
	s.route(authed, http.MethodDelete, "/activity-types/:id", s.deleteActivityType, op{
		Summary: "Hard-delete activity type",
		Tags:    []string{"Activity types"},
		Empty:   true,
		Errors:  []string{domain.CodeNotFound, domain.CodeActivityTypeHasSessions},
	})
	s.route(authed, http.MethodGet, "/activity-types/:id/session-count", s.activityTypeSessionCount, op{
		Summary: "Session count for activity type",
		Tags:    []string{"Activity types"},
		Success: countDTO{},
		Errors:  []string{domain.CodeNotFound},
	})

	s.route(authed, http.MethodGet, "/sessions", s.listSessions, op{
		Summary:     "List sessions",
		Description: "Newest first. status is a comma-separated list of active, stopped. limit defaults to 20, max 100. cursor is an opaque nextCursor from the previous page.",
		Tags:        []string{"Sessions"},
		Success:     sessionListDTO{},
		Query: []queryParam{
			{Name: "status", Type: "string", Description: "Comma-separated: active, stopped."},
			{Name: "limit", Type: "integer", Description: "Positive integer, default 20, max 100."},
			{Name: "cursor", Type: "string", Description: "Opaque cursor from nextCursor. Omit on the first page."},
		},
		Errors: []string{domain.CodeInvalidQuery},
	})
	s.route(authed, http.MethodPost, "/sessions", s.startSession, op{
		Summary:     "Start session",
		Description: "409 session_already_active if one is already active. Restart is a new POST, not a resume of a stopped log.",
		Tags:        []string{"Sessions"},
		Body:        startSessionBody{},
		Success:     sessionDTO{},
		SuccessCode: http.StatusCreated,
		Errors:      []string{domain.CodeSessionAlreadyActive, domain.CodeNotFound, domain.CodeProjectArchived},
	})
	s.route(authed, http.MethodGet, "/sessions/active", s.getActiveSession, op{
		Summary:     "Active session",
		Description: "Idle → 404 session_not_active.",
		Tags:        []string{"Sessions"},
		Success:     sessionDTO{},
		Errors:      []string{domain.CodeSessionNotActive},
	})
	s.route(authed, http.MethodPost, "/sessions/manual", s.createManualSession, op{
		Summary:     "Manual time entry",
		Description: "Creates a stopped session with startedAt and endedAt. Allowed while a live session exists. Archived projects are allowed.",
		Tags:        []string{"Sessions"},
		Body:        createManualSessionBody{},
		Success:     sessionDTO{},
		SuccessCode: http.StatusCreated,
		Errors:      []string{domain.CodeNotFound},
	})
	s.route(authed, http.MethodGet, "/sessions/:id", s.getSession, op{
		Summary: "Get session",
		Tags:    []string{"Sessions"},
		Success: sessionDTO{},
		Errors:  []string{domain.CodeNotFound},
	})
	s.route(authed, http.MethodPatch, "/sessions/:id", s.updateSession, op{
		Summary:     "Update session",
		Description: "All fields optional. Omit = leave unchanged. Do not send status or id. Live endedAt must stay null; stopped endedAt must stay set.",
		Tags:        []string{"Sessions"},
		Body:        updateSessionBody{},
		Success:     sessionDTO{},
		Errors:      []string{domain.CodeNotFound},
	})
	s.route(authed, http.MethodDelete, "/sessions/:id", s.deleteSession, op{
		Summary:     "Delete session",
		Description: "Hard-delete any session, including the live timer.",
		Tags:        []string{"Sessions"},
		Empty:       true,
		Errors:      []string{domain.CodeNotFound},
	})
	s.route(authed, http.MethodPost, "/sessions/:id/stop", s.stopSession, op{
		Summary: "Stop session",
		Tags:    []string{"Sessions"},
		Success: sessionDTO{},
		Errors:  []string{domain.CodeNotFound, domain.CodeInvalidTransition},
	})

	s.route(authed, http.MethodGet, "/stats/days", s.listDayTotals, op{
		Summary:     "Tracked time per day",
		Description: "Stopped sessions summed by the local date of startedAt in timeZone, then project, then activity type. Inclusive civil dates, at most 400 days. The live session is not included.",
		Tags:        []string{"Stats"},
		Success:     listDTO[dayTotalDTO]{},
		Query: []queryParam{
			{Name: "from", Type: "string", Description: "First date, YYYY-MM-DD. Required."},
			{Name: "to", Type: "string", Description: "Last date, YYYY-MM-DD, inclusive. Required."},
			{Name: "timeZone", Type: "string", Description: "IANA time zone for the dates, e.g. Europe/Warsaw. Required."},
		},
		Errors: []string{domain.CodeInvalidQuery},
	})

	s.mountDocs(r)
	notFound := func(c *gin.Context) {
		writeError(c, domain.ErrNotFound())
	}
	r.NoRoute(notFound)
	r.NoMethod(notFound)
	return r
}

func corsAllowOrigins(spa []string, publicOrigin string) []string {
	out := append([]string{}, spa...)
	if publicOrigin == "" {
		return out
	}
	for _, o := range out {
		if o == publicOrigin {
			return out
		}
	}
	return append(out, publicOrigin)
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
