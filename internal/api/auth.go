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
	name := "Minh Nguyễn" // the PRD's persona — a name, not a label, reads better than "Demo" on a first look
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
		if err := NewCategoryRepo(tx).SeedDefaults(ctx, id); err != nil {
			return err
		}
		return seedSampleData(ctx, tx, id)
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, 50005, "Không thể tạo tài khoản demo")
		return
	}

	h.respondWithToken(c, u)
}

// ---- demo account narrative ----
//
// The one-click demo account is most people's first look at the app, so its
// seed data is written as one believable story — a mid-career professional
// on a steadily rising salary — rather than a handful of disconnected rows:
//   - incomes/fixedCosts/settings are populated so the Safe-to-spend hero
//     (Epic 1) shows a real number on first load instead of the "declare
//     your income" prompt.
//   - Giải trí and Mua sắm are budgeted tight enough that today's spending
//     alone crosses their thresholds, so Epic 2's alert banner and 3-tier
//     budget pill are visible immediately, every time — not just when the
//     demo happens to be clicked late in the month.

// demoCycleStartDay/demoSavingsGoal are the settings every demo account
// gets. The cycle start day is kept at 1 (a plain calendar month) so every
// "current month" concept the frontend computes — the cycle window and the
// wall-clock's own calendar month (AD-4) — agrees, which keeps a demo
// account's numbers internally consistent.
const (
	demoCycleStartDay        = 1
	demoSavingsGoal    int64 = 3_000_000
)

// demoFixedCosts are the recurring committed costs every demo account is
// seeded with.
var demoFixedCosts = []FixedCost{
	{Name: "Thuê nhà", Amount: 5_000_000},
	{Name: "Điện & nước", Amount: 500_000},
	{Name: "Internet & điện thoại", Amount: 350_000},
	{Name: "Bảo hiểm sức khỏe", Amount: 600_000},
	{Name: "Trả góp xe máy", Amount: 800_000},
}

// demoSalaries is the declared "Lương" for this cycle and the 5 before it,
// oldest first — a steady raise trend, so the CycleUpdateSheet's
// previous-cycle income comparison and the cashflow chart both show
// believable growth instead of a flat line.
var demoSalaries = []int64{16_000_000, 16_500_000, 17_000_000, 17_500_000, 18_000_000, 18_500_000}

// sampleTransaction/sampleBudget describe the seed rows a demo account gets;
// pulled to package scope (rather than local literals) so tests can derive
// expected counts from len(sampleTransactions)/len(sampleBudgets) instead of
// hand-copied numbers that would silently drift from the real dataset.
type sampleTransaction struct {
	typ    string
	amount int64
	cat    string
	d      int // days ago from "today"
}

// sampleTransactions is built once at package init from static day offsets
// (never from time.Now() — a long-running server must compute real dates
// per request, in seedSampleData, not once at process start).
var sampleTransactions = buildSampleTransactions()

func buildSampleTransactions() []sampleTransaction {
	var out []sampleTransaction

	// One salary credit per of the last 6 months (oldest to newest, ~30
	// days apart), rising per demoSalaries. This month's is dated today
	// (d:0) — the only offset guaranteed to land in the current cycle no
	// matter what day of the month the demo account is created on.
	for i, amt := range demoSalaries {
		monthsAgo := len(demoSalaries) - 1 - i
		d := monthsAgo * 30
		if d > 0 {
			d -= 3 // salary lands a few days into its month, not right on the boundary
		}
		out = append(out, sampleTransaction{"income", amt, "Lương", d})
	}
	// A couple of one-off credits for texture on the cashflow chart.
	out = append(out,
		sampleTransaction{"income", 2_500_000, "Thưởng", 100},
		sampleTransaction{"income", 1_200_000, "Đầu tư", 45},
	)

	// A recurring monthly spending pattern, replayed for each of the 5 fully
	// elapsed months with a small per-month wiggle so the trend line has
	// real shape instead of repeating identically. Each item's offset is
	// anchored on that month's salary offset (monthsAgo*30 - 3, ±13 days for
	// where in the month it falls) rather than a literal day-of-month, so
	// the whole cluster lands inside the same calendar-month bucket as that
	// month's "Lương" row regardless of what day of the month "today" is.
	monthlyPattern := []struct {
		cat string
		amt int64
		day int // spread within the month, 2-25 — not a literal day-of-month
	}{
		{"Ăn uống", 320_000, 2}, {"Ăn uống", 280_000, 8}, {"Ăn uống", 350_000, 14},
		{"Ăn uống", 300_000, 19}, {"Ăn uống", 260_000, 25},
		{"Di chuyển", 450_000, 5}, {"Di chuyển", 200_000, 21},
		{"Mua sắm", 900_000, 10}, {"Mua sắm", 500_000, 24},
		{"Hóa đơn", 1_800_000, 6},
		{"Giải trí", 250_000, 16},
		{"Sức khỏe", 300_000, 12},
	}
	for monthsAgo := 5; monthsAgo >= 1; monthsAgo-- {
		anchor := monthsAgo*30 - 3
		wiggle := int64(monthsAgo%3) * 20_000
		for _, p := range monthlyPattern {
			out = append(out, sampleTransaction{"expense", p.amt + wiggle, p.cat, anchor - (p.day - 15)})
		}
	}

	// This month so far: a lighter version of the same everyday spending,
	// dated within the first couple of days so it still lands in the
	// current cycle even if the demo account happens to be created early in
	// the month.
	out = append(out,
		sampleTransaction{"expense", 150_000, "Ăn uống", 1},
		sampleTransaction{"expense", 130_000, "Ăn uống", 2},
		sampleTransaction{"expense", 90_000, "Di chuyển", 1},
		sampleTransaction{"expense", 850_000, "Hóa đơn", 2},
	)
	// ...and, dated today (d:0) so this part is *always* true: two
	// entertainment purchases that push Giải trí over its budget, and a
	// shopping purchase that puts Mua sắm in the "near" band — see
	// sampleBudgets and the alert-triggering loop in seedSampleData.
	out = append(out,
		sampleTransaction{"expense", 250_000, "Giải trí", 0},
		sampleTransaction{"expense", 220_000, "Giải trí", 0},
		sampleTransaction{"expense", 1_180_000, "Mua sắm", 0},
	)
	return out
}

