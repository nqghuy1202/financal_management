package services

import (
	"context"
	"sort"
	"testing"
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------
// Repo giả
// ---------------------------------------------------------------------

type fakeTransactionRepo struct {
	items    map[uuid.UUID]sqlc.Transaction
	forceErr error
}

func newFakeTransactionRepo() *fakeTransactionRepo {
	return &fakeTransactionRepo{items: make(map[uuid.UUID]sqlc.Transaction)}
}

func (f *fakeTransactionRepo) Create(_ context.Context, arg sqlc.CreateTransactionParams) (sqlc.Transaction, error) {
	if f.forceErr != nil {
		return sqlc.Transaction{}, f.forceErr
	}
	tx := sqlc.Transaction{
		ID: arg.ID, UserID: arg.UserID, AccountID: arg.AccountID,
		CategoryID: arg.CategoryID, Type: arg.Type, Amount: arg.Amount,
		Currency: arg.Currency, Note: arg.Note, OccurredAt: arg.OccurredAt,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	f.items[tx.ID] = tx
	return tx, nil
}

func (f *fakeTransactionRepo) Get(_ context.Context, id, userID uuid.UUID) (sqlc.Transaction, error) {
	if f.forceErr != nil {
		return sqlc.Transaction{}, f.forceErr
	}
	tx, ok := f.items[id]
	if !ok || tx.UserID != userID || tx.DeletedAt != nil {
		return sqlc.Transaction{}, repo.ErrTransactionNotFound
	}
	return tx, nil
}

func (f *fakeTransactionRepo) Update(_ context.Context, arg sqlc.UpdateTransactionParams) (sqlc.Transaction, error) {
	if f.forceErr != nil {
		return sqlc.Transaction{}, f.forceErr
	}
	tx, ok := f.items[arg.ID]
	if !ok || tx.UserID != arg.UserID || tx.DeletedAt != nil {
		return sqlc.Transaction{}, repo.ErrTransactionNotFound
	}
	tx.AccountID, tx.CategoryID = arg.AccountID, arg.CategoryID
	tx.Type, tx.Amount, tx.Note, tx.OccurredAt = arg.Type, arg.Amount, arg.Note, arg.OccurredAt
	f.items[arg.ID] = tx
	return tx, nil
}

func (f *fakeTransactionRepo) SoftDelete(_ context.Context, id, userID uuid.UUID) (sqlc.Transaction, error) {
	if f.forceErr != nil {
		return sqlc.Transaction{}, f.forceErr
	}
	tx, ok := f.items[id]
	if !ok || tx.UserID != userID || tx.DeletedAt != nil {
		return sqlc.Transaction{}, repo.ErrTransactionNotFound
	}
	now := time.Now()
	tx.DeletedAt = &now
	f.items[id] = tx
	return tx, nil
}

// matches mô phỏng mệnh đề WHERE của câu SQL thật.
func matches(tx sqlc.Transaction, userID uuid.UUID, txType *string,
	categoryID, accountID *uuid.UUID, from, to *time.Time) bool {
	if tx.UserID != userID || tx.DeletedAt != nil {
		return false
	}
	if txType != nil && tx.Type != *txType {
		return false
	}
	if categoryID != nil && (tx.CategoryID == nil || *tx.CategoryID != *categoryID) {
		return false
	}
	if accountID != nil && (tx.AccountID == nil || *tx.AccountID != *accountID) {
		return false
	}
	if from != nil && tx.OccurredAt.Before(*from) {
		return false
	}
	if to != nil && !tx.OccurredAt.Before(*to) {
		return false
	}
	return true
}

func (f *fakeTransactionRepo) List(_ context.Context, arg sqlc.ListTransactionsParams) ([]sqlc.Transaction, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	out := make([]sqlc.Transaction, 0)
	for _, tx := range f.items {
		if !matches(tx, arg.UserID, arg.Type, arg.CategoryID, arg.AccountID, arg.From, arg.To) {
			continue
		}
		// Mô phỏng điều kiện con trỏ: (occurred_at, id) < (cursor...)
		if arg.CursorOccurredAt != nil {
			if tx.OccurredAt.After(*arg.CursorOccurredAt) {
				continue
			}
			if tx.OccurredAt.Equal(*arg.CursorOccurredAt) &&
				tx.ID.String() >= arg.CursorID.String() {
				continue
			}
		}
		out = append(out, tx)
	}

	// ORDER BY occurred_at DESC, id DESC
	sort.Slice(out, func(i, j int) bool {
		if !out[i].OccurredAt.Equal(out[j].OccurredAt) {
			return out[i].OccurredAt.After(out[j].OccurredAt)
		}
		return out[i].ID.String() > out[j].ID.String()
	})

	if int(arg.PageSize) < len(out) {
		out = out[:arg.PageSize]
	}
	return out, nil
}

func (f *fakeTransactionRepo) Count(_ context.Context, arg sqlc.CountTransactionsParams) (int64, error) {
	if f.forceErr != nil {
		return 0, f.forceErr
	}
	var n int64
	for _, tx := range f.items {
		if matches(tx, arg.UserID, arg.Type, arg.CategoryID, arg.AccountID, arg.From, arg.To) {
			n++
		}
	}
	return n, nil
}

// ---------------------------------------------------------------------
// Hàm dựng test
// ---------------------------------------------------------------------

type txFixture struct {
	svc        *TransactionService
	txRepo     *fakeTransactionRepo
	accRepo    *fakeAccountRepo
	catRepo    *fakeCategoryRepo
	userID     uuid.UUID
	accountID  uuid.UUID
	expenseCat uuid.UUID
	incomeCat  uuid.UUID
}

func newTxFixture(t *testing.T) *txFixture {
	t.Helper()

	txRepo := newFakeTransactionRepo()
	accRepo := newFakeAccountRepo()
	catRepo := newFakeCategoryRepo()
	userID := uuid.New()

	account, err := accRepo.Create(context.Background(), sqlc.CreateAccountParams{
		ID: uuid.New(), UserID: userID, Name: "Tiền mặt", Type: model.AccountTypeCash,
	})
	if err != nil {
		t.Fatalf("tạo nguồn tiền mẫu lỗi: %v", err)
	}

	expenseCat := catRepo.addSystem("Ăn uống", model.CategoryTypeExpense)
	incomeCat := catRepo.addSystem("Lương", model.CategoryTypeIncome)

	return &txFixture{
		svc:        NewTransactionService(txRepo, accRepo, catRepo),
		txRepo:     txRepo,
		accRepo:    accRepo,
		catRepo:    catRepo,
		userID:     userID,
		accountID:  account.ID,
		expenseCat: expenseCat.ID,
		incomeCat:  incomeCat.ID,
	}
}

func amount(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// ---------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------

func TestTransactionCreate_ThanhCong(t *testing.T) {
	f := newTxFixture(t)

	tx, err := f.svc.Create(context.Background(), CreateTransactionInput{
		UserID:     f.userID,
		AccountID:  &f.accountID,
		CategoryID: &f.expenseCat,
		Type:       model.TransactionTypeExpense,
		Amount:     amount("50000"),
		Note:       "  Ăn trưa  ",
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	if tx.Amount.String() != "50000" {
		t.Errorf("Amount = %s, mong đợi 50000", tx.Amount)
	}
	if tx.Note != "Ăn trưa" {
		t.Errorf("Note = %q, mong đợi %q (phải cắt khoảng trắng)", tx.Note, "Ăn trưa")
	}
	if tx.Currency != model.DefaultCurrency {
		t.Errorf("Currency = %s, mong đợi %s", tx.Currency, model.DefaultCurrency)
	}
}

// Ghi nhanh: không chọn nguồn tiền và danh mục vẫn phải được.
func TestTransactionCreate_KhongCanDanhMucVaNguonTien(t *testing.T) {
	f := newTxFixture(t)

	tx, err := f.svc.Create(context.Background(), CreateTransactionInput{
		UserID:     f.userID,
		Type:       model.TransactionTypeExpense,
		Amount:     amount("30000"),
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}
	if tx.CategoryID != nil || tx.AccountID != nil {
		t.Error("không chọn thì hai trường này phải là NULL")
	}
}

// Đây là quy tắc nghiệp vụ quan trọng nhất: khoản chi phải gắn danh mục
// chi. Nếu lẫn lộn, báo cáo "chi tiêu theo danh mục" sẽ hiện danh mục
// Lương nằm trong phần chi tiêu.
func TestTransactionCreate_LoaiDanhMucPhaiKhopLoaiGiaoDich(t *testing.T) {
	f := newTxFixture(t)
	ctx := context.Background()

	t.Run("khoản chi gắn danh mục thu", func(t *testing.T) {
		_, err := f.svc.Create(ctx, CreateTransactionInput{
			UserID: f.userID, CategoryID: &f.incomeCat,
			Type: model.TransactionTypeExpense, Amount: amount("50000"),
			OccurredAt: time.Now(),
		})
		wantCode(t, err, response.CodeValidationFailed)
	})

	t.Run("khoản thu gắn danh mục chi", func(t *testing.T) {
		_, err := f.svc.Create(ctx, CreateTransactionInput{
			UserID: f.userID, CategoryID: &f.expenseCat,
			Type: model.TransactionTypeIncome, Amount: amount("50000"),
			OccurredAt: time.Now(),
		})
		wantCode(t, err, response.CodeValidationFailed)
	})
}

func TestTransactionCreate_SoTienPhaiDuong(t *testing.T) {
	f := newTxFixture(t)
	ctx := context.Background()

	for _, v := range []string{"0", "-1000"} {
		_, err := f.svc.Create(ctx, CreateTransactionInput{
			UserID: f.userID, Type: model.TransactionTypeExpense,
			Amount: amount(v), OccurredAt: time.Now(),
		})
		wantCode(t, err, response.CodeValidationFailed)
	}
}

func TestTransactionCreate_LoaiKhongHopLe(t *testing.T) {
	f := newTxFixture(t)

	_, err := f.svc.Create(context.Background(), CreateTransactionInput{
		UserID: f.userID, Type: "transfer", Amount: amount("1000"),
		OccurredAt: time.Now(),
	})
	wantCode(t, err, response.CodeValidationFailed)
}

// Không được gắn giao dịch vào nguồn tiền của người khác.
func TestTransactionCreate_NguonTienCuaNguoiKhac(t *testing.T) {
	f := newTxFixture(t)
	ctx := context.Background()

	nguoiKhac, err := f.accRepo.Create(ctx, sqlc.CreateAccountParams{
		ID: uuid.New(), UserID: uuid.New(), Name: "Ví người khác", Type: model.AccountTypeCash,
	})
	if err != nil {
		t.Fatalf("tạo ví lỗi: %v", err)
	}

	_, err = f.svc.Create(ctx, CreateTransactionInput{
		UserID: f.userID, AccountID: &nguoiKhac.ID,
		Type: model.TransactionTypeExpense, Amount: amount("50000"),
		OccurredAt: time.Now(),
	})
	wantCode(t, err, response.CodeNotFound)
}

// ---------------------------------------------------------------------
// Get / Update / Delete
// ---------------------------------------------------------------------

func TestTransactionKhongTruyCapDuocGiaoDichCuaNguoiKhac(t *testing.T) {
	f := newTxFixture(t)
	ctx := context.Background()
	keLa := uuid.New()

	tx, err := f.svc.Create(ctx, CreateTransactionInput{
		UserID: f.userID, Type: model.TransactionTypeExpense,
		Amount: amount("50000"), OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	t.Run("đọc", func(t *testing.T) {
		_, err := f.svc.Get(ctx, tx.ID, keLa)
		wantCode(t, err, response.CodeNotFound)
	})
	t.Run("sửa", func(t *testing.T) {
		_, err := f.svc.Update(ctx, UpdateTransactionInput{
			ID: tx.ID, UserID: keLa, Type: model.TransactionTypeExpense,
			Amount: amount("1"), OccurredAt: time.Now(),
		})
		wantCode(t, err, response.CodeNotFound)
	})
	t.Run("xoá", func(t *testing.T) {
		err := f.svc.Delete(ctx, tx.ID, keLa)
		wantCode(t, err, response.CodeNotFound)
	})
}

func TestTransactionDelete_XoaRoiKhongDocDuoc(t *testing.T) {
	f := newTxFixture(t)
	ctx := context.Background()

	tx, err := f.svc.Create(ctx, CreateTransactionInput{
		UserID: f.userID, Type: model.TransactionTypeExpense,
		Amount: amount("50000"), OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	if err := f.svc.Delete(ctx, tx.ID, f.userID); err != nil {
		t.Fatalf("Delete lỗi: %v", err)
	}
	_, err = f.svc.Get(ctx, tx.ID, f.userID)
	wantCode(t, err, response.CodeNotFound)
}

// ---------------------------------------------------------------------
// List
// ---------------------------------------------------------------------

// seed tạo n giao dịch, mỗi cái cách nhau một ngày lùi dần từ hôm nay.
func (f *txFixture) seed(t *testing.T, n int, txType string, categoryID *uuid.UUID) {
	t.Helper()
	base := time.Now()
	for i := 0; i < n; i++ {
		if _, err := f.svc.Create(context.Background(), CreateTransactionInput{
			UserID: f.userID, CategoryID: categoryID, Type: txType,
			Amount: amount("10000"), OccurredAt: base.AddDate(0, 0, -i),
		}); err != nil {
			t.Fatalf("seed giao dịch %d lỗi: %v", i, err)
		}
	}
}

func TestTransactionList_PhanTrangBangConTro(t *testing.T) {
	f := newTxFixture(t)
	ctx := context.Background()
	f.seed(t, 25, model.TransactionTypeExpense, &f.expenseCat)

	// Trang đầu.
	first, err := f.svc.List(ctx, ListTransactionsInput{UserID: f.userID, PageSize: 10})
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if len(first.Items) != 10 {
		t.Fatalf("trang đầu có %d bản ghi, mong đợi 10", len(first.Items))
	}
	if first.Total != 25 {
		t.Errorf("Total = %d, mong đợi 25", first.Total)
	}
	if first.NextCursorID == nil {
		t.Fatal("còn dữ liệu thì phải có con trỏ trang sau")
	}

	// Trang hai, dùng con trỏ của trang đầu.
	second, err := f.svc.List(ctx, ListTransactionsInput{
		UserID:           f.userID,
		PageSize:         10,
		CursorOccurredAt: first.NextCursorOccurredAt,
		CursorID:         first.NextCursorID,
	})
	if err != nil {
		t.Fatalf("List trang 2 lỗi: %v", err)
	}
	if len(second.Items) != 10 {
		t.Fatalf("trang hai có %d bản ghi, mong đợi 10", len(second.Items))
	}

	// Hai trang không được lặp bản ghi nào.
	seen := make(map[uuid.UUID]bool, 20)
	for _, tx := range append(append([]sqlc.Transaction{}, first.Items...), second.Items...) {
		if seen[tx.ID] {
			t.Fatalf("bản ghi %s xuất hiện ở cả hai trang", tx.ID)
		}
		seen[tx.ID] = true
	}
}

// Trang cuối không được trả về con trỏ, nếu không client sẽ gọi thêm một
// lần vô ích.
func TestTransactionList_TrangCuoiKhongCoConTro(t *testing.T) {
	f := newTxFixture(t)
	f.seed(t, 5, model.TransactionTypeExpense, &f.expenseCat)

	page, err := f.svc.List(context.Background(), ListTransactionsInput{
		UserID: f.userID, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if len(page.Items) != 5 {
		t.Errorf("có %d bản ghi, mong đợi 5", len(page.Items))
	}
	if page.NextCursorID != nil {
		t.Error("hết dữ liệu thì không được trả về con trỏ")
	}
}

func TestTransactionList_LocTheoLoai(t *testing.T) {
	f := newTxFixture(t)
	f.seed(t, 3, model.TransactionTypeExpense, &f.expenseCat)
	f.seed(t, 2, model.TransactionTypeIncome, &f.incomeCat)

	page, err := f.svc.List(context.Background(), ListTransactionsInput{
		UserID: f.userID, Type: model.TransactionTypeIncome,
	})
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if page.Total != 2 {
		t.Errorf("lọc income được %d, mong đợi 2", page.Total)
	}
}

func TestTransactionList_LocTheoKhoangThoiGian(t *testing.T) {
	f := newTxFixture(t)
	f.seed(t, 10, model.TransactionTypeExpense, &f.expenseCat)

	// Chỉ lấy 3 ngày gần nhất.
	from := time.Now().AddDate(0, 0, -2).Truncate(24 * time.Hour)
	page, err := f.svc.List(context.Background(), ListTransactionsInput{
		UserID: f.userID, From: &from,
	})
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if page.Total != 3 {
		t.Errorf("lọc 3 ngày gần nhất được %d, mong đợi 3", page.Total)
	}
}

func TestTransactionList_KhoangThoiGianNguoc(t *testing.T) {
	f := newTxFixture(t)
	from := time.Now()
	to := from.AddDate(0, 0, -5)

	_, err := f.svc.List(context.Background(), ListTransactionsInput{
		UserID: f.userID, From: &from, To: &to,
	})
	wantCode(t, err, response.CodeValidationFailed)
}

// Không ai được yêu cầu trang quá lớn để tránh kéo cả bảng về.
func TestTransactionList_GioiHanKichThuocTrang(t *testing.T) {
	f := newTxFixture(t)
	f.seed(t, 30, model.TransactionTypeExpense, &f.expenseCat)

	page, err := f.svc.List(context.Background(), ListTransactionsInput{
		UserID: f.userID, PageSize: 10000,
	})
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if len(page.Items) > maxPageSize {
		t.Errorf("trả về %d bản ghi, vượt giới hạn %d", len(page.Items), maxPageSize)
	}
}

func TestTransactionList_KhongThayGiaoDichCuaNguoiKhac(t *testing.T) {
	f := newTxFixture(t)
	f.seed(t, 5, model.TransactionTypeExpense, &f.expenseCat)

	page, err := f.svc.List(context.Background(), ListTransactionsInput{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("người khác thấy %d giao dịch, mong đợi 0", page.Total)
	}
}
