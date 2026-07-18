package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"desrosiers.org/budget/model"
	"desrosiers.org/budget/repository"
	"github.com/dromara/carbon/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

//go:embed views/*
var viewsFS embed.FS

func main() {
	// Connect to MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://root:password@localhost:27017/admin"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo, err := repository.NewMongoDBRepository(ctx, mongoURI, "budget")
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer repo.Close(ctx)

	log.Println("Successfully connected to MongoDB")

	// Seed initial data if database is empty
	err = seedData(ctx, repo)
	if err != nil {
		log.Fatalf("Failed to seed initial data: %v", err)
	}

	// Parse templates
	tmplDashboard, err := template.ParseFS(viewsFS, "views/index.html")
	if err != nil {
		log.Fatalf("Failed to parse dashboard HTML template: %v", err)
	}
	tmplPay, err := template.ParseFS(viewsFS, "views/pay.html")
	if err != nil {
		log.Fatalf("Failed to parse pay HTML template: %v", err)
	}

	// Router setup
	http.HandleFunc("/", handleDashboard(repo, tmplDashboard))
	http.HandleFunc("/expense/pay", handleExpensePay(repo, tmplPay))
	http.HandleFunc("/expense/add", handleAddExpense(repo))
	http.HandleFunc("/template/add", handleAddTemplate(repo))
	http.HandleFunc("/static/css/styles.css", handleCSS)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func handleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	cssData, err := viewsFS.ReadFile("views/css/styles.css")
	if err != nil {
		http.Error(w, "CSS not found", http.StatusNotFound)
		return
	}
	w.Write(cssData)
}

type PeriodView struct {
	PeriodStartStr string
	PayDayStr      string
	Expenses       []*model.Expense
	TotalAmount    int
}

func handleDashboard(repo *repository.MongoDBRepository, tmpl *template.Template) http.HandlerFunc {
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
			CurrentDate string
			Periods     []PeriodView
		}{
			CurrentDate: time.Now().Format(time.DateOnly),
			Periods:     periods,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

func handleExpensePay(repo *repository.MongoDBRepository, tmpl *template.Template) http.HandlerFunc {
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

func handleAddExpense(repo *repository.MongoDBRepository) http.HandlerFunc {
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

func handleAddTemplate(repo *repository.MongoDBRepository) http.HandlerFunc {
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

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func seedData(ctx context.Context, repo *repository.MongoDBRepository) error {
	// Check if templates collection is empty
	templates, err := repo.GetExpenseTemplates(ctx)
	if err != nil {
		return err
	}

	if len(templates) > 0 {
		return nil // already seeded
	}

	log.Println("Seeding initial recurring templates...")

	// Seed templates matching the user's checklist
	seedTemplates := []*model.ExpenseTemplate{
		model.NewExpenseTemplate(4, "(Apple) iCloud+",
			model.WithInitialToBePaidOn(2026, time.July, 11),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(12, "Crave ( prime channels)",
			model.WithInitialToBePaidOn(2026, time.July, 16),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(140, "Provision pour électricité",
			model.WithInitialToBePaidOn(2026, time.July, 17),
			model.WithRepeatabilityInterval(2, "W"), // every 2 weeks
		),
		model.NewExpenseTemplate(53, "Fizz",
			model.WithInitialToBePaidOn(2026, time.July, 19),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(11, "(Apple) Apple Music",
			model.WithInitialToBePaidOn(2026, time.July, 20),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(250, "Mensualité Orthodontiste d'Alice (CC)",
			model.WithInitialToBePaidOn(2026, time.July, 22),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(1026, "Provision pour Loyer",
			model.WithInitialToBePaidOn(2026, time.July, 22),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(500, "Épicerie",
			model.WithInitialToBePaidOn(2026, time.July, 24),
			model.WithRepeatabilityInterval(2, "W"), // every 2 weeks
		),
		model.NewExpenseTemplate(59, "Fido",
			model.WithInitialToBePaidOn(2026, time.July, 26),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(100, "Visa Affaires SM - Marge de crédit",
			model.WithInitialToBePaidOn(2026, time.July, 27),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(35, "SAAQ",
			model.WithInitialToBePaidOn(2026, time.July, 27),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(100, "Passe STM du mois",
			model.WithInitialToBePaidOn(2026, time.August, 1),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(146, "Assurances — Desjardins & Manuvie (dans compte de Julien)",
			model.WithInitialToBePaidOn(2026, time.August, 2),
			model.WithRepeatabilityInterval(1, "M"),
		),
		model.NewExpenseTemplate(120, "Intérêts sur prêt — Marge",
			model.WithInitialToBePaidOn(2026, time.August, 3),
			model.WithRepeatabilityInterval(1, "M"),
		),
	}

	for _, tpl := range seedTemplates {
		tpl.ID = primitive.NewObjectID()
		if err := repo.SaveTemplate(ctx, tpl); err != nil {
			return err
		}
	}

	log.Println("Seeding initial one-time expenses...")

	// Seed one-time expenses
	seedExpenses := []*model.Expense{
		model.NewExpense(348, model.Date(2026, time.July, 10), model.WithDescription("Épicerie")),
		model.NewExpense(0, model.Date(2026, time.July, 14), model.WithDescription("Café")),
	}

	for _, exp := range seedExpenses {
		_, err := repo.InsertOneTimeExpense(ctx, exp)
		if err != nil {
			return err
		}
	}

	log.Println("Seeding completed successfully")
	return nil
}
