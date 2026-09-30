package httpserver

import (
	"net/http"

	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/gin-gonic/gin"
)

type dayTotalDTO struct {
	Date           string  `json:"date"`
	ProjectID      string  `json:"projectId"`
	ActivityTypeID *string `json:"activityTypeId"`
	DurationMs     int64   `json:"durationMs"`
	SessionCount   int     `json:"sessionCount"`
}

func (s *Server) listDayTotals(c *gin.Context) {
	r, err := domain.ParseDayRange(c.Query("from"), c.Query("to"), c.Query("timeZone"))
	if err != nil {
		writeError(c, err)
		return
	}
	totals, err := s.userSvc(c).DayTotals(c.Request.Context(), r)
	if err != nil {
		writeError(c, err)
		return
	}
	items := make([]dayTotalDTO, 0, len(totals))
	for _, t := range totals {
		items = append(items, dayTotalDTO{
			Date:           t.Date,
			ProjectID:      t.ProjectID,
			ActivityTypeID: t.ActivityTypeID,
			DurationMs:     t.DurationMs,
			SessionCount:   t.SessionCount,
		})
	}
	c.JSON(http.StatusOK, listDTO[dayTotalDTO]{Items: items})
}
