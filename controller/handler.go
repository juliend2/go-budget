package controller

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"desrosiers.org/budget/model"
	"desrosiers.org/budget/repository"
	"github.com/dromara/carbon/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PeriodView struct {
	PeriodStartStr string
	PayDayStr      string
	Expenses       []*model.Expense
	TotalAmount    int
}

func HandleDashboard(repo *repository.MongoDBRepository, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		dateRange := model.DateRange{
			From: model.GetPreviousPayDate(time.Now()),
			To:   model.GetNextPayDate(carbon.NewCarbon(time.Now()).AddMonth().StdTime()),
		}

		// 1. Generate/Ensure repeating expenses for active templates in the range
		templates, err := repo.GetExpenseTemplates(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch templates: %v", err), http.StatusInternalServerError)
			return
		}

		for _, tpl := range templates {
			repeating, err := tpl.GenerateRepeatingExpenses(dateRange)
			if err != nil {
				log.Printf("Error generating expenses for template %s: %v", tpl.Description, err)
				continue
			}

			for _, exp := range repeating {
				_, err := repo.GetOrInsertExpense(ctx, exp)
				if err != nil {
					log.Printf("Error ensuring repeating expense: %v", err)
				}
			}
		}

		// 2. Fetch all expenses in dashboard range (plus overdue unpaid ones)
		expenses, err := repo.GetExpensesForDashboard(ctx, dateRange)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch dashboard expenses: %v", err), http.StatusInternalServerError)
			return
		}

		// 3. Get paydays list
		payDays := model.GetPayDays(dateRange)

		// 4. Group expenses
		grouped := model.PutExpensesInTheirPayPeriods(payDays, expenses)

		// 5. Structure period views chronologically
		var periods []PeriodView
		for i := 0; i < len(payDays)-1; i++ {
			start := payDays[i]
			end := payDays[i+1]

			exps := grouped[start.Format("2006-01-02")]

			// Sort chronologically in each period
			sort.Slice(exps, func(i, j int) bool {
				return exps[i].ToBePaidAt.Before(exps[j].ToBePaidAt)
			})

			total := 0
			for _, e := range exps {
				total += e.GetRemainingAmount()
			}

			periods = append(periods, PeriodView{
				PeriodStartStr: start.Format("01/02/2006"),
				PayDayStr:      end.Format("01/02/2006"),
				Expenses:       exps,
				TotalAmount:    total,
			})
		}

		data := struct {
			CurrentDate time.Time
			Periods     []PeriodView
		}{
			CurrentDate: time.Now(),
			Periods:     periods,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

func HandleExpensePay(repo *repository.MongoDBRepository, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method == http.MethodGet {
			// Serve the payment page
			idHex := r.URL.Query().Get("id")
			id, err := primitive.ObjectIDFromHex(idHex)
			if err != nil {
				http.Error(w, "Invalid expense ID", http.StatusBadRequest)
				return
			}

			expenses, err := repo.GetExpensesWithPayments(ctx, bson.M{"_id": id})
			if err != nil || len(expenses) == 0 {
				http.Error(w, "Expense not found", http.StatusNotFound)
				return
			}

			exp := expenses[0]

			// Calculate paid and remaining
			paidAmount := 0
			for _, p := range exp.Payments {
				paidAmount += p.Amount
			}
			remainingAmount := exp.Amount - paidAmount

			data := struct {
				Expense         *model.Expense
				PaidAmount      int
				RemainingAmount int
			}{
				Expense:         exp,
				PaidAmount:      paidAmount,
				RemainingAmount: remainingAmount,
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.Execute(w, data); err != nil {
				log.Printf("Error rendering pay template: %v", err)
			}
			return
		}

		if r.Method == http.MethodPost {
			// Record the payment
			expenseIDHex := r.FormValue("expense_id")
			expenseID, err := primitive.ObjectIDFromHex(expenseIDHex)
			if err != nil {
				http.Error(w, "Invalid expense ID", http.StatusBadRequest)
				return
			}

			amountStr := r.FormValue("amount")
			amount, err := strconv.Atoi(amountStr)
			if err != nil || amount <= 0 {
				http.Error(w, "Invalid payment amount", http.StatusBadRequest)
				return
			}

			// Verify the amount doesn't exceed remaining
			expenses, err := repo.GetExpensesWithPayments(ctx, bson.M{"_id": expenseID})
			if err != nil || len(expenses) == 0 {
				http.Error(w, "Expense not found", http.StatusNotFound)
				return
			}
			exp := expenses[0]

			paidAmount := 0
			for _, p := range exp.Payments {
				paidAmount += p.Amount
			}
			remainingAmount := exp.Amount - paidAmount

			if amount > remainingAmount {
				http.Error(w, fmt.Sprintf("Payment amount %d exceeds remaining amount %d", amount, remainingAmount), http.StatusBadRequest)
				return
			}

			_, err = repo.CreatePayment(ctx, expenseID, amount, time.Now())
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to record payment: %v", err), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleExpenseEdit(repo *repository.MongoDBRepository, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method == http.MethodGet {
			// Serve the edit form pre-filled with the expense's current values.
			idHex := r.URL.Query().Get("id")
			id, err := primitive.ObjectIDFromHex(idHex)
			if err != nil {
				http.Error(w, "Invalid expense ID", http.StatusBadRequest)
				return
			}

			expenses, err := repo.GetExpensesWithPayments(ctx, bson.M{"_id": id})
			if err != nil || len(expenses) == 0 {
				http.Error(w, "Expense not found", http.StatusNotFound)
				return
			}

			data := struct {
				Expense *model.Expense
			}{
				Expense: expenses[0],
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.Execute(w, data); err != nil {
				log.Printf("Error rendering edit template: %v", err)
			}
			return
		}

		if r.Method == http.MethodPost {
			idHex := r.FormValue("expense_id")
			id, err := primitive.ObjectIDFromHex(idHex)
			if err != nil {
				http.Error(w, "Invalid expense ID", http.StatusBadRequest)
				return
			}

			desc := r.FormValue("description")
			amount, err := strconv.Atoi(r.FormValue("amount"))
			if err != nil || amount < 0 {
				http.Error(w, "Invalid amount", http.StatusBadRequest)
				return
			}

			toBePaidAt, err := time.Parse("2006-01-02", r.FormValue("to_be_paid_at"))
			if err != nil {
				http.Error(w, "Invalid date format", http.StatusBadRequest)
				return
			}

			exp := &model.Expense{
				ID:          id,
				Description: desc,
				Amount:      amount,
				ToBePaidAt:  toBePaidAt,
			}
			if err := repo.UpdateExpense(ctx, exp); err != nil {
				http.Error(w, fmt.Sprintf("Failed to update expense: %v", err), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleAddExpense(repo *repository.MongoDBRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		desc := r.FormValue("description")
		amountStr := r.FormValue("amount")
		toBePaidAtStr := r.FormValue("to_be_paid_at")

		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount < 0 {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		toBePaidAt, err := time.Parse("2006-01-02", toBePaidAtStr)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		exp := model.NewExpense(amount, toBePaidAt, model.WithDescription(desc))
		_, err = repo.InsertOneTimeExpense(ctx, exp)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save expense: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func HandleAddTemplate(repo *repository.MongoDBRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		desc := r.FormValue("description")
		amountStr := r.FormValue("amount")
		initialStr := r.FormValue("initial_to_be_paid_on")
		unitStr := r.FormValue("interval_unit")
		pace := r.FormValue("interval_pace")

		amount, err := strconv.Atoi(amountStr)
		if err != nil || amount < 0 {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		initialDate, err := time.Parse("2006-01-02", initialStr)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}

		unit, err := strconv.Atoi(unitStr)
		if err != nil || unit <= 0 {
			http.Error(w, "Invalid unit", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		tpl := model.NewExpenseTemplate(amount, desc,
			model.WithInitialToBePaidOn(initialDate.Year(), initialDate.Month(), initialDate.Day()),
			model.WithRepeatabilityInterval(unit, pace),
		)

		err = repo.SaveTemplate(ctx, tpl)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save template: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/templates", http.StatusSeeOther)
	}
}

func HandleTemplatesList(repo *repository.MongoDBRepository, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		templates, err := repo.GetAllTemplates(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch templates: %v", err), http.StatusInternalServerError)
			return
		}

		sort.Slice(templates, func(i, j int) bool {
			return templates[i].Description < templates[j].Description
		})

		data := struct {
			Templates []*model.ExpenseTemplate
		}{
			Templates: templates,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error rendering templates list: %v", err)
		}
	}
}

func HandleTemplateEdit(repo *repository.MongoDBRepository, tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method == http.MethodGet {
			id, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
			if err != nil {
				http.Error(w, "Invalid template ID", http.StatusBadRequest)
				return
			}

			tpl, err := repo.GetTemplateByID(ctx, id)
			if err != nil {
				http.Error(w, "Template not found", http.StatusNotFound)
				return
			}

			data := struct {
				Template *model.ExpenseTemplate
			}{
				Template: tpl,
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.Execute(w, data); err != nil {
				log.Printf("Error rendering template edit: %v", err)
			}
			return
		}

		if r.Method == http.MethodPost {
			id, err := primitive.ObjectIDFromHex(r.FormValue("template_id"))
			if err != nil {
				http.Error(w, "Invalid template ID", http.StatusBadRequest)
				return
			}

			desc := r.FormValue("description")
			amount, err := strconv.Atoi(r.FormValue("amount"))
			if err != nil || amount < 0 {
				http.Error(w, "Invalid amount", http.StatusBadRequest)
				return
			}

			initialDate, err := time.Parse("2006-01-02", r.FormValue("initial_to_be_paid_on"))
			if err != nil {
				http.Error(w, "Invalid date format", http.StatusBadRequest)
				return
			}

			unit, err := strconv.Atoi(r.FormValue("interval_unit"))
			if err != nil || unit <= 0 {
				http.Error(w, "Invalid interval unit", http.StatusBadRequest)
				return
			}

			tpl := &model.ExpenseTemplate{
				ID:                        id,
				Amount:                    amount,
				Description:               desc,
				InitialToBePaidOn:         initialDate,
				RepeatabilityIntervalUnit: unit,
				RepeatabilityIntervalPace: r.FormValue("interval_pace"),
				IsOnHold:                  r.FormValue("is_on_hold") == "on",
			}
			if err := repo.SaveTemplate(ctx, tpl); err != nil {
				http.Error(w, fmt.Sprintf("Failed to update template: %v", err), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/templates", http.StatusSeeOther)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleDeleteTemplate(repo *repository.MongoDBRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id, err := primitive.ObjectIDFromHex(r.FormValue("template_id"))
		if err != nil {
			http.Error(w, "Invalid template ID", http.StatusBadRequest)
			return
		}

		if err := repo.DeleteTemplate(r.Context(), id); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete template: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/templates", http.StatusSeeOther)
	}
}

func HandleCSS(viewsFS embed.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		cssData, err := viewsFS.ReadFile("views/css/styles.css")
		if err != nil {
			http.Error(w, "CSS not found", http.StatusNotFound)
			return
		}
		w.Write(cssData)
	}
}
