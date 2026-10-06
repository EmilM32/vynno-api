package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
)

func (s *Server) mountDocs(r *gin.Engine) {
	s.buildSpec()
	r.GET("/openapi.json", docsHeaders, s.serveOpenAPI)
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
	r.Any("/swagger/*any", docsHeaders, gin.WrapH(ui))
}

// docsHeaders stops other sites from framing Swagger UI. It runs on the API origin,
// which the cookie CSRF check trusts, so a framed page could be clickjacked into
// Try-it-out calls with the operator's session.
func docsHeaders(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Content-Security-Policy", "frame-ancestors 'none'")
	c.Next()
}
