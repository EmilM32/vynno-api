package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
)

func (s *Server) mountDocs(r *gin.Engine) {
	s.buildSpec()
	r.GET("/openapi.json", s.serveOpenAPI)
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/")
	})
	ui := v5emb.NewHandlerWithConfig(swgui.Config{
		Title:       "Vynno API",
		SwaggerJSON: "/openapi.json",
		BasePath:    "/swagger/",
		SettingsUI: map[string]string{
			"withCredentials":      "true",
			"persistAuthorization": "true",
		},
	})
	r.Any("/swagger/*any", gin.WrapH(ui))
}
