package store

import (
	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/google/uuid"
)

func DefaultUserID() uuid.UUID {
	return uuid.MustParse("00000000-0000-4000-8000-000000000001")
}

func DefaultProfile() domain.Profile {
	return domain.Profile{
		DisplayName: "Alex Dev",
	}
}

func DefaultProject() domain.Project {
	code := "AUTH"
	return domain.Project{
		ID:       uuid.MustParse("00000000-0000-4000-8000-000000000010").String(),
		Name:     "Identity",
		Color:    "#3b82f6",
		Code:     &code,
		Archived: false,
	}
}
