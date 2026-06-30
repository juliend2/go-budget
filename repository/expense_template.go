package repository

import (
	"time"

	// "github.com/dromara/carbon/v2"
	"desrosiers.org/budget/model"
)

func GenerateRepeatingExpenses(expTpl *model.ExpenseTemplate, dtRange model.DateRange) []*model.Expense {
	if dtRange.To.Before(expTpl.InitialToBePaidOn) {
		return []*model.Expense{} // out-or-range
	}

	return []*model.Expense{
		model.NewExpense(100, model.Date(2026, time.June, 11)),
	}
}
