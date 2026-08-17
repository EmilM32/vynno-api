package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-gonic/gin"
)

const maxAvatarMultipart = 2 << 20

type updateProfileBody struct {
	DisplayName    *string
	DisplayNameSet bool
}

func (u *updateProfileBody) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["handle"]; ok {
		return fmt.Errorf("handle cannot be updated")
	}
	if _, ok := raw["avatarUrl"]; ok {
		return fmt.Errorf("use PUT /me/avatar to set a photo")
	}
	if v, ok := raw["displayName"]; ok {
		u.DisplayNameSet = true
		if string(v) == "null" {
			return fmt.Errorf("display name is required")
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.DisplayName = &s
	}
	return nil
}

func (s *Server) getMe(c *gin.Context) {
	p, err := s.userSvc(c).GetProfile(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, s.toProfileDTO(p))
}

func (s *Server) patchMe(c *gin.Context) {
	var body updateProfileBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	in := service.UpdateProfileInput{}
	if body.DisplayNameSet {
		in.DisplayName = body.DisplayName
	}
	p, err := s.userSvc(c).UpdateProfile(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, s.toProfileDTO(p))
}

func (s *Server) putMeAvatar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarMultipart)
	file, err := c.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(c, domain.ErrInvalidBody("Avatar must be at most 1 MiB."))
			return
		}
		writeError(c, domain.ErrInvalidBody("Avatar file is required."))
		return
	}
	src, err := file.Open()
	if err != nil {
		writeError(c, domain.ErrInvalidBody("Avatar file is required."))
		return
	}
	defer func() { _ = src.Close() }()
	data, err := io.ReadAll(io.LimitReader(src, domain.AvatarMaxBytes+1))
	if err != nil {
		writeError(c, domain.ErrInvalidBody("Avatar file is required."))
		return
	}
	p, err := s.userSvc(c).ReplaceAvatar(c.Request.Context(), data)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, s.toProfileDTO(p))
}

func (s *Server) deleteMeAvatar(c *gin.Context) {
	p, err := s.userSvc(c).DeleteAvatar(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, s.toProfileDTO(p))
}

func (s *Server) getAvatar(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	av, err := s.svc.GetAvatar(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Data(http.StatusOK, av.ContentType, av.Bytes)
}
