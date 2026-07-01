package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/dromara/carbon/v2"
)

type ExpenseTemplate struct {
	Id                        int
	Amount                    int
	Description               string
	InitialToBePaidOn         time.Time
	RepeatabilityIntervalUnit int
	RepeatabilityIntervalPace string
	IsOnHold                  bool
}

type ExpenseTemplateOption func(*ExpenseTemplate)

func NewExpenseTemplate(amnt int, desc string, opt ...ExpenseTemplateOption) *ExpenseTemplate {
	expenseTemplate := &ExpenseTemplate{
		Amount:      amnt,
		Description: desc,
	}

	for _, o := range opt {
		o(expenseTemplate)
	}

	return expenseTemplate
}

func WithInitialToBePaidOn(y int, m time.Month, d int) ExpenseTemplateOption {
	return func(e *ExpenseTemplate) {
		e.InitialToBePaidOn = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
}

func WithRepeatabilityInterval(unit int, pace string) ExpenseTemplateOption {
	return func(e *ExpenseTemplate) {
		e.RepeatabilityIntervalUnit = unit
		e.RepeatabilityIntervalPace = pace
	}
}

func (tpl *ExpenseTemplate) GenerateRepeatingExpenses(dateRange DateRange) ([]*Expense, error) {
	expenses := []*Expense{}
	cTime := carbon.NewCarbon(tpl.InitialToBePaidOn)

	// Here, we don't want to also do cTime.Gte(carbon.NewCarbon(dateRange.From))
	// because if we do, we never start the loop since the condition is never met
	// when the InitialToBePaidOn happened before the dateRange.From.
	// And we're not concerned about filtering out paid expenses just yet, so if
	// there are past unpaid expenses, we do want them in those results.
	for cTime.Lte(carbon.NewCarbon(dateRange.To)) {
		switch pace := tpl.RepeatabilityIntervalPace; pace {
		case "D":
			cTime = cTime.AddDays(tpl.RepeatabilityIntervalUnit)
		case "W":
			cTime = cTime.AddWeeks(tpl.RepeatabilityIntervalUnit)
		case "M":
			cTime = cTime.AddMonths(tpl.RepeatabilityIntervalUnit)
		case "Y":
			cTime = cTime.AddYears(tpl.RepeatabilityIntervalUnit)
		default:
			return []*Expense{}, errors.New(fmt.Sprintf("Time interval pace '%s' not supported", pace))
		}
		expenses = append(expenses, NewExpense(tpl.Amount, cTime.StdTime()))
	}

	return expenses, nil
}
