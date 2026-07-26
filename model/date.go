package model

import (
	"time"

	"github.com/dromara/carbon/v2"
)

func Date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func GetPreviousPayDate(t time.Time) time.Time {
	d := carbon.NewCarbon(t)
	if d.Day() > 15 {
		return d.SetDay(15).StartOfDay().StdTime()
	} else {
		// past month's last day:
		return d.StartOfMonth().SubDay().StartOfDay().StdTime()
	}
}

func GetNextPayDate(t time.Time) time.Time {
	d := carbon.NewCarbon(t)
	if d.Day() > 15 {
		return d.EndOfMonth().StdTime()
	} else {
		return d.SetDay(15).StdTime()
	}
}

// nextPayDateAfter returns the first pay day (15th or last day of the month)
// strictly after t, normalized to the start of the day.
func nextPayDateAfter(t *carbon.Carbon) *carbon.Carbon {
	d := t.StartOfDay()
	if d.Day() < 15 {
		return d.SetDay(15)
	}
	// last day of the current month at 00:00:00
	lastDay := d.AddMonth().StartOfMonth().SubDay()
	if d.Day() < lastDay.Day() {
		return lastDay
	}
	// d is the last day of the month -> 15th of next month
	return d.AddMonth().SetDay(15)
}
