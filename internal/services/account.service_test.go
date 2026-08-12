package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"financal_management/internal/pkg/response"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------
// Repo giả
// ---------------------------------------------------------------------

type fakeAccountRepo struct {
	items map[uuid.UUID]sqlc.Account
	// txCount giả lập số giao dịch của từng ví.
	txCount  map[uuid.UUID]int64
	forceErr error
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{
		items:   make(map[uuid.UUID]sqlc.Account),
		txCount: make(map[uuid.UUID]int64),
	}
}

func (f *fakeAccountRepo) Create(_ context.Context, arg sqlc.CreateAccountParams) (sqlc.Account, error) {
	if f.forceErr != nil {
		return sqlc.Account{}, f.forceErr
	}
	// Giả lập ràng buộc duy nhất (user_id, tên ví).
	for _, a := range f.items {
		if a.UserID == arg.UserID && a.Name == arg.Name && a.DeletedAt == nil {
			return sqlc.Account{}, repo.ErrDuplicate
		}
	}

	account := sqlc.Account{
		ID: arg.ID, UserID: arg.UserID, Name: arg.Name, Type: arg.Type,
		Currency: arg.Currency, Balance: arg.Balance, Icon: arg.Icon,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	f.items[account.ID] = account
	return account, nil
}

func (f *fakeAccountRepo) Get(_ context.Context, id, userID uuid.UUID) (sqlc.Account, error) {
	if f.forceErr != nil {
		return sqlc.Account{}, f.forceErr
	}
	a, ok := f.items[id]
	// Điều kiện userID mô phỏng đúng câu SQL: ví của người khác coi như
	// không tồn tại.
	if !ok || a.UserID != userID || a.DeletedAt != nil {
		return sqlc.Account{}, repo.ErrAccountNotFound
	}
	return a, nil
}

func (f *fakeAccountRepo) List(_ context.Context, userID uuid.UUID) ([]sqlc.Account, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	out := make([]sqlc.Account, 0)
	for _, a := range f.items {
		if a.UserID == userID && a.DeletedAt == nil {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeAccountRepo) Update(_ context.Context, arg sqlc.UpdateAccountParams) (sqlc.Account, error) {
	if f.forceErr != nil {
		return sqlc.Account{}, f.forceErr
	}
	a, ok := f.items[arg.ID]
	if !ok || a.UserID != arg.UserID || a.DeletedAt != nil {
		return sqlc.Account{}, repo.ErrAccountNotFound
	}
	for id, other := range f.items {
		if id != arg.ID && other.UserID == arg.UserID && other.Name == arg.Name && other.DeletedAt == nil {
			return sqlc.Account{}, repo.ErrDuplicate
		}
	}
	a.Name, a.Type, a.Icon = arg.Name, arg.Type, arg.Icon
	f.items[arg.ID] = a
	return a, nil
}

func (f *fakeAccountRepo) SoftDelete(_ context.Context, id, userID uuid.UUID) (sqlc.Account, error) {
	if f.forceErr != nil {
		return sqlc.Account{}, f.forceErr
	}
	a, ok := f.items[id]
	if !ok || a.UserID != userID || a.DeletedAt != nil {
		return sqlc.Account{}, repo.ErrAccountNotFound
	}
	now := time.Now()
	a.DeletedAt = &now
	f.items[id] = a
	return a, nil
}

func (f *fakeAccountRepo) CountTransactions(_ context.Context, accountID uuid.UUID) (int64, error) {
	if f.forceErr != nil {
		return 0, f.forceErr
	}
	return f.txCount[accountID], nil
}

// ---------------------------------------------------------------------
// Test
// ---------------------------------------------------------------------

func newAccountService() (*AccountService, *fakeAccountRepo) {
	r := newFakeAccountRepo()
	return NewAccountService(r), r
}

func TestAccountCreate_ThanhCong(t *testing.T) {
	svc, _ := newAccountService()
	userID := uuid.New()

	account, err := svc.Create(context.Background(), CreateAccountInput{
		UserID:         userID,
		Name:           "  Tiền mặt  ",
		Type:           "cash",
		Currency:       "vnd",
		InitialBalance: decimal.RequireFromString("500000"),
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	// Tên phải được cắt khoảng trắng thừa, mã tiền tệ phải viết hoa.
	if account.Name != "Tiền mặt" {
		t.Errorf("Name = %q, mong đợi %q", account.Name, "Tiền mặt")
	}
	if account.Currency != "VND" {
		t.Errorf("Currency = %q, mong đợi VND", account.Currency)
	}
	if account.Balance.String() != "500000" {
		t.Errorf("Balance = %s, mong đợi 500000", account.Balance)
	}
}

func TestAccountCreate_LoaiViKhongHopLe(t *testing.T) {
	svc, _ := newAccountService()

	_, err := svc.Create(context.Background(), CreateAccountInput{
		UserID: uuid.New(), Name: "Ví lạ", Type: "bitcoin", Currency: "VND",
	})
	wantCode(t, err, response.CodeValidationFailed)
}

func TestAccountCreate_TrungTen(t *testing.T) {
	svc, _ := newAccountService()
	ctx := context.Background()
	userID := uuid.New()

	in := CreateAccountInput{UserID: userID, Name: "Tiền mặt", Type: "cash", Currency: "VND"}
	if _, err := svc.Create(ctx, in); err != nil {
		t.Fatalf("lần tạo đầu phải thành công: %v", err)
	}

	_, err := svc.Create(ctx, in)
	wantCode(t, err, response.CodeConflict)
}

// Hai người dùng khác nhau được đặt ví trùng tên.
func TestAccountCreate_HaiNguoiDungTrungTenViVanDuoc(t *testing.T) {
	svc, _ := newAccountService()
	ctx := context.Background()

	in := CreateAccountInput{UserID: uuid.New(), Name: "Tiền mặt", Type: "cash", Currency: "VND"}
	if _, err := svc.Create(ctx, in); err != nil {
		t.Fatalf("người dùng 1 tạo ví lỗi: %v", err)
	}

	in.UserID = uuid.New()
	if _, err := svc.Create(ctx, in); err != nil {
		t.Errorf("người dùng 2 phải đặt được ví trùng tên, lỗi: %v", err)
	}
}

// Đây là test bảo mật quan trọng nhất của module này: người dùng A không
// được đọc, sửa hay xoá ví của người dùng B. Và lỗi trả về phải là "không
// tìm thấy" chứ không phải "không có quyền" — nếu phân biệt, kẻ tấn công
// dò được id nào đang tồn tại.
func TestAccountKhongTruyCapDuocViCuaNguoiKhac(t *testing.T) {
	svc, _ := newAccountService()
	ctx := context.Background()

	chuSoHuu := uuid.New()
	keLa := uuid.New()

	account, err := svc.Create(ctx, CreateAccountInput{
		UserID: chuSoHuu, Name: "Ví riêng", Type: "bank", Currency: "VND",
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	t.Run("đọc", func(t *testing.T) {
		_, err := svc.Get(ctx, account.ID, keLa)
		wantCode(t, err, response.CodeNotFound)
	})

	t.Run("sửa", func(t *testing.T) {
		_, err := svc.Update(ctx, UpdateAccountInput{
			ID: account.ID, UserID: keLa, Name: "Bị chiếm", Type: "cash",
		})
		wantCode(t, err, response.CodeNotFound)
	})

	t.Run("xoá", func(t *testing.T) {
		err := svc.Delete(ctx, account.ID, keLa)
		wantCode(t, err, response.CodeNotFound)
	})

	t.Run("liệt kê", func(t *testing.T) {
		items, err := svc.List(ctx, keLa)
		if err != nil {
			t.Fatalf("List lỗi: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("người dùng khác thấy %d ví, mong đợi 0", len(items))
		}
	})
}

func TestAccountDelete_ConGiaoDichThiKhongXoaDuoc(t *testing.T) {
	svc, fake := newAccountService()
	ctx := context.Background()
	userID := uuid.New()

	account, err := svc.Create(ctx, CreateAccountInput{
		UserID: userID, Name: "Ví có giao dịch", Type: "cash", Currency: "VND",
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	fake.txCount[account.ID] = 5

	err = svc.Delete(ctx, account.ID, userID)
	wantCode(t, err, response.CodeConflict)
}

func TestAccountDelete_KhongConGiaoDichThiXoaDuoc(t *testing.T) {
	svc, _ := newAccountService()
	ctx := context.Background()
	userID := uuid.New()

	account, err := svc.Create(ctx, CreateAccountInput{
		UserID: userID, Name: "Ví trống", Type: "cash", Currency: "VND",
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	if err := svc.Delete(ctx, account.ID, userID); err != nil {
		t.Fatalf("Delete lỗi: %v", err)
	}

	// Sau khi xoá mềm thì không đọc được nữa.
	_, err = svc.Get(ctx, account.ID, userID)
	wantCode(t, err, response.CodeNotFound)
}

// Xoá mềm phải nhả lại tên ví để người dùng đặt lại tên đó cho ví mới.
func TestAccountDelete_XoaRoiDatLaiTenDuoc(t *testing.T) {
	svc, _ := newAccountService()
	ctx := context.Background()
	userID := uuid.New()

	in := CreateAccountInput{UserID: userID, Name: "Tiền mặt", Type: "cash", Currency: "VND"}
	account, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}
	if err := svc.Delete(ctx, account.ID, userID); err != nil {
		t.Fatalf("Delete lỗi: %v", err)
	}

	if _, err := svc.Create(ctx, in); err != nil {
		t.Errorf("phải tạo lại được ví cùng tên sau khi xoá, lỗi: %v", err)
	}
}

func TestAccountUpdate_ThanhCong(t *testing.T) {
	svc, _ := newAccountService()
	ctx := context.Background()
	userID := uuid.New()

	account, err := svc.Create(ctx, CreateAccountInput{
		UserID: userID, Name: "Ví cũ", Type: "cash", Currency: "VND",
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	updated, err := svc.Update(ctx, UpdateAccountInput{
		ID: account.ID, UserID: userID, Name: "Ví mới", Type: "bank", Icon: "🏦",
	})
	if err != nil {
		t.Fatalf("Update lỗi: %v", err)
	}

	if updated.Name != "Ví mới" || updated.Type != "bank" {
		t.Errorf("cập nhật sai: %+v", updated)
	}
	// Loại tiền và số dư không được đổi qua Update.
	if updated.Currency != "VND" {
		t.Errorf("Update không được đổi loại tiền, đang là %s", updated.Currency)
	}
}

func TestAccountList_LoiDatabase(t *testing.T) {
	svc, fake := newAccountService()
	fake.forceErr = errors.New("mất kết nối")

	_, err := svc.List(context.Background(), uuid.New())
	wantCode(t, err, response.CodeDatabaseError)

	if !errors.Is(err, fake.forceErr) {
		t.Error("lỗi gốc phải được giữ lại bên trong để ghi log")
	}
}
