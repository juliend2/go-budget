package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"desrosiers.org/budget/controller"
	"desrosiers.org/budget/model"
	"desrosiers.org/budget/repository"
	"github.com/coreos/go-oidc/v3/oidc"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/oauth2"
)

var (
	clientID          = os.Getenv("GOOGLE_OAUTH2_CLIENT_ID")
	clientSecret      = os.Getenv("GOOGLE_OAUTH2_CLIENT_SECRET")
	clientRedirectUrl = os.Getenv("OAUTH2_REDIRECT_URL")
)

//go:embed views/*
var viewsFS embed.FS

func main() {
	ctx := context.Background()

	// Connect to MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://root:password@localhost:27017/admin"
	}

	// OIDC / OAuth2:
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		log.Fatal(err)
	}
	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  clientRedirectUrl,
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	// Authorization: only these Google accounts may log in. Refuse to start
	// without an allowlist, otherwise the app would be open to anyone.
	allowedEmails := controller.ParseAllowedEmails(os.Getenv("ALLOWED_EMAILS"))
	if len(allowedEmails) == 0 {
		log.Fatal("ALLOWED_EMAILS is unset or empty; set it to a comma-separated list of authorized Google account emails")
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
	tmplLogin, err := template.ParseFS(viewsFS, "views/login.html")
	if err != nil {
		log.Fatalf("Failed to parse login HTML template: %v", err)
	}
	tmplEdit, err := template.ParseFS(viewsFS, "views/edit.html")
	if err != nil {
		log.Fatalf("Failed to parse edit HTML template: %v", err)
	}
	tmplTemplates, err := template.ParseFS(viewsFS, "views/templates.html")
	if err != nil {
		log.Fatalf("Failed to parse templates HTML template: %v", err)
	}
	tmplTemplateEdit, err := template.ParseFS(viewsFS, "views/template_edit.html")
	if err != nil {
		log.Fatalf("Failed to parse template edit HTML template: %v", err)
	}

	sessions := controller.NewSessionStore()

	// Router setup
	//
	// Auth endpoints and static assets are public; everything else requires a
	// valid session (RequireAuth redirects to /login otherwise).
	http.HandleFunc("/login", controller.HandleLoginPage(tmplLogin))
	http.HandleFunc("/auth/google/login", controller.HandleLogin(config))
	http.HandleFunc("/logout", controller.HandleLogout(sessions))
	http.HandleFunc("/auth/google/callback", controller.HandleCallback(config, verifier, sessions, allowedEmails))
	http.HandleFunc("/static/css/styles.css", controller.HandleCSS(viewsFS))

	http.HandleFunc("/", controller.RequireAuth(sessions, controller.HandleDashboard(repo, tmplDashboard)))
	http.HandleFunc("/expense/pay", controller.RequireAuth(sessions, controller.HandleExpensePay(repo, tmplPay)))
	http.HandleFunc("/expense/edit", controller.RequireAuth(sessions, controller.HandleExpenseEdit(repo, tmplEdit)))
	http.HandleFunc("/expense/add", controller.RequireAuth(sessions, controller.HandleAddExpense(repo)))
	http.HandleFunc("/template/add", controller.RequireAuth(sessions, controller.HandleAddTemplate(repo)))
	http.HandleFunc("/templates", controller.RequireAuth(sessions, controller.HandleTemplatesList(repo, tmplTemplates)))
	http.HandleFunc("/template/edit", controller.RequireAuth(sessions, controller.HandleTemplateEdit(repo, tmplTemplateEdit)))
	http.HandleFunc("/template/delete", controller.RequireAuth(sessions, controller.HandleDeleteTemplate(repo)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
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
