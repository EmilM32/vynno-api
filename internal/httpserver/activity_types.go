package httpserver

import (
	"net/http"

	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-gonic/gin"
)

func (s *Server) listActivityTypes(c *gin.Context) {
	items, err := s.userSvc(c).ListActivityTypes(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	dtos := make([]activityTypeDTO, 0, len(items))
	for _, a := range items {
		dtos = append(dtos, toActivityTypeDTO(a))
	}
	c.JSON(http.StatusOK, listDTO[activityTypeDTO]{Items: dtos})
}

func (s *Server) getActivityType(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	a, err := s.userSvc(c).GetActivityType(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityTypeDTO(a))
}

func (s *Server) createActivityType(c *gin.Context) {
	var body createActivityTypeBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	a, err := s.userSvc(c).CreateActivityType(c.Request.Context(), service.CreateActivityTypeInput{
		Name: body.Name, Color: body.Color,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toActivityTypeDTO(a))
}

func (s *Server) updateActivityType(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	var body updateActivityTypeBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	a, err := s.userSvc(c).UpdateActivityType(c.Request.Context(), id, service.UpdateActivityTypeInput{
		Name: body.Name, Color: body.Color,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toActivityTypeDTO(a))
}

func (s *Server) deleteActivityType(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	if err := s.userSvc(c).DeleteActivityType(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) activityTypeSessionCount(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	n, err := s.userSvc(c).ActivityTypeSessionCount(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": n})
}
