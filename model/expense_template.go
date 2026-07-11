package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/dromara/carbon/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ExpenseTemplate struct {
	ID                        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Amount                    int                `bson:"amount" json:"amount"`
	Description               string             `bson:"description" json:"description"`
	InitialToBePaidOn         time.Time          `bson:"initial_to_be_paid_on" json:"initial_to_be_paid_on"`
	RepeatabilityIntervalUnit int                `bson:"repeatability_interval_unit" json:"repeatability_interval_unit"`
	RepeatabilityIntervalPace string             `bson:"repeatability_interval_pace" json:"repeatability_interval_pace"`
	IsOnHold                  bool               `bson:"is_on_hold" json:"is_on_hold"`
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

func WithTemplateIDVal(id primitive.ObjectID) ExpenseTemplateOption {
	return func(e *ExpenseTemplate) {
		e.ID = id
	}
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

func (tpl *ExpenseTemplate) getNextToBePaidAt(t *carbon.Carbon) (*carbon.Carbon, error) {
	switch pace := tpl.RepeatabilityIntervalPace; pace {
	case "D":
		return t.AddDays(tpl.RepeatabilityIntervalUnit), nil
	case "W":
		return t.AddWeeks(tpl.RepeatabilityIntervalUnit), nil
	case "M":
		return t.AddMonths(tpl.RepeatabilityIntervalUnit), nil
	case "Y":
		return t.AddYears(tpl.RepeatabilityIntervalUnit), nil
	default:
		return carbon.ZeroValue(), errors.New(fmt.Sprintf("Time interval pace '%s' not supported", pace))
	}
}

func (tpl *ExpenseTemplate) GenerateRepeatingExpenses(dateRange DateRange) ([]*Expense, error) {
	if tpl.RepeatabilityIntervalUnit <= 0 {
		return nil, fmt.Errorf("repeatability interval unit must be greater than 0, got %d", tpl.RepeatabilityIntervalUnit)
	}

	expenses := []*Expense{}
	t := carbon.NewCarbon(tpl.InitialToBePaidOn)
	next, err := tpl.getNextToBePaidAt(t)
	if err != nil {
		return nil, err
	}

	// first is InitialToBePaidOn
	expenses = append(expenses, NewExpense(
		tpl.Amount,
		t.StdTime(),
		WithDescription(tpl.Description),
		WithTemplateID(tpl.ID),
	))

	for next.Lte(carbon.NewCarbon(dateRange.To)) {
		tCarbon, err := tpl.getNextToBePaidAt(t)
		if err != nil {
			return nil, err
		}
		t = tCarbon

		nextCarbon, err := tpl.getNextToBePaidAt(t)
		if err != nil {
			return nil, err
		}
		next = nextCarbon

		expenses = append(expenses, NewExpense(
			tpl.Amount,
			t.StdTime(),
			WithDescription(tpl.Description),
			WithTemplateID(tpl.ID),
		))
	}

	return expenses, nil
}
