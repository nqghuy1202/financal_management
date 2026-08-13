package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const ctxUserID = "userID"

// ---- token helpers ----

func (h *Handler) issueToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(h.secret)
}

func (h *Handler) parseToken(tok string) (string, error) {
	parsed, err := jwt.ParseWithClaims(tok, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.secret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}

// AuthMiddleware requires a valid "Authorization: Bearer <token>" header and
// stores the resolved user id in the context.
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" || token == header {
			fail(c, http.StatusUnauthorized, 40100, "Thiếu token xác thực")
			return
		}
		userID, err := h.parseToken(token)
		if err != nil {
			fail(c, http.StatusUnauthorized, 40101, "Token không hợp lệ hoặc đã hết hạn")
			return
		}
		c.Set(ctxUserID, userID)
		c.Next()
	}
}

func userIDFrom(c *gin.Context) string {
	v, _ := c.Get(ctxUserID)
	s, _ := v.(string)
	return s
}

// ---- handlers ----

type credentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) RegisterUser(c *gin.Context) {
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40000, "Dữ liệu không hợp lệ")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || in.Email == "" || len(in.Password) < 6 {
		fail(c, http.StatusBadRequest, 40001, "Vui lòng nhập tên, email và mật khẩu ≥ 6 ký tự")
		return
	}

	var exists int
	_ = h.db.QueryRow(`SELECT 1 FROM users WHERE email = ?`, in.Email).Scan(&exists)
	if exists == 1 {
		fail(c, http.StatusConflict, 40900, "Email đã được đăng ký")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50000, "Không thể tạo tài khoản")
		return
	}

	id := uuid.NewString()
	if _, err := h.db.Exec(
		`INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)`,
		id, in.Name, in.Email, string(hash),
	); err != nil {
		fail(c, http.StatusInternalServerError, 50001, "Không thể tạo tài khoản")
		return
	}

	if err := h.seedDefaultCategories(id); err != nil {
		// non-fatal: account is created; categories can be added manually
		_ = err
	}

	h.respondWithToken(c, User{ID: id, Name: in.Name, Email: in.Email})
}

func (h *Handler) Login(c *gin.Context) {
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40000, "Dữ liệu không hợp lệ")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	var u User
	var hash string
	err := h.db.QueryRow(
		`SELECT id, name, email, password_hash FROM users WHERE email = ?`, in.Email,
	).Scan(&u.ID, &u.Name, &u.Email, &hash)
	if errors.Is(err, sql.ErrNoRows) || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		fail(c, http.StatusUnauthorized, 40102, "Email hoặc mật khẩu không đúng")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, 50002, "Lỗi máy chủ")
		return
	}
	h.respondWithToken(c, u)
}

func (h *Handler) Me(c *gin.Context) {
	var u User
	err := h.db.QueryRow(
		`SELECT id, name, email FROM users WHERE id = ?`, userIDFrom(c),
	).Scan(&u.ID, &u.Name, &u.Email)
	if err != nil {
		fail(c, http.StatusUnauthorized, 40103, "Phiên đăng nhập không hợp lệ")
		return
	}
	ok(c, u)
}

func (h *Handler) respondWithToken(c *gin.Context, u User) {
	token, err := h.issueToken(u.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50003, "Không thể cấp token")
		return
	}
	ok(c, gin.H{"token": token, "user": u})
}

// Demo creates a fresh throwaway account pre-filled with sample data and logs
// straight in — one click for recruiters/visitors, no form, isolated per click.
func (h *Handler) Demo(c *gin.Context) {
	id := uuid.NewString()
	email := "demo-" + id[:8] + "@fina.vn"
	name := "Demo"
	hash, err := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50004, "Không thể tạo tài khoản demo")
		return
	}
	if _, err := h.db.Exec(
		`INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)`,
		id, name, email, string(hash),
	); err != nil {
		fail(c, http.StatusInternalServerError, 50005, "Không thể tạo tài khoản demo")
		return
	}
	_ = h.seedDefaultCategories(id)
	_ = h.seedSampleData(id)
	h.respondWithToken(c, User{ID: id, Name: name, Email: email})
}

// seedSampleData populates a demo account with realistic transactions + budgets
// so the dashboard/charts show data immediately. Notes are left empty so rows
// display the (translatable) category name in either language.
func (h *Handler) seedSampleData(userID string) error {
	rows, err := h.db.Query(`SELECT id, name FROM categories WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	byName := map[string]string{}
	for rows.Next() {
		var cid, name string
		if err := rows.Scan(&cid, &name); err == nil {
			byName[name] = cid
		}
	}

	day := func(n int) string { return time.Now().AddDate(0, 0, -n).Format(dateLayout) }
	type sample struct {
		typ    string
		amount int64
		cat    string
		d      int
	}
	txs := []sample{
		{"income", 18000000, "Lương", 28},
		{"income", 2500000, "Thưởng", 20},
		{"income", 1200000, "Đầu tư", 12},
		{"expense", 320000, "Ăn uống", 1},
		{"expense", 150000, "Ăn uống", 2},
		{"expense", 450000, "Di chuyển", 3},
		{"expense", 1200000, "Mua sắm", 5},
		{"expense", 850000, "Hóa đơn", 7},
		{"expense", 299000, "Giải trí", 8},
		{"expense", 500000, "Sức khỏe", 10},
		{"expense", 2200000, "Hóa đơn", 14},
		{"expense", 640000, "Mua sắm", 16},
		{"expense", 95000, "Di chuyển", 18},
	}
	for _, s := range txs {
		cid := byName[s.cat]
		if cid == "" {
			continue
		}
		_, _ = h.db.Exec(
			`INSERT INTO transactions (id, user_id, type, amount, category_id, note, date) VALUES (?, ?, ?, ?, ?, '', ?)`,
			uuid.NewString(), userID, s.typ, s.amount, cid, day(s.d),
		)
	}

	month := time.Now().Format("2006-01")
	budgets := []struct {
		cat   string
		limit int64
	}{
		{"Ăn uống", 3000000},
		{"Di chuyển", 1000000},
		{"Mua sắm", 2000000},
		{"Hóa đơn", 3500000},
		{"Giải trí", 800000},
	}
	for _, b := range budgets {
		cid := byName[b.cat]
		if cid == "" {
			continue
		}
		_, _ = h.db.Exec(
			`INSERT INTO budgets (id, user_id, category_id, limit_amount, month) VALUES (?, ?, ?, ?, ?)`,
			uuid.NewString(), userID, cid, b.limit, month,
		)
	}
	return nil
}

// seedDefaultCategories gives a new user the same starter categories the UI
// used to seed locally.
func (h *Handler) seedDefaultCategories(userID string) error {
	defaults := []Category{
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
	for _, cat := range defaults {
		if _, err := h.db.Exec(
			`INSERT INTO categories (id, user_id, name, type, color, icon) VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.NewString(), userID, cat.Name, cat.Type, cat.Color, cat.Icon,
		); err != nil {
			return err
		}
	}
	return nil
}
