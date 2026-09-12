package api

import (
	"context"
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
	ctx := c.Request.Context()

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

	exists, err := h.users.EmailExists(ctx, in.Email)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50006, "Lỗi máy chủ")
		return
	}
	if exists {
		fail(c, http.StatusConflict, 40900, "Email đã được đăng ký")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50000, "Không thể tạo tài khoản")
		return
	}

	u := User{ID: uuid.NewString(), Name: in.Name, Email: in.Email}

	// Create the user and seed their starter categories atomically: if
	// seeding fails partway through, the whole signup rolls back instead of
	// leaving a user account with no (or half of the) categories.
	err = h.withTx(ctx, func(tx *sql.Tx) error {
		if err := NewUserRepo(tx).Create(ctx, u, string(hash)); err != nil {
			return err
		}
		return NewCategoryRepo(tx).SeedDefaults(ctx, u.ID)
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50001, "Không thể tạo tài khoản")
		return
	}

	h.respondWithToken(c, u)
}

func (h *Handler) Login(c *gin.Context) {
	ctx := c.Request.Context()

	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, 40000, "Dữ liệu không hợp lệ")
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	u, hash, err := h.users.FindByEmail(ctx, in.Email)
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
	u, err := h.users.FindByID(c.Request.Context(), userIDFrom(c))
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
// The whole account (user + categories + sample data) is created in a single
// transaction: if any step fails, nothing is left behind for the client to
// retry against.
func (h *Handler) Demo(c *gin.Context) {
	ctx := c.Request.Context()

	id := uuid.NewString()
	email := "demo-" + id[:8] + "@fina.vn"
	name := "Demo"
	hash, err := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, 50004, "Không thể tạo tài khoản demo")
		return
	}
	u := User{ID: id, Name: name, Email: email}

	err = h.withTx(ctx, func(tx *sql.Tx) error {
		if err := NewUserRepo(tx).Create(ctx, u, string(hash)); err != nil {
			return err
		}
		catRepo := NewCategoryRepo(tx)
		if err := catRepo.SeedDefaults(ctx, id); err != nil {
			return err
		}
		return seedSampleData(ctx, catRepo, NewTransactionRepo(tx), NewBudgetRepo(tx), id)
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50005, "Không thể tạo tài khoản demo")
		return
	}

	h.respondWithToken(c, u)
}

// sampleTransaction/sampleBudget describe the seed rows a demo account gets;
// pulled to package scope (rather than local literals) so tests can derive
// expected counts from len(sampleTransactions)/len(sampleBudgets) instead of
// hand-copied numbers that would silently drift from the real dataset.
type sampleTransaction struct {
	typ    string
	amount int64
	cat    string
	d      int
}

var sampleTransactions = []sampleTransaction{
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

type sampleBudget struct {
	cat   string
	limit int64
}

var sampleBudgets = []sampleBudget{
	{"Ăn uống", 3000000},
	{"Di chuyển", 1000000},
	{"Mua sắm", 2000000},
	{"Hóa đơn", 3500000},
	{"Giải trí", 800000},
}

// seedSampleData populates a demo account with realistic transactions +
// budgets so the dashboard/charts show data immediately. Notes are left
// empty so rows display the (translatable) category name in either language.
func seedSampleData(ctx context.Context, catRepo *CategoryRepo, txRepo *TransactionRepo, budgetRepo *BudgetRepo, userID string) error {
	byName, err := catRepo.ByName(ctx, userID)
	if err != nil {
		return err
	}

	day := func(n int) string { return time.Now().AddDate(0, 0, -n).Format(dateLayout) }
	for _, s := range sampleTransactions {
		cid := byName[s.cat]
		if cid == "" {
			continue
		}
		t := Transaction{ID: uuid.NewString(), Type: s.typ, Amount: s.amount, CategoryID: cid, Note: "", Date: day(s.d)}
		if err := txRepo.Create(ctx, userID, t); err != nil {
			return err
		}
	}

	month := time.Now().Format("2006-01")
	for _, b := range sampleBudgets {
		cid := byName[b.cat]
		if cid == "" {
			continue
		}
		if _, err := budgetRepo.Upsert(ctx, userID, Budget{CategoryID: cid, Limit: b.limit, Month: month}); err != nil {
			return err
		}
	}
	return nil
}