type sampleBudget struct {
	cat   string
	limit int64
}

// sampleBudgets deliberately leaves Giải trí and Mua sắm tight against the
// "today" spending buildSampleTransactions seeds for them, so a fresh demo
// account always has one "over" and one "near" category — the other three
// stay comfortably "within" so the dashboard doesn't read as all-alarms.
var sampleBudgets = []sampleBudget{
	{"Ăn uống", 3_500_000},
	{"Di chuyển", 900_000},
	{"Hóa đơn", 2_500_000},
	{"Mua sắm", 1_500_000},
	{"Giải trí", 400_000},
}

// seedSampleData populates a demo account with a 6-month transaction
// history, declared income/fixed costs/savings goal, budgets, and the
// budget-threshold alerts that history actually crosses — everything the
// dashboard's headline features (Safe-to-spend, alerts, 3-tier budget
// status) need to show real data on first load. Notes are left empty so
// rows display the (translatable) category name in either language.
func seedSampleData(ctx context.Context, tx *sql.Tx, userID string) error {
	catRepo := NewCategoryRepo(tx)
	txRepo := NewTransactionRepo(tx)

	byName, err := catRepo.ByName(ctx, userID)
	if err != nil {
		return err
	}

	now := time.Now()
	day := func(n int) string { return now.AddDate(0, 0, -n).Format(dateLayout) }
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

	if _, err := NewSettingsRepo(tx).Upsert(ctx, userID, demoSavingsGoal, demoCycleStartDay); err != nil {
		return err
	}

	for _, fc := range demoFixedCosts {
		fc.ID = uuid.NewString()
		if err := NewFixedCostRepo(tx).Create(ctx, userID, fc); err != nil {
			return err
		}
	}

	// Declare income for the current cycle and the one before it (matching
	// the two most recent "Lương" transactions above), so Safe-to-spend has
	// a real number and CycleUpdateSheet's vs-last-cycle comparison has
	// something to compare against.
	incomeRepo := NewIncomeRepo(tx)
	cycleStart, cycleEnd := CycleWindow(demoCycleStartDay, now)
	last := len(demoSalaries) - 1
	if _, err := incomeRepo.Upsert(ctx, userID, cycleStart, demoSalaries[last]); err != nil {
		return err
	}
	if prevCycles := PreviousCycles(demoCycleStartDay, now, 1); len(prevCycles) > 0 {
		if _, err := incomeRepo.Upsert(ctx, userID, prevCycles[0].Start, demoSalaries[last-1]); err != nil {
			return err
		}
	}

	budgetRepo := NewBudgetRepo(tx)
	alertRepo := NewAlertStateRepo(tx)
	month := cycleStart.Format("2006-01")
	for _, b := range sampleBudgets {
		cid := byName[b.cat]
		if cid == "" {
			continue
		}
		if _, err := budgetRepo.Upsert(ctx, userID, Budget{CategoryID: cid, Limit: b.limit, Month: month}); err != nil {
			return err
		}

		// Mirrors checkBudgetThreshold's crossing check (transactions.go)
		// with prevSpent=0, since this is a brand-new account: whatever this
		// cycle's seeded spend actually crosses, record it — so
		// GET /cycle/summary's activeAlerts isn't empty on a fresh demo.
		spent, err := txRepo.SumExpensesInCategory(ctx, userID, cid, cycleStart, cycleEnd)
		if err != nil {
			return err
		}
		if threshold, crossed := CrossedThreshold(0, spent, b.limit); crossed {
			if err := alertRepo.Trigger(ctx, userID, cid, cycleStart, threshold); err != nil {
				return err
			}
		}
	}
	return nil
}
