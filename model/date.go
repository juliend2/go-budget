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
		return d.SetDay(15).StdTime()
	} else {
		// past month's last day:
		return d.StartOfMonth().SubDay().StdTime()
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
