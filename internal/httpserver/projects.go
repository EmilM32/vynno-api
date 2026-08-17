package httpserver

import (
	"net/http"
	"strings"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) listProjects(c *gin.Context) {
	include, err := parseIncludeArchived(c.Query("includeArchived"))
	if err != nil {
		writeError(c, err)
		return
	}
	items, err := s.userSvc(c).ListProjects(c.Request.Context(), include)
	if err != nil {
		writeError(c, err)
		return
	}
	dtos := make([]projectDTO, 0, len(items))
	for _, p := range items {
		dtos = append(dtos, toProjectDTO(p))
	}
	c.JSON(http.StatusOK, listDTO[projectDTO]{Items: dtos})
}

func (s *Server) getProject(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	p, err := s.userSvc(c).GetProject(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectDTO(p))
}

func (s *Server) createProject(c *gin.Context) {
	var body createProjectBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	p, err := s.userSvc(c).CreateProject(c.Request.Context(), service.CreateProjectInput{
		Name: body.Name, Color: body.Color, Code: body.Code,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProjectDTO(p))
}

func (s *Server) updateProject(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	var body updateProjectBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	p, err := s.userSvc(c).UpdateProject(c.Request.Context(), id, service.UpdateProjectInput{
		Name: body.Name, Color: body.Color, Code: body.Code, CodeSet: body.CodeSet,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectDTO(p))
}

func (s *Server) deleteProject(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	if err := s.userSvc(c).DeleteProject(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) archiveProject(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	p, err := s.userSvc(c).ArchiveProject(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectDTO(p))
}

func (s *Server) restoreProject(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	p, err := s.userSvc(c).RestoreProject(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectDTO(p))
}

func (s *Server) projectSessionCount(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	n, err := s.userSvc(c).ProjectSessionCount(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": n})
}

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, domain.ErrNotFound()
	}
	return id, nil
}

func parseIncludeArchived(raw string) (bool, error) {
	if raw == "" {
		return false, nil
	}
	switch strings.ToLower(raw) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, domain.ErrInvalidQuery("includeArchived must be true or false.")
	}
}
