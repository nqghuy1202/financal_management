package api

import (
	"context"

	"github.com/google/uuid"
)

// CategoryRepo is the data-access layer for categories.
type CategoryRepo struct{ db dbtx }

func NewCategoryRepo(db dbtx) *CategoryRepo { return &CategoryRepo{db: db} }

func (r *CategoryRepo) List(ctx context.Context, userID string) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, type, color, icon FROM categories WHERE user_id = ? ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Category, 0)
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Color, &cat.Icon); err != nil {
			return nil, err
		}
		list = append(list, cat)
	}
	return list, rows.Err()
}

func (r *CategoryRepo) Create(ctx context.Context, userID string, cat Category) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (id, user_id, name, type, color, icon) VALUES (?, ?, ?, ?, ?, ?)`,
		cat.ID, userID, cat.Name, cat.Type, cat.Color, cat.Icon,
	)
	return err
}

func (r *CategoryRepo) Delete(ctx context.Context, userID, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// Owns reports whether categoryID exists and belongs to userID — used to
// validate the categoryId a transaction/budget is filed against.
func (r *CategoryRepo) Owns(ctx context.Context, userID, categoryID string) (bool, error) {
	var one int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM categories WHERE id = ? AND user_id = ?`, categoryID, userID,
	).Scan(&one)
	if err != nil {
		return false, ignoreNoRows(err)
	}
	return one == 1, nil
}

// ByName returns the user's categories keyed by name (used by demo-account
// seeding to resolve sample transactions/budgets to real category ids).
func (r *CategoryRepo) ByName(ctx context.Context, userID string) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM categories WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byName := map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		byName[name] = id
	}
	return byName, rows.Err()
}

// defaultCategories are the starter categories every new account gets.
var defaultCategories = []Category{
	{Name: "Lương", Type: "income", Color: "#10b981", Icon: "Wallet"},
	{Name: "Thưởng", Type: "income", Color: "#22c55e", Icon: "Gift"},
	{Name: "Đầu tư", Type: "income", Color: "#0ea5e9", Icon: "TrendingUp"},
	{Name: "Ăn uống", Type: "expense", Color: "#f97316", Icon: "Utensils"},
	{Name: "Di chuyển", Type: "expense", Color: "#6366f1", Icon: "Car"},
	{Name: "Mua sắm", Type: "expense", Color: "#ec4899", Icon: "ShoppingBag"},
	{Name: "Hóa đơn", Type: "expense", Color: "#eab308", Icon: "Receipt"},
	{Name: "Sức khỏe", Type: "expense", Color: "#ef4444", Icon: "HeartPulse"},
	{Name: "Giải trí", Type: "expense", Color: "#8b5cf6", Icon: "Gamepad2"},
	{Name: "Khác", Type: "expense", Color: "#64748b", Icon: "MoreHorizontal"},
}

// SeedDefaults gives userID the standard starter categories. Call it inside
// withTx alongside user creation so a failure here rolls back the whole
// signup instead of leaving a user with no categories.
func (r *CategoryRepo) SeedDefaults(ctx context.Context, userID string) error {
	for _, cat := range defaultCategories {
		cat.ID = uuid.NewString()
		if err := r.Create(ctx, userID, cat); err != nil {
			return err
		}
	}
	return nil
}
