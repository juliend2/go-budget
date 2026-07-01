package repository

import (
	"errors"
	"fmt"

	"desrosiers.org/budget/model"
	"github.com/dromara/carbon/v2"
)

func GenerateRepeatingExpenses(expTpl *model.ExpenseTemplate, dateRange model.DateRange) ([]*model.Expense, error) {
	expenses := []*model.Expense{}
	cTime := carbon.NewCarbon(expTpl.InitialToBePaidOn)

	// Here, we don't want to also do cTime.Gte(carbon.NewCarbon(dateRange.From))
	// because if we do, we never start the loop since the condition is never met
	// when the InitialToBePaidOn happened before the dateRange.From.
	// And we're not concerned about filtering out paid expenses just yet, so if
	// there are past unpaid expenses, we do want them in those results.
	for cTime.Lt(carbon.NewCarbon(dateRange.To)) {
		switch pace := expTpl.RepeatabilityIntervalPace; pace {
		case "D":
			cTime = cTime.AddDays(expTpl.RepeatabilityIntervalUnit)
		case "W":
			cTime = cTime.AddWeeks(expTpl.RepeatabilityIntervalUnit)
		case "M":
			cTime = cTime.AddMonths(expTpl.RepeatabilityIntervalUnit)
		case "Y":
			cTime = cTime.AddYears(expTpl.RepeatabilityIntervalUnit)
		default:
			return []*model.Expense{}, errors.New(fmt.Sprintf("Time interval pace '%s' not supported", pace))
		}
		expenses = append(expenses, model.NewExpense(expTpl.Amount, cTime.StdTime()))
	}

	return expenses, nil
}
