package httpserver

import (
	"encoding/json"
	"time"

	"github.com/EmilM32/vynno-api/internal/domain"
)

const isoMilli = "2006-01-02T15:04:05.000Z07:00"

type profileDTO struct {
	DisplayName string  `json:"displayName"`
	Handle      string  `json:"handle"`
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
	ID               string   `json:"id"`
	ProjectID        string   `json:"projectId"`
	Note             string   `json:"note"`
	TicketID         *string  `json:"ticketId"`
	ActivityType     *string  `json:"activityType"`
	Tags             []string `json:"tags"`
	Status           string   `json:"status"`
	StartedAt        string   `json:"startedAt"`
	EndedAt          *string  `json:"endedAt"`
	PausedMs         int64    `json:"pausedMs"`
	PausedAt         *string  `json:"pausedAt"`
	TargetDurationMs *int64   `json:"targetDurationMs"`
}

type listDTO[T any] struct {
	Items []T `json:"items"`
}

type createProjectBody struct {
	Name  string  `json:"name"`
	Color string  `json:"color"`
	Code  *string `json:"code"`
}

type updateProjectBody struct {
	Name    *string
	Color   *string
	Code    *string
	CodeSet bool
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
	return nil
}

type startSessionBody struct {
	ProjectID        string   `json:"projectId"`
	Note             string   `json:"note"`
	TicketID         *string  `json:"ticketId"`
	ActivityType     *string  `json:"activityType"`
	Tags             []string `json:"tags"`
	TargetDurationMs *int64   `json:"targetDurationMs"`
}

func toProfileDTO(p domain.Profile) profileDTO {
	return profileDTO{DisplayName: p.DisplayName, Handle: p.Handle, AvatarURL: p.AvatarURL}
}

func toProjectDTO(p domain.Project) projectDTO {
	return projectDTO{
		ID: p.ID, Name: p.Name, Color: p.Color, Code: p.Code,
		ProgressPercent: p.ProgressPercent, Archived: p.Archived,
	}
}

func toSessionDTO(s domain.Session) sessionDTO {
	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}
	return sessionDTO{
		ID:               s.ID,
		ProjectID:        s.ProjectID,
		Note:             s.Note,
		TicketID:         s.TicketID,
		ActivityType:     s.ActivityType,
		Tags:             tags,
		Status:           s.Status,
		StartedAt:        formatTime(s.StartedAt),
		EndedAt:          formatTimePtr(s.EndedAt),
		PausedMs:         s.PausedMs,
		PausedAt:         formatTimePtr(s.PausedAt),
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
