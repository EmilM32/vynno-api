package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) listSessions(c *gin.Context) {
	statuses, err := parseStatusQuery(c.Query("status"))
	if err != nil {
		writeError(c, err)
		return
	}
	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		writeError(c, err)
		return
	}
	items, err := s.userSvc(c).ListSessions(c.Request.Context(), statuses, limit)
	if err != nil {
		writeError(c, err)
		return
	}
	dtos := make([]sessionDTO, 0, len(items))
	for _, sess := range items {
		dtos = append(dtos, toSessionDTO(sess))
	}
	c.JSON(http.StatusOK, listDTO[sessionDTO]{Items: dtos})
}

func (s *Server) getSession(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	sess, err := s.userSvc(c).GetSession(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSessionDTO(sess))
}

func (s *Server) getActiveSession(c *gin.Context) {
	sess, err := s.userSvc(c).GetActiveSession(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSessionDTO(sess))
}

func (s *Server) startSession(c *gin.Context) {
	var body startSessionBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	if strings.TrimSpace(body.ProjectID) == "" {
		writeError(c, domain.ErrInvalidBody("projectId is required."))
		return
	}
	sess, err := s.userSvc(c).StartSession(c.Request.Context(), service.StartSessionInput{
		ProjectID:        body.ProjectID,
		Note:             body.Note,
		TicketID:         body.TicketID,
		ActivityTypeID:   body.ActivityTypeID,
		Tags:             body.Tags,
		TargetDurationMs: body.TargetDurationMs,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toSessionDTO(sess))
}

func (s *Server) pauseSession(c *gin.Context) {
	s.sessionVerb(c, s.userSvc(c).PauseSession)
}

func (s *Server) resumeSession(c *gin.Context) {
	s.sessionVerb(c, s.userSvc(c).ResumeSession)
}

func (s *Server) stopSession(c *gin.Context) {
	s.sessionVerb(c, s.userSvc(c).StopSession)
}

func (s *Server) sessionVerb(c *gin.Context, fn func(context.Context, uuid.UUID) (domain.Session, error)) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	sess, err := fn(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSessionDTO(sess))
}

func parseStatusQuery(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !domain.ValidStatusFilter(p) {
			return nil, domain.ErrInvalidQuery("status must be a comma-separated list of active, paused, stopped.")
		}
		out = append(out, p)
	}
	return out, nil
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, domain.ErrInvalidQuery("limit must be a positive integer.")
	}
	return n, nil
}
