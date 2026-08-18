package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type healthResponse struct {
	Status string `json:"status"`
}

func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

func (s *Server) handleReady(c *gin.Context) {
	if s.ready != nil {
		if err := s.ready(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, healthResponse{Status: "not_ready"})
			return
		}
	}
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
