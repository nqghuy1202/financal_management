package services

import (
	"context"
	"testing"
	"time"

	"financal_management/internal/model"
	"financal_management/internal/pkg/response"
	"financal_management/internal/repo"
	"financal_management/internal/repo/sqlc"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------
// Repo giả
// ---------------------------------------------------------------------

type fakeCategoryRepo struct {
	items    map[uuid.UUID]sqlc.Category
	txCount  map[uuid.UUID]int64
	forceErr error
}

func newFakeCategoryRepo() *fakeCategoryRepo {
	return &fakeCategoryRepo{
		items:   make(map[uuid.UUID]sqlc.Category),
		txCount: make(map[uuid.UUID]int64),
	}
}

// addSystem thêm một danh mục hệ thống (user_id NULL).
func (f *fakeCategoryRepo) addSystem(name, categoryType string) sqlc.Category {
	c := sqlc.Category{
		ID: uuid.New(), UserID: nil, Name: name, Type: categoryType,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	f.items[c.ID] = c
	return c
}

func (f *fakeCategoryRepo) List(_ context.Context, userID uuid.UUID, categoryType string) ([]sqlc.Category, error) {
	if f.forceErr != nil {
		return nil, f.forceErr
	}
	out := make([]sqlc.Category, 0)
	for _, c := range f.items {
		if c.DeletedAt != nil {
			continue
		}
		// Thấy danh mục hệ thống và danh mục của chính mình.
		if c.UserID != nil && *c.UserID != userID {
			continue
		}
		if categoryType != "" && c.Type != categoryType {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeCategoryRepo) Get(_ context.Context, id, userID uuid.UUID) (sqlc.Category, error) {
	if f.forceErr != nil {
		return sqlc.Category{}, f.forceErr
	}
	c, ok := f.items[id]
	if !ok || c.DeletedAt != nil {
		return sqlc.Category{}, repo.ErrCategoryNotFound
	}
	if c.UserID != nil && *c.UserID != userID {
		return sqlc.Category{}, repo.ErrCategoryNotFound
	}
	return c, nil
}

func (f *fakeCategoryRepo) Create(_ context.Context, arg sqlc.CreateCategoryParams) (sqlc.Category, error) {
	if f.forceErr != nil {
		return sqlc.Category{}, f.forceErr
	}
	for _, c := range f.items {
		if c.DeletedAt != nil || c.Name != arg.Name || c.Type != arg.Type {
			continue
		}
		// Trùng với danh mục hệ thống, hoặc với danh mục của chính mình.
		if c.UserID == nil || (arg.UserID != nil && *c.UserID == *arg.UserID) {
			return sqlc.Category{}, repo.ErrDuplicate
		}
	}
	c := sqlc.Category{
		ID: arg.ID, UserID: arg.UserID, Name: arg.Name, Type: arg.Type,
		Icon: arg.Icon, Color: arg.Color, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	f.items[c.ID] = c
	return c, nil
}

// Update và SoftDelete mô phỏng đúng điều kiện `user_id = @user_id` của
// câu SQL: danh mục hệ thống có UserID nil nên không bao giờ khớp.
func (f *fakeCategoryRepo) Update(_ context.Context, arg sqlc.UpdateCategoryParams) (sqlc.Category, error) {
	if f.forceErr != nil {
		return sqlc.Category{}, f.forceErr
	}
	c, ok := f.items[arg.ID]
	if !ok || c.DeletedAt != nil || c.UserID == nil || arg.UserID == nil || *c.UserID != *arg.UserID {
		return sqlc.Category{}, repo.ErrCategoryNotFound
	}
	c.Name, c.Icon, c.Color = arg.Name, arg.Icon, arg.Color
	f.items[arg.ID] = c
	return c, nil
}

func (f *fakeCategoryRepo) SoftDelete(_ context.Context, id, userID uuid.UUID) (sqlc.Category, error) {
	if f.forceErr != nil {
		return sqlc.Category{}, f.forceErr
	}
	c, ok := f.items[id]
	if !ok || c.DeletedAt != nil || c.UserID == nil || *c.UserID != userID {
		return sqlc.Category{}, repo.ErrCategoryNotFound
	}
	now := time.Now()
	c.DeletedAt = &now
	f.items[id] = c
	return c, nil
}

func (f *fakeCategoryRepo) CountTransactions(_ context.Context, categoryID uuid.UUID) (int64, error) {
	if f.forceErr != nil {
		return 0, f.forceErr
	}
	return f.txCount[categoryID], nil
}

// ---------------------------------------------------------------------
// Test
// ---------------------------------------------------------------------

func newCategoryService() (*CategoryService, *fakeCategoryRepo) {
	r := newFakeCategoryRepo()
	return NewCategoryService(r), r
}

// Người dùng phải thấy được danh mục hệ thống ngay cả khi chưa tạo danh
// mục nào của riêng mình.
func TestCategoryList_ThayDanhMucHeThong(t *testing.T) {
	svc, fake := newCategoryService()
	fake.addSystem("Ăn uống", model.CategoryTypeExpense)
	fake.addSystem("Lương", model.CategoryTypeIncome)

	items, err := svc.List(context.Background(), uuid.New(), "")
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("thấy %d danh mục, mong đợi 2", len(items))
	}
}

func TestCategoryList_LocTheoLoai(t *testing.T) {
	svc, fake := newCategoryService()
	fake.addSystem("Ăn uống", model.CategoryTypeExpense)
	fake.addSystem("Đi lại", model.CategoryTypeExpense)
	fake.addSystem("Lương", model.CategoryTypeIncome)

	items, err := svc.List(context.Background(), uuid.New(), model.CategoryTypeExpense)
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("lọc theo expense được %d, mong đợi 2", len(items))
	}
}

func TestCategoryList_LoaiKhongHopLe(t *testing.T) {
	svc, _ := newCategoryService()
	_, err := svc.List(context.Background(), uuid.New(), "chuyen_khoan")
	wantCode(t, err, response.CodeValidationFailed)
}

// Danh mục riêng của người này không được lọt sang người khác.
func TestCategoryList_KhongThayDanhMucRiengCuaNguoiKhac(t *testing.T) {
	svc, fake := newCategoryService()
	ctx := context.Background()
	fake.addSystem("Ăn uống", model.CategoryTypeExpense)

	nguoiA := uuid.New()
	if _, err := svc.Create(ctx, CreateCategoryInput{
		UserID: nguoiA, Name: "Cà phê sáng", Type: model.CategoryTypeExpense,
	}); err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	// Người A thấy 2 (1 hệ thống + 1 riêng), người B chỉ thấy 1.
	itemsA, err := svc.List(ctx, nguoiA, "")
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if len(itemsA) != 2 {
		t.Errorf("người A thấy %d danh mục, mong đợi 2", len(itemsA))
	}

	itemsB, err := svc.List(ctx, uuid.New(), "")
	if err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if len(itemsB) != 1 {
		t.Errorf("người B thấy %d danh mục, mong đợi 1 (chỉ danh mục hệ thống)", len(itemsB))
	}
}

// Đây là quy tắc quan trọng nhất của module này: ai cũng ĐỌC được danh
// mục hệ thống, nhưng không ai SỬA hay XOÁ được nó.
func TestCategoryDanhMucHeThongKhongSuaXoaDuoc(t *testing.T) {
	svc, fake := newCategoryService()
	ctx := context.Background()
	userID := uuid.New()

	heThong := fake.addSystem("Ăn uống", model.CategoryTypeExpense)

	t.Run("đọc được", func(t *testing.T) {
		if _, err := svc.Get(ctx, heThong.ID, userID); err != nil {
			t.Errorf("phải đọc được danh mục hệ thống, lỗi: %v", err)
		}
	})

	t.Run("không sửa được", func(t *testing.T) {
		_, err := svc.Update(ctx, UpdateCategoryInput{
			ID: heThong.ID, UserID: userID, Name: "Bị đổi tên",
		})
		wantCode(t, err, response.CodeNotFound)
	})

	t.Run("không xoá được", func(t *testing.T) {
		err := svc.Delete(ctx, heThong.ID, userID)
		wantCode(t, err, response.CodeNotFound)
	})
}

func TestCategoryCreate_ThanhCong(t *testing.T) {
	svc, _ := newCategoryService()
	userID := uuid.New()

	category, err := svc.Create(context.Background(), CreateCategoryInput{
		UserID: userID, Name: "  Cà phê  ", Type: model.CategoryTypeExpense, Icon: "☕",
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	if category.Name != "Cà phê" {
		t.Errorf("Name = %q, mong đợi %q", category.Name, "Cà phê")
	}
	if category.UserID == nil || *category.UserID != userID {
		t.Error("danh mục tự tạo phải gắn với người dùng, không được là danh mục hệ thống")
	}
}

func TestCategoryCreate_LoaiKhongHopLe(t *testing.T) {
	svc, _ := newCategoryService()
	_, err := svc.Create(context.Background(), CreateCategoryInput{
		UserID: uuid.New(), Name: "Lạ", Type: "transfer",
	})
	wantCode(t, err, response.CodeValidationFailed)
}

func TestCategoryCreate_TrungTenCuaChinhMinh(t *testing.T) {
	svc, _ := newCategoryService()
	ctx := context.Background()
	in := CreateCategoryInput{UserID: uuid.New(), Name: "Cà phê", Type: model.CategoryTypeExpense}

	if _, err := svc.Create(ctx, in); err != nil {
		t.Fatalf("lần tạo đầu phải thành công: %v", err)
	}
	_, err := svc.Create(ctx, in)
	wantCode(t, err, response.CodeConflict)
}

// Hai người dùng khác nhau được đặt danh mục trùng tên.
func TestCategoryCreate_HaiNguoiDungTrungTenVanDuoc(t *testing.T) {
	svc, _ := newCategoryService()
	ctx := context.Background()

	in := CreateCategoryInput{UserID: uuid.New(), Name: "Cà phê", Type: model.CategoryTypeExpense}
	if _, err := svc.Create(ctx, in); err != nil {
		t.Fatalf("người dùng 1 lỗi: %v", err)
	}
	in.UserID = uuid.New()
	if _, err := svc.Create(ctx, in); err != nil {
		t.Errorf("người dùng 2 phải tạo được danh mục trùng tên, lỗi: %v", err)
	}
}

func TestCategoryDelete_ConGiaoDichThiKhongXoaDuoc(t *testing.T) {
	svc, fake := newCategoryService()
	ctx := context.Background()
	userID := uuid.New()

	category, err := svc.Create(ctx, CreateCategoryInput{
		UserID: userID, Name: "Cà phê", Type: model.CategoryTypeExpense,
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}
	fake.txCount[category.ID] = 3

	err = svc.Delete(ctx, category.ID, userID)
	wantCode(t, err, response.CodeConflict)
}

func TestCategoryUpdate_ThanhCong(t *testing.T) {
	svc, _ := newCategoryService()
	ctx := context.Background()
	userID := uuid.New()

	category, err := svc.Create(ctx, CreateCategoryInput{
		UserID: userID, Name: "Cà phê", Type: model.CategoryTypeExpense,
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}

	updated, err := svc.Update(ctx, UpdateCategoryInput{
		ID: category.ID, UserID: userID, Name: "Cà phê & trà", Icon: "🍵",
	})
	if err != nil {
		t.Fatalf("Update lỗi: %v", err)
	}
	if updated.Name != "Cà phê & trà" {
		t.Errorf("Name = %q, mong đợi %q", updated.Name, "Cà phê & trà")
	}
	// Loại thu/chi không được đổi qua Update.
	if updated.Type != model.CategoryTypeExpense {
		t.Errorf("Update không được đổi loại danh mục, đang là %s", updated.Type)
	}
}
