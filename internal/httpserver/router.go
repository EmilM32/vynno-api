package httpserver

import "github.com/gin-gonic/gin"

// NewRouter builds the HTTP API. Phase 1 only serves health.
func NewRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	_ = r.SetTrustedProxies(nil)
	r.GET("/healthz", handleHealth)
	return r
}
