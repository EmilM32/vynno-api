package domain

// Prefs are account-wide settings that follow the user across devices.
// A nil field is unset; the SPA applies its own default.
type Prefs struct {
	DailyTargetMs    *int64
	DefaultProjectID *string
}

const (
	DailyTargetMinMs int64 = 60_000
	DailyTargetMaxMs int64 = 24 * 60 * 60 * 1000
)

// ValidateDailyTargetMs keeps the goal between one minute and one day.
func ValidateDailyTargetMs(ms int64) error {
	if ms < DailyTargetMinMs || ms > DailyTargetMaxMs {
		return ErrInvalidBody("dailyTargetMs must be between 60000 and 86400000.")
	}
	return nil
}
