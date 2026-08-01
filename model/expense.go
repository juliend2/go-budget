package model

import (
	"fmt"
	"time"

	"github.com/dromara/carbon/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DateRange struct {
	From time.Time
	To   time.Time
}

type Expense struct {
	now         time.Time           // DI for testing
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Description string              `bson:"description" json:"description"`
	Amount      int                 `bson:"amount" json:"amount"` // dollars
	ToBePaidAt  time.Time           `bson:"to_be_paid_at" json:"to_be_paid_at"`
	TemplateID  *primitive.ObjectID `bson:"template_id,omitempty" json:"template_id,omitempty"`
	Payments    []Payment           `bson:"-" json:"payments,omitempty"` // populated at runtime
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

func WithID(id primitive.ObjectID) ExpenseOption {
	return func(e *Expense) {
		e.ID = id
	}
}

func WithDescription(desc string) ExpenseOption {
	return func(e *Expense) {
		e.Description = desc
	}
}

func WithTemplateID(id primitive.ObjectID) ExpenseOption {
	return func(e *Expense) {
		e.TemplateID = &id
	}
}

func WithPayments(payments []Payment) ExpenseOption {
	return func(e *Expense) {
		e.Payments = payments
	}
}

func (e *Expense) IsDue() bool {
	return !e.now.Before(e.ToBePaidAt)
}

func (e *Expense) IsPaid() bool {
	sum := 0
	for _, p := range e.Payments {
		sum += p.Amount
	}
	return sum >= e.Amount
}

func (e *Expense) GetRemainingAmount() int {
	rem := e.Amount
	for _, p := range e.Payments {
		rem -= p.Amount
	}
	return rem
}

func GetPayDays(dateRange DateRange) []time.Time {
	dates := []time.Time{}
	firstValue := carbon.NewCarbon(dateRange.From).StartOfDay()
	d := dateRange.From.Day()
	if d > 15 {
		firstValue = firstValue.EndOfMonth().StartOfDay()
	} else if d < 15 {
		firstValue = firstValue.SetDay(15)
	}
	// d is 15; do nothing

	dates = append(dates, firstValue.StdTime())

	for carbon.NewCarbon(dates[len(dates)-1]).Lt(carbon.NewCarbon(dateRange.To)) {
		value := carbon.NewCarbon(dates[len(dates)-1])
		if value.Day() == 15 {
			value = value.EndOfMonth().StartOfDay()
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
	if len(pays) == 0 {
		return acc
	}

	firstPay := carbon.NewCarbon(pays[0]).StartOfDay()

	for _, exp := range exps {
		toBePaidAt := carbon.NewCarbon(exp.ToBePaidAt).StartOfDay()

		// Overdue unpaid check: put unpaid expenses due strictly before the
		// first visible pay day into the first pay slot.
		if toBePaidAt.Lt(firstPay) {
			if !exp.IsPaid() {
				key := firstPay.ToDateString()
				acc[key] = append(acc[key], exp)
			}
			continue
		}

		// Group other expenses in their corresponding slots [from, to):
		// an expense due on a pay day belongs to the period that pay day starts.
		for i := 0; i < len(pays)-1; i++ {
			from := carbon.NewCarbon(pays[i]).StartOfDay()
			to := carbon.NewCarbon(pays[i+1]).StartOfDay()

			if toBePaidAt.Gte(from) && toBePaidAt.Lt(to) {
				fmt.Println("toBePaidAt: " + toBePaidAt.ToDateString() + ", from: " + from.ToDateString() + ", to: " + to.ToDateString())
				key := from.ToDateString()
				acc[key] = append(acc[key], exp)
				break
			}
		}
	}

	return acc
}
