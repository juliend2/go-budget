package repository

import (
	"time"

	"desrosiers.org/budget/model"
	"github.com/dromara/carbon/v2"
)

func GetPayDays(dateRange model.DateRange) []time.Time {
	dates := []time.Time{}
	firstValue := carbon.NewCarbon(dateRange.From)
	d := dateRange.From.Day()
	if d > 15 {
		firstValue = firstValue.EndOfMonth()
	} else if d < 15 {
		firstValue = firstValue.SetDay(15)
	}
	// is 15; do nothing

	dates = append(dates, firstValue.StdTime())

	for carbon.NewCarbon(dates[len(dates)-1]).Lt(carbon.NewCarbon(dateRange.To)) {
		value := carbon.NewCarbon(dates[len(dates)-1])
		if value.Day() == 15 {
			value = value.EndOfMonth()
		} else {
			// Assume it can only be 15th or last-of-month
			// so go to 1st of next month and set the day to 15th
			value = value.AddDays(1).SetDay(15)
		}
		dates = append(dates, value.StdTime())
	}

	return dates
}
