package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/gin-gonic/gin"
)

type prefsDTO struct {
	DailyTargetMs    *int64  `json:"dailyTargetMs"`
	DefaultProjectID *string `json:"defaultProjectId"`
}

type updatePrefsBody struct {
	DailyTargetMs     *int64  `json:"dailyTargetMs"`
	DailyTargetSet    bool    `json:"-"`
	DefaultProjectID  *string `json:"defaultProjectId"`
	DefaultProjectSet bool    `json:"-"`
}

func (u *updatePrefsBody) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k := range raw {
		switch k {
		case "dailyTargetMs", "defaultProjectId":
		default:
			return domain.ErrInvalidBody("Unknown field.")
		}
	}
	if v, ok := raw["dailyTargetMs"]; ok {
		u.DailyTargetSet = true
		if string(v) != "null" {
			var n int64
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			u.DailyTargetMs = &n
		}
	}
	if v, ok := raw["defaultProjectId"]; ok {
		u.DefaultProjectSet = true
		if string(v) != "null" {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			u.DefaultProjectID = &s
		}
	}
	return nil
}

func toPrefsDTO(p domain.Prefs) prefsDTO {
	return prefsDTO{DailyTargetMs: p.DailyTargetMs, DefaultProjectID: p.DefaultProjectID}
}

func (s *Server) getPrefs(c *gin.Context) {
	p, err := s.userSvc(c).GetPrefs(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPrefsDTO(p))
}

func (s *Server) patchPrefs(c *gin.Context) {
	var body updatePrefsBody
	if err := decodeJSON(c, &body); err != nil {
		writeError(c, err)
		return
	}
	p, err := s.userSvc(c).UpdatePrefs(c.Request.Context(), service.UpdatePrefsInput{
		DailyTargetMs:     body.DailyTargetMs,
		DailyTargetSet:    body.DailyTargetSet,
		DefaultProjectID:  body.DefaultProjectID,
		DefaultProjectSet: body.DefaultProjectSet,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPrefsDTO(p))
}
