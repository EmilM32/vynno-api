package httpserver

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-gonic/gin"
)

type registerBody struct {
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	DisplayName *string `json:"displayName"`
	RememberMe  *bool   `json:"rememberMe"`
}

type loginBody struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe *bool  `json:"rememberMe"`
}

type authResponse struct {
	Profile profileDTO `json:"profile"`
}

func (s *Server) register(c *gin.Context) {
	var body registerBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	res, err := s.svc.Register(c.Request.Context(), service.RegisterInput{
		Username:    body.Username,
		Password:    body.Password,
		DisplayName: body.DisplayName,
		RememberMe:  body.RememberMe,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	s.setSessionCookie(c, res.Token, res.RememberMe)
	c.JSON(http.StatusCreated, authResponse{Profile: s.toProfileDTO(res.Profile)})
}

func (s *Server) login(c *gin.Context) {
	var body loginBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	res, err := s.svc.Login(c.Request.Context(), service.LoginInput{
		Username:   body.Username,
		Password:   body.Password,
		RememberMe: body.RememberMe,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	s.setSessionCookie(c, res.Token, res.RememberMe)
	c.JSON(http.StatusOK, authResponse{Profile: s.toProfileDTO(res.Profile)})
}

func (s *Server) logout(c *gin.Context) {
	if err := s.svc.Logout(c.Request.Context(), sessionFromRequest(c)); err != nil {
		writeError(c, err)
		return
	}
	s.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := sessionFromRequest(c)
		if raw == "" {
			writeError(c, domain.ErrUnauthorized())
			c.Abort()
			return
		}
		userID, err := s.svc.ResolveToken(c.Request.Context(), raw)
		if err != nil {
			writeError(c, err)
			c.Abort()
			return
		}
		if err := s.checkCSRF(c); err != nil {
			writeError(c, err)
			c.Abort()
			return
		}
		c.Set(ctxUserID, userID)
		c.Next()
	}
}

func sessionFromRequest(c *gin.Context) string {
	if v, err := c.Cookie(sessionCookie); err == nil && v != "" {
		return v
	}
	h := c.GetHeader("Authorization")
	const prefix = "bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

func (s *Server) checkCSRF(c *gin.Context) error {
	cookie, err := c.Cookie(sessionCookie)
	if err != nil || cookie == "" {
		return nil
	}
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return nil
	}
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = originFromReferer(c.GetHeader("Referer"))
	}
	if origin == "" {
		return nil
	}
	if _, ok := s.spaOrigins[origin]; !ok {
		return domain.ErrUnauthorized()
	}
	return nil
}

func originFromReferer(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
