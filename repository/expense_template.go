package repository

import (
	"errors"
	"fmt"
	"time"

	"desrosiers.org/budget/model"
	"github.com/dromara/carbon/v2"
)

func GenerateRepeatingExpenses(expTpl *model.ExpenseTemplate, dtRange model.DateRange) ([]*model.Expense, error) {
	if dtRange.To.Before(expTpl.InitialToBePaidOn) {
		return []*model.Expense{}, nil // out-or-range
	}

	cTime := carbon.NewCarbon(expTpl.InitialToBePaidOn)
	// TODO: create a var that will contain the expenses and that we will push
	// expenses into

	for cTime.Lt(carbon.NewCarbon(expTpl.InitialToBePaidOn)) {
		switch pace := expTpl.RepeatabilityIntervalPace; pace {
		case "m":
			cTime = cTime.AddMonths(expTpl.RepeatabilityIntervalUnit)
		default:
			return []*model.Expense{}, errors.New(fmt.Sprintf("Time interval pace '%s' not supported", pace))
		}

	}

	return []*model.Expense{
		model.NewExpense(100, model.Date(2026, time.June, 11)),
	}, nil
}
