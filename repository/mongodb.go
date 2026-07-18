package repository

import (
	"context"
	"time"

	"desrosiers.org/budget/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBRepository struct {
	client    *mongo.Client
	db        *mongo.Database
	templates *mongo.Collection
	expenses  *mongo.Collection
	payments  *mongo.Collection
}

func NewMongoDBRepository(ctx context.Context, uri, dbName string) (*MongoDBRepository, error) {
	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	return &MongoDBRepository{
		client:    client,
		db:        db,
		templates: db.Collection("expense_templates"),
		expenses:  db.Collection("expenses"),
		payments:  db.Collection("payments"),
	}, nil
}

func (r *MongoDBRepository) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}

// SaveTemplate inserts or updates an expense template
func (r *MongoDBRepository) SaveTemplate(ctx context.Context, tpl *model.ExpenseTemplate) error {
	if tpl.ID.IsZero() {
		tpl.ID = primitive.NewObjectID()
		_, err := r.templates.InsertOne(ctx, tpl)
		return err
	}
	filter := bson.M{"_id": tpl.ID}
	update := bson.M{"$set": tpl}
	_, err := r.templates.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// GetExpenseTemplates retrieves all active templates
func (r *MongoDBRepository) GetExpenseTemplates(ctx context.Context) ([]*model.ExpenseTemplate, error) {
	filter := bson.M{"is_on_hold": false}
	cursor, err := r.templates.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []*model.ExpenseTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

// GetAllTemplates retrieves every template, including those on hold. Used by the
// template management page (GetExpenseTemplates only returns active ones).
func (r *MongoDBRepository) GetAllTemplates(ctx context.Context) ([]*model.ExpenseTemplate, error) {
	cursor, err := r.templates.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []*model.ExpenseTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

// GetTemplateByID retrieves a single template by its ID.
func (r *MongoDBRepository) GetTemplateByID(ctx context.Context, id primitive.ObjectID) (*model.ExpenseTemplate, error) {
	var tpl model.ExpenseTemplate
	if err := r.templates.FindOne(ctx, bson.M{"_id": id}).Decode(&tpl); err != nil {
		return nil, err
	}
	return &tpl, nil
}

// DeleteTemplate removes a template by its ID.
func (r *MongoDBRepository) DeleteTemplate(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.templates.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// GetExpensesWithPayments fetches expenses matching the filter and attaches their payments
func (r *MongoDBRepository) GetExpensesWithPayments(ctx context.Context, filter bson.M) ([]*model.Expense, error) {
	cursor, err := r.expenses.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var expenses []*model.Expense
	if err := cursor.All(ctx, &expenses); err != nil {
		return nil, err
	}

	if len(expenses) == 0 {
		return expenses, nil
	}

	// Collect expense IDs
	var expIDs []primitive.ObjectID
	expMap := make(map[primitive.ObjectID]*model.Expense)
	for _, exp := range expenses {
		expIDs = append(expIDs, exp.ID)
		expMap[exp.ID] = exp
		exp.Payments = []model.Payment{} // initialize
	}

	// Fetch all payments for these expenses
	payCursor, err := r.payments.Find(ctx, bson.M{"expense_id": bson.M{"$in": expIDs}})
	if err != nil {
		return nil, err
	}
	defer payCursor.Close(ctx)

	var payments []model.Payment
	if err := payCursor.All(ctx, &payments); err != nil {
		return nil, err
	}

	// Attach payments to their expenses
	for _, pay := range payments {
		if exp, ok := expMap[pay.ExpenseID]; ok {
			exp.Payments = append(exp.Payments, pay)
		}
	}

	return expenses, nil
}

// GetOrInsertExpense retrieves an expense for a template and date, or inserts it if it doesn't exist
func (r *MongoDBRepository) GetOrInsertExpense(ctx context.Context, exp *model.Expense) (*model.Expense, error) {
	if exp.TemplateID != nil {
		filter := bson.M{
			"template_id":   exp.TemplateID,
			"to_be_paid_at": exp.ToBePaidAt,
		}
		existing, err := r.GetExpensesWithPayments(ctx, filter)
		if err != nil {
			return nil, err
		}
		if len(existing) > 0 {
			return existing[0], nil
		}
	}

	if exp.ID.IsZero() {
		exp.ID = primitive.NewObjectID()
	}
	_, err := r.expenses.InsertOne(ctx, exp)
	if err != nil {
		return nil, err
	}
	exp.Payments = []model.Payment{}
	return exp, nil
}

// InsertOneTimeExpense inserts a new one-time expense
func (r *MongoDBRepository) InsertOneTimeExpense(ctx context.Context, exp *model.Expense) (*model.Expense, error) {
	if exp.ID.IsZero() {
		exp.ID = primitive.NewObjectID()
	}
	exp.TemplateID = nil
	_, err := r.expenses.InsertOne(ctx, exp)
	if err != nil {
		return nil, err
	}
	exp.Payments = []model.Payment{}
	return exp, nil
}

// UpdateExpense updates the editable fields of an existing expense.
func (r *MongoDBRepository) UpdateExpense(ctx context.Context, exp *model.Expense) error {
	filter := bson.M{"_id": exp.ID}
	update := bson.M{"$set": bson.M{
		"description":   exp.Description,
		"amount":        exp.Amount,
		"to_be_paid_at": exp.ToBePaidAt,
	}}
	_, err := r.expenses.UpdateOne(ctx, filter, update)
	return err
}

// CreatePayment creates a new payment for an expense
func (r *MongoDBRepository) CreatePayment(ctx context.Context, expenseID primitive.ObjectID, amount int, paidAt time.Time) (*model.Payment, error) {
	payment := model.Payment{
		ID:        primitive.NewObjectID(),
		ExpenseID: expenseID,
		Amount:    amount,
		PaidAt:    paidAt,
	}
	_, err := r.payments.InsertOne(ctx, payment)
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetExpensesForDashboard returns all relevant expenses for the dashboard dateRange,
// including past unpaid (overdue) expenses.
func (r *MongoDBRepository) GetExpensesForDashboard(ctx context.Context, dateRange model.DateRange) ([]*model.Expense, error) {
	// Fetch all expenses up to dateRange.To
	filter := bson.M{
		"to_be_paid_at": bson.M{"$lte": dateRange.To},
	}
	allExps, err := r.GetExpensesWithPayments(ctx, filter)
	if err != nil {
		return nil, err
	}

	var result []*model.Expense
	for _, exp := range allExps {
		// If the expense is on or before dateRange.From, only include it if it's unpaid
		if exp.ToBePaidAt.Before(dateRange.From) || exp.ToBePaidAt.Equal(dateRange.From) {
			if !exp.IsPaid() {
				result = append(result, exp)
			}
		} else {
			result = append(result, exp)
		}
	}

	return result, nil
}
