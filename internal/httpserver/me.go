package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getMe(c *gin.Context) {
	p, err := s.userSvc(c).GetProfile(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProfileDTO(p))
}
