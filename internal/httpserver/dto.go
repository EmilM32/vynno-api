package httpserver

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
)

const isoMilli = "2006-01-02T15:04:05.000Z07:00"

type profileDTO struct {
	DisplayName string  `json:"displayName"`
	Email       string  `json:"email"`
	AvatarURL   *string `json:"avatarUrl"`
}

type projectDTO struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Color           string  `json:"color"`
	Code            *string `json:"code"`
	ProgressPercent *int    `json:"progressPercent"`
	Archived        bool    `json:"archived"`
}

type sessionDTO struct {
	ID               string  `json:"id"`
	ProjectID        string  `json:"projectId"`
	Note             string  `json:"note"`
	TicketID         *string `json:"ticketId"`
	ActivityTypeID   *string `json:"activityTypeId"`
	Status           string  `json:"status"`
	StartedAt        string  `json:"startedAt"`
	EndedAt          *string `json:"endedAt"`
	TargetDurationMs *int64  `json:"targetDurationMs"`
}

type listDTO[T any] struct {
	Items []T `json:"items"`
}

type sessionListDTO struct {
	Items      []sessionDTO `json:"items"`
	NextCursor *string      `json:"nextCursor"`
}

type createProjectBody struct {
	Name            string  `json:"name"`
	Color           string  `json:"color"`
	Code            *string `json:"code"`
	ProgressPercent *int    `json:"progressPercent"`
}

type countDTO struct {
	Count int `json:"count"`
}

type updateProjectBody struct {
	Name            *string `json:"name"`
	Color           *string `json:"color"`
	Code            *string `json:"code"`
	CodeSet         bool    `json:"-"`
	ProgressPercent *int    `json:"progressPercent"`
	ProgressSet     bool    `json:"-"`
}

func (u *updateProjectBody) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["name"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.Name = &s
	}
	if v, ok := raw["color"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.Color = &s
	}
	if v, ok := raw["code"]; ok {
		u.CodeSet = true
		if string(v) != "null" {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			u.Code = &s
		}
	}
	if v, ok := raw["progressPercent"]; ok {
		u.ProgressSet = true
		if string(v) != "null" {
			var n int
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			u.ProgressPercent = &n
		}
	}
	return nil
}

type startSessionBody struct {
	ProjectID        string  `json:"projectId"`
	Note             string  `json:"note"`
	TicketID         *string `json:"ticketId"`
	ActivityTypeID   *string `json:"activityTypeId"`
	TargetDurationMs *int64  `json:"targetDurationMs"`
}

type createManualSessionBody struct {
	ProjectID        string  `json:"projectId"`
	Note             string  `json:"note"`
	TicketID         *string `json:"ticketId"`
	ActivityTypeID   *string `json:"activityTypeId"`
	TargetDurationMs *int64  `json:"targetDurationMs"`
	StartedAt        string  `json:"startedAt"`
	EndedAt          string  `json:"endedAt"`
}

type updateSessionBody struct {
	ProjectID        *string `json:"projectId"`
	Note             *string `json:"note"`
	TicketID         *string `json:"ticketId"`
	TicketSet        bool    `json:"-"`
	ActivityTypeID   *string `json:"activityTypeId"`
	ActivityTypeSet  bool    `json:"-"`
	StartedAt        *string `json:"startedAt"`
	EndedAt          *string `json:"endedAt"`
	EndedSet         bool    `json:"-"`
	TargetDurationMs *int64  `json:"targetDurationMs"`
	TargetSet        bool    `json:"-"`
}

func (u *updateSessionBody) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if _, ok := raw["status"]; ok {
		return errWritable("status is not writable; use stop.")
	}
	if _, ok := raw["pausedAt"]; ok {
		return errWritable("pausedAt is not writable.")
	}
	if _, ok := raw["pausedMs"]; ok {
		return errWritable("pausedMs is not writable.")
	}
	if _, ok := raw["id"]; ok {
		return errWritable("id is not writable.")
	}
	if v, ok := raw["projectId"]; ok {
		if string(v) == "null" {
			return errWritable("projectId cannot be null.")
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.ProjectID = &s
	}
	if v, ok := raw["note"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.Note = &s
	}
	if v, ok := raw["ticketId"]; ok {
		u.TicketSet = true
		if string(v) != "null" {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			u.TicketID = &s
		}
	}
	if v, ok := raw["activityTypeId"]; ok {
		u.ActivityTypeSet = true
		if string(v) != "null" {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			u.ActivityTypeID = &s
		}
	}
	if v, ok := raw["startedAt"]; ok {
		if string(v) == "null" {
			return errWritable("startedAt cannot be null.")
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.StartedAt = &s
	}
	if v, ok := raw["endedAt"]; ok {
		u.EndedSet = true
		if string(v) != "null" {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			u.EndedAt = &s
		}
	}
	if v, ok := raw["targetDurationMs"]; ok {
		u.TargetSet = true
		if string(v) != "null" {
			var n int64
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			u.TargetDurationMs = &n
		}
	}
	return nil
}

func errWritable(msg string) error {
	return domain.ErrInvalidBody(msg)
}

type activityTypeDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type createActivityTypeBody struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type updateActivityTypeBody struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (u *updateActivityTypeBody) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["name"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.Name = &s
	}
	if v, ok := raw["color"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return err
		}
		u.Color = &s
	}
	return nil
}

func (s *Server) toProfileDTO(p domain.Profile) profileDTO {
	return profileDTO{DisplayName: p.DisplayName, Email: p.Email, AvatarURL: s.absoluteAvatarURL(p.AvatarURL)}
}

func (s *Server) absoluteAvatarURL(stored *string) *string {
	if stored == nil || *stored == "" {
		return nil
	}
	path := *stored
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return stored
	}
	abs := strings.TrimRight(s.publicOrigin, "/") + path
	return &abs
}

func toActivityTypeDTO(a domain.ActivityType) activityTypeDTO {
	return activityTypeDTO{ID: a.ID, Name: a.Name, Color: a.Color}
}

func toProjectDTO(p domain.Project) projectDTO {
	return projectDTO{
		ID: p.ID, Name: p.Name, Color: p.Color, Code: p.Code,
		ProgressPercent: p.ProgressPercent, Archived: p.Archived,
	}
}

func toSessionDTO(s domain.Session) sessionDTO {
	return sessionDTO{
		ID:               s.ID,
		ProjectID:        s.ProjectID,
		Note:             s.Note,
		TicketID:         s.TicketID,
		ActivityTypeID:   s.ActivityTypeID,
		Status:           s.Status,
		StartedAt:        formatTime(s.StartedAt),
		EndedAt:          formatTimePtr(s.EndedAt),
		TargetDurationMs: s.TargetDurationMs,
	}
}

func formatTime(t time.Time) string {
	return t.UTC().Format(isoMilli)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatTime(*t)
	return &s
}
