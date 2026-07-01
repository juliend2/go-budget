package model

import (
	"time"

	"github.com/dromara/carbon/v2"
)

type DateRange struct {
	From time.Time
	To   time.Time
}

type Expense struct {
	now        time.Time // DI for testing
	Amount     int       // dollars
	ToBePaidAt time.Time
}

type ExpenseOption func(*Expense)

func NewExpense(amount int, toBePaidAt time.Time, opt ...ExpenseOption) *Expense {
	expense := &Expense{
		now:        time.Now(),
		Amount:     amount,
		ToBePaidAt: toBePaidAt,
	}

	for _, o := range opt {
		o(expense)
	}

	return expense
}

func WithNow(now time.Time) ExpenseOption {
	return func(e *Expense) {
		e.now = now
	}
}

func (e *Expense) IsDue() bool {
	return !e.now.Before(e.ToBePaidAt)
}

func GetPayDays(dateRange DateRange) []time.Time {
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

func PutExpensesInTheirPayPeriods(pays []time.Time, exps []*Expense) map[string][]*Expense {
	acc := make(map[string][]*Expense)

	for i := range pays {
		from := carbon.NewCarbon(pays[i])
		to := carbon.ZeroValue()
		if i != len(pays)-1 {
			to = carbon.NewCarbon(pays[i+1])
		}

		for _, exp := range exps {
			toBePaidAt := carbon.NewCarbon(exp.ToBePaidAt)
			if toBePaidAt.Gte(from) && (to.IsZero() || toBePaidAt.Lt(to)) {
				_, ok := acc[from.ToDateString()]
				if ok {
					acc[from.ToDateString()] = append(acc[from.ToDateString()], exp)
				} else {
					acc[from.ToDateString()] = []*Expense{exp}
				}
			}
		}
	}

	return acc
}
