---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - '{planning_artifacts}/prds/prd-financal_management-2026-09-11/prd.md'
  - '{planning_artifacts}/architecture/architecture-financal_management-2026-09-12/ARCHITECTURE-SPINE.md'
  - '{planning_artifacts}/ux-designs/ux-financal_management-2026-09-11/DESIGN.md'
  - '{planning_artifacts}/ux-designs/ux-financal_management-2026-09-11/EXPERIENCE.md'
---

# HL Personal Finance — Cải tổ - Epic Breakdown

## Overview

Tài liệu này chia PRD, UX (DESIGN.md + EXPERIENCE.md), và Kiến trúc (ARCHITECTURE-SPINE.md) của bản cải tổ "HL Personal Finance" thành epic và story triển khai được.

## Requirements Inventory

### Functional Requirements

FR1: Hiển thị Safe-to-spend khi mở app (Dashboard, tính lại mỗi lần tải, không cache cũ).
FR2: Khai báo/cập nhật Lương cho chu kỳ hiện tại (nhập tay, không tự chuyển sang chu kỳ sau).
FR3: Khai báo Chi phí cố định và Mục tiêu tiết kiệm (CRUD danh sách chi phí cố định, chỉ nhập tay).
FR4: Cấu hình ngày bắt đầu Chu kỳ ngân sách (mặc định ngày 1, có thể đổi).
FR5: Phát cảnh báo khi chạm Ngưỡng cảnh báo (70/90/100% ngân sách danh mục, hiển thị trong app).
FR6: Không lặp lại cảnh báo cùng ngưỡng trong cùng chu kỳ.
FR7: Tổng quan trạng thái ngân sách trên Dashboard (3 mức: trong hạn mức/gần vượt/đã vượt).
~~FR8: Gắn cờ giao dịch/danh mục bất thường (>1.5 lần trung bình 3 chu kỳ gần nhất).~~
~~FR9: Đánh dấu đã xem xét cờ bất thường (ẩn cờ vĩnh viễn cho giao dịch đó).~~
~~FR10: Không gắn cờ khi thiếu dữ liệu nền (cần ≥1 chu kỳ dữ liệu đầy đủ).~~
**[CẮT KHỎI V1 — 2026-09-13]** FR8/FR9/FR10 (Epic 3) bị cắt khỏi kế hoạch theo quyết định của người dùng: dự án một-người-dùng không cần tính năng tự động gắn cờ chi tiêu bất thường. Xem `sprint-change-proposal-2026-09-13.md`.

### NonFunctional Requirements

NFR1 (Reliability): Sau khi hoàn thiện refactor backend, toàn bộ hành vi CRUD hiện có (giao dịch, danh mục, ngân sách, auth, dashboard) phải cho kết quả nhất quán với trước refactor — không mất dữ liệu, không sai lệch số liệu.
NFR2 (Performance): Safe-to-spend và trạng thái ngân sách tải xong dưới ~1 giây trên kết nối di động thông thường.
NFR3 (Security & Privacy): Dữ liệu tài chính của một người dùng chỉ thuộc về người dùng đó (JWT + bcrypt + scoping theo user hiện có); không thêm bên thứ ba theo dõi/phân tích dữ liệu.
NFR4 (Cost): Không giới thiệu hạ tầng trả phí mới — tiếp tục self-host/free-tier.
NFR5 (Accessibility, từ EXPERIENCE.md): WCAG AA cho mọi màu mới; không dựa vào màu sắc một mình để truyền đạt ý nghĩa; touch-target tối thiểu 40×40px; `role="alert"` cho alert-banner; tôn trọng `prefers-reduced-motion` hiện có.

### Additional Requirements (từ Kiến trúc)

- Không có starter template — đây là dự án brownfield, mở rộng `internal/api` hiện có, giữ nguyên paradigm layered-lite đã có (AD-1).
- **Ranh giới phạm vi bắt buộc**: không đọc/sửa/dùng làm tham chiếu bất kỳ phần nào của "learning scaffold" (`internal/controller`, `internal/services`, `internal/repo`, `internal/routers`, `internal/server`, `internal/database`, `internal/initialize`, `internal/middlewares`, `global/`, `config/*.yaml`, `cmd/api`, `cmd/server`, `cmd/cli`) — AD-2.
- 5 bảng mới qua `Migrate()` hiện có (`user_settings`, `fixed_costs`, `incomes`, `budget_alert_state`, `transaction_flags`) — không ALTER 4 bảng gốc (AD-5).
- Một hàm chia sẻ duy nhất `CycleWindow()`/`PreviousCycles()` trong `cycle.go`, tạo bởi story FR-1 (AD-3).
- `budgets.month` diễn giải lại theo chu kỳ ở MỌI nơi đọc, kể cả trang Ngân sách hiện có (AD-4).
- Compute-on-write trong cùng `withTx` với ghi giao dịch; tối đa 1 row cảnh báo mới/danh mục/lần ghi; insert idempotent (`INSERT IGNORE`) (AD-7).
- `GET /budgets` (endpoint hiện có) mở rộng thêm `spent/percent/status`; `GET /cycle/summary` mới trả hợp đồng JSON đã ghim cụ thể trong spine (AD-8).
- `PUT /cycle-settings` gộp Lương + savings_goal + cycle_start_day trong một `withTx`; `fixed_costs` là CRUD riêng.
- Không cần migration tool có version, không cần hạ tầng triển khai mới (Deferred).

### UX Design Requirements

UX-DR1: `safe-to-spend-hero` — card lớn, nền `brand-50`, số `display-number` (40px/800), luôn ở đầu Dashboard, full-width mọi breakpoint; icon bút chì mở `cycle-update-sheet`.
UX-DR2: `alert-banner` — card viền trái dày (amber=near, rose=over), nút đóng, tối đa 3 hiển thị cùng lúc (rose ưu tiên trước amber, phần dư gộp "+N khác"), `role="alert"`, không tự ẩn như Toast.
UX-DR3: `budget-status-pill` — 3 trạng thái màu tái dùng brand/amber/rose đã có (không tạo màu mới), thay thế hiển thị nhị phân hiện tại trên trang Ngân sách và Dashboard.
UX-DR4: `anomaly-badge` — màu mới `anomaly` (violet, `anomaly-50`/`anomaly-700`), icon `Info` (không dùng icon ăn mừng/tam giác cảnh báo), gắn trên dòng giao dịch trong bảng Giao dịch hiện có; nhấn mở popover lý do + nút "Đã xem, không vấn đề".
UX-DR5: `cycle-update-sheet` — sheet (mobile)/modal (desktop) mở từ icon bút chì trên hero; trường Lương điền sẵn giá trị chu kỳ trước làm gợi ý (không tự áp dụng); Chi phí cố định/Mục tiêu tiết kiệm hiển thị mặc định; mục "Nâng cao" (ngày bắt đầu chu kỳ) gấp lại mặc định; validate Lương > 0 tại chỗ.
UX-DR6: Voice & Tone — mọi microcopy cảnh báo/gắn cờ nêu sự kiện + con số, không phán xét, không emoji/dấu chấm than tạo cảm giác khẩn cấp, không gamify (bảng Do/Don't trong EXPERIENCE.md).
UX-DR7: State Patterns bắt buộc — hero hiển thị trạng thái "chưa khai báo Lương" thay vì số gây hiểu lầm; không banner nào khi không có cảnh báo (im lặng có chủ đích); không gắn cờ khi thiếu baseline; cold-load skeleton cho hero+stat card.
UX-DR8: Accessibility Floor — xem NFR5; áp dụng cho toàn bộ 4 component mới.

## Epic List

### Epic 1: Biết mỗi ngày còn tiêu được bao nhiêu (Safe-to-spend)
Người dùng khai báo Lương, Chi phí cố định, Mục tiêu tiết kiệm, tuỳ chỉnh ngày bắt đầu chu kỳ nếu cần, và thấy ngay con số "hôm nay còn tiêu được bao nhiêu" trên Dashboard mỗi khi mở app. Epic này cũng hoàn thiện phần refactor backend đang dở dang (repository layer) làm nền cho toàn bộ các repo mới — không tách thành epic kỹ thuật riêng vì không tự nó tạo giá trị cho người dùng.
**FRs covered:** FR1, FR2, FR3, FR4 (+ NFR1 Reliability như một phần nền tảng bắt buộc trước khi thêm repo mới)

### Epic 2: Được cảnh báo trước khi vượt ngân sách
Người dùng nhận cảnh báo ngay trong app khi một danh mục chi tiêu chạm 70%/90%/100% ngân sách, không phải chờ đến cuối chu kỳ mới biết; xem tổng quan trạng thái ngân sách (3 mức) trên Dashboard và trang Ngân sách. Xây trên `CycleWindow()` đã có từ Epic 1.
**FRs covered:** FR5, FR6, FR7

### ~~Epic 3: Tự nhận ra chi tiêu bất thường~~ — [CẮT KHỎI V1 — 2026-09-13]
~~Người dùng thấy một khoản chi bất thường (so với thói quen 3 chu kỳ gần nhất) được gắn cờ ngay trong danh sách giao dịch, có thể xem lý do và đánh dấu đã xem. Chỉ phụ thuộc `CycleWindow()` từ Epic 1 (thêm `PreviousCycles()` cho riêng mình); không phụ thuộc chức năng của Epic 2 — có thể hoạt động độc lập nếu Epic 2 chưa tồn tại, chỉ tình cờ chạm cùng vị trí trong handler giao dịch.~~
~~**FRs covered:** FR8, FR9, FR10~~
Cắt khỏi scope theo quyết định người dùng — dự án một-người-dùng không cần tự động phát hiện chi tiêu bất thường. Chi tiết story giữ nguyên bên dưới làm lịch sử/tham khảo, không triển khai.

### FR Coverage Map

FR1: Epic 1 - Hiển thị Safe-to-spend trên Dashboard
FR2: Epic 1 - Khai báo/cập nhật Lương theo chu kỳ
FR3: Epic 1 - Chi phí cố định + Mục tiêu tiết kiệm
FR4: Epic 1 - Cấu hình ngày bắt đầu chu kỳ
FR5: Epic 2 - Phát cảnh báo khi chạm ngưỡng
FR6: Epic 2 - Không lặp cảnh báo cùng ngưỡng/chu kỳ
FR7: Epic 2 - Tổng quan trạng thái ngân sách 3 mức
~~FR8: Epic 3 - Gắn cờ giao dịch/danh mục bất thường~~ — [CẮT KHỎI V1]
~~FR9: Epic 3 - Đánh dấu đã xem xét cờ bất thường~~ — [CẮT KHỎI V1]
~~FR10: Epic 3 - Không gắn cờ khi thiếu dữ liệu nền~~ — [CẮT KHỎI V1]

## Epic 1: Biết mỗi ngày còn tiêu được bao nhiêu (Safe-to-spend)

Người dùng khai báo Lương, Chi phí cố định, Mục tiêu tiết kiệm, tuỳ chỉnh ngày bắt đầu chu kỳ nếu cần, và thấy ngay con số "hôm nay còn tiêu được bao nhiêu" trên Dashboard mỗi khi mở app.

### Story 1.1: Hoàn thiện & xác thực refactor repository layer hiện có

As a người dùng,
I want các tính năng hiện có (đăng nhập, giao dịch, danh mục, ngân sách) tiếp tục hoạt động chính xác sau khi backend được tái cấu trúc,
So that tôi không mất dữ liệu hay gặp lỗi khi các tính năng mới được xây trên nền này.

**Acceptance Criteria:**

**Given** phần tách repository layer đang dở dang (`repo_budget.go`, `repo_category.go`, `repo_helpers.go`, `repo_transaction.go`, `repo_user.go`, `config.go`, chưa commit)
**When** hoàn thiện việc tách và rà soát lại toàn bộ handler gốc (`api.go`, `auth.go`, `budgets.go`, `categories.go`, `db.go`, `transactions.go`) để dùng đúng các repo mới
**Then** mọi endpoint hiện có (`/auth/*`, `/categories`, `/transactions`, `/budgets`) trả kết quả giống hệt trước refactor cho cùng input

**And** chạy lại toàn bộ kịch bản thao tác thủ công hiện có (đăng ký, đăng nhập, tạo/sửa/xoá giao dịch, danh mục, ngân sách, tài khoản demo) không phát sinh lỗi mới, không mất dữ liệu (NFR1)

**And** không đọc, không sửa, không tham chiếu bất kỳ phần nào của "learning scaffold" (`internal/controller`, `internal/services`, `internal/repo`, `internal/routers`, `internal/server`, `internal/database`, `internal/initialize`, `internal/middlewares`, `global/`, `config/*.yaml`, `cmd/api`, `cmd/server`, `cmd/cli`)

### Story 1.2: Khai báo Lương cho chu kỳ hiện tại

As a người dùng,
I want khai báo hoặc cập nhật Lương cho chu kỳ ngân sách hiện tại,
So that hệ thống có đầu vào đầu tiên để tính Safe-to-spend.

**Acceptance Criteria:**

**Given** tôi chưa khai báo Lương cho chu kỳ hiện tại
**When** tôi mở phần cập nhật chu kỳ
**Then** trường Lương trống, có gợi ý điền sẵn giá trị của chu kỳ trước (không tự áp dụng)

**Given** tôi nhập một giá trị Lương lớn hơn 0 và lưu
**When** lưu thành công
**Then** giá trị được ghi vào bảng `incomes` mới, khoá theo `(user_id, cycle_start_date)` — `cycle_start_date` tính qua hàm `CycleWindow(cycleStartDay, now)` mới được tạo trong `cycle.go`, dùng giá trị mặc định 1 vì chưa có cài đặt tuỳ chỉnh nào tồn tại — story này hoạt động đầy đủ, độc lập với ngày cấu hình

**And** giá trị Lương của chu kỳ này không tự động áp dụng cho chu kỳ kế tiếp — mở lại vào chu kỳ mới sẽ thấy trạng thái "chưa khai báo" trở lại

**Given** tôi nhập Lương ≤ 0 hoặc để trống rồi lưu
**When** lưu
**Then** hệ thống báo lỗi ngay tại trường, không lưu

### Story 1.3: Quản lý Chi phí cố định và Mục tiêu tiết kiệm

As a người dùng,
I want thêm, sửa, xoá các khoản Chi phí cố định và đặt Mục tiêu tiết kiệm,
So that Safe-to-spend trừ đúng các khoản này trước khi tính số tiền còn lại mỗi ngày.

**Acceptance Criteria:**

**Given** tôi chưa có khoản chi phí cố định nào
**When** tôi thêm một khoản mới (tên + số tiền)
**Then** khoản đó xuất hiện trong danh sách và cộng vào tổng Chi phí cố định dùng cho Safe-to-spend

**Given** một khoản chi phí cố định đã có
**When** tôi sửa số tiền hoặc xoá khoản đó
**Then** tổng Chi phí cố định cập nhật ngay, không cần tải lại trang

**Given** tôi đặt hoặc sửa Mục tiêu tiết kiệm
**When** lưu
**Then** giá trị được lưu vào bảng `user_settings` mới (1:1 với user, cột `savings_goal` mặc định 0, cột `cycle_start_day` mặc định 1) và trừ vào công thức Safe-to-spend — story này chỉ cần cột `savings_goal`; `cycle_start_day` tồn tại với giá trị mặc định nhưng chưa có UI chỉnh sửa

**And** bảng `fixed_costs` mới (danh sách đứng, không snapshot theo chu kỳ) được thêm qua `Migrate()` hiện có

### Story 1.4: Cấu hình ngày bắt đầu chu kỳ ngân sách

As a người dùng,
I want chọn ngày trong tháng mà chu kỳ ngân sách của tôi bắt đầu,
So that Safe-to-spend tính đúng theo chu kỳ lương thực tế của tôi, không cố định theo ngày 1 dương lịch.

**Acceptance Criteria:**

**Given** tôi mở mục "Nâng cao" (gấp lại mặc định) trong sheet cập nhật chu kỳ
**When** tôi chọn một ngày khác ngày 1 và lưu
**Then** `user_settings.cycle_start_day` cập nhật, và `CycleWindow()` trong `cycle.go` được sửa để đọc giá trị này thay vì hằng số 1 cố định (khép lại phần còn để trống ở Story 1.2)

**Given** `cycle_start_day` vừa đổi
**When** tôi xem lại Safe-to-spend
**Then** số ngày còn lại và chu kỳ hiện tại được tính lại ngay theo giá trị mới — đây là hiệu lực tức thời có chủ đích, các row `incomes` đã ghi theo chu kỳ cũ trở thành lịch sử không còn được truy vấn hiện tại trỏ tới (không bị xoá)

**And** `PUT /cycle-settings` gộp cả Lương (Story 1.2) và cài đặt (savings_goal, cycle_start_day) trong một `h.withTx`, khớp với hành động "Lưu" duy nhất của sheet cập nhật chu kỳ

### Story 1.5: Hiển thị Safe-to-spend trên Dashboard

As a người dùng,
I want thấy ngay số tiền còn có thể tiêu hôm nay khi mở app,
So that tôi biết giới hạn chi tiêu mà không cần tự tính tay.

**Acceptance Criteria:**

**Given** tôi đã khai báo Lương, Chi phí cố định, Mục tiêu tiết kiệm cho chu kỳ hiện tại
**When** tôi mở Dashboard
**Then** `safe-to-spend-hero` hiển thị đầu tiên, full-width, với số Safe-to-spend (kiểu `display-number`) và số ngày còn lại trong chu kỳ — lấy từ `GET /cycle/summary` mới, tính lại mỗi lần tải, không cache cũ

**Given** tôi chưa khai báo Lương cho chu kỳ hiện tại
**When** tôi mở Dashboard
**Then** hero hiển thị trạng thái "Cập nhật Lương để xem Safe-to-spend" thay vì một con số gây hiểu lầm (`income: null` phân biệt với `income: 0`)

**Given** tôi vừa lưu một giao dịch chi tiêu mới
**When** tôi quay lại Dashboard
**Then** Safe-to-spend phản ánh đúng số dư còn lại sau giao dịch đó

**And** `safeToSpend` là số nguyên có dấu — âm nghĩa là đã chi vượt, không bao giờ bị clamp về 0

## Epic 2: Được cảnh báo trước khi vượt ngân sách

Người dùng nhận cảnh báo ngay trong app khi một danh mục chi tiêu chạm 70%/90%/100% ngân sách, không phải chờ đến cuối chu kỳ mới biết; xem tổng quan trạng thái ngân sách (3 mức) trên Dashboard và trang Ngân sách.

### Story 2.1: Phát cảnh báo khi giao dịch vượt ngưỡng ngân sách

As a người dùng,
I want nhận cảnh báo ngay khi một danh mục chi tiêu chạm 70%, 90%, hoặc 100% ngân sách,
So that tôi biết kịp thời để điều chỉnh, không phải đợi đến cuối chu kỳ.

**Acceptance Criteria:**

**Given** danh mục "Ăn uống" đã có ngân sách cho chu kỳ hiện tại và đang ở dưới 70%
**When** tôi lưu một giao dịch đẩy % đã dùng qua 70%, 90%, hoặc 100%
**Then** một row `budget_alert_state` mới được ghi (bảng mới, qua `Migrate()`) trong **cùng một `h.withTx`** với việc lưu giao dịch — không phải một ghi riêng sau khi commit

**Given** danh mục đã có cảnh báo cho ngưỡng 90% trong chu kỳ hiện tại
**When** tôi lưu thêm giao dịch vẫn trên 90% nhưng chưa tới 100%
**Then** không có row cảnh báo mới nào được ghi thêm cho ngưỡng 90%

**Given** một giao dịch lớn đẩy % từ 40% lên 105% trong một lần lưu
**When** giao dịch được ghi
**Then** chỉ đúng một row cảnh báo mới được ghi — cho ngưỡng cao nhất mới vượt qua (100%), không phải ba row cho 70/90/100

**And** insert dùng `INSERT IGNORE`/`ON DUPLICATE KEY UPDATE` — hai request đồng thời cùng kích hoạt một ngưỡng không gây lỗi 500

### Story 2.2: Xem và đóng cảnh báo ngân sách trên Dashboard

As a người dùng,
I want thấy danh sách cảnh báo đang hoạt động trên Dashboard và đóng từng cảnh báo,
So that tôi không bị làm phiền lặp lại sau khi đã ghi nhận.

**Acceptance Criteria:**

**Given** có cảnh báo chưa đóng cho chu kỳ hiện tại
**When** tôi mở Dashboard
**Then** tôi thấy `alert-banner` tương ứng ngay dưới `safe-to-spend-hero` (tối đa 3, "over" ưu tiên trước "near", phần dư gộp "+N cảnh báo khác") — dữ liệu từ `GET /cycle/summary.activeAlerts`

**Given** tôi nhấn nút đóng trên một banner
**When** đóng
**Then** `dismissed_at` được ghi qua `POST /alerts/:categoryId/:threshold/dismiss`, banner biến mất và không hiện lại trong chu kỳ này

**Given** không có cảnh báo nào đang hoạt động trong chu kỳ hiện tại
**When** tôi mở Dashboard
**Then** không có banner nào hiển thị — im lặng có chủ đích, không có trạng thái "trống" hiển thị ra

**Given** một cảnh báo từ chu kỳ trước chưa từng bị đóng
**When** tôi xem Dashboard ở chu kỳ hiện tại
**Then** cảnh báo đó KHÔNG hiển thị — truy vấn luôn lọc thêm `cycle_start_date` bằng chu kỳ hiện tại, không chỉ `dismissed_at IS NULL`

**And** mỗi `alert-banner` có `role="alert"` (trình đọc màn hình thông báo ngay khi xuất hiện) và nút đóng có `aria-label` nêu rõ danh mục (NFR5/UX-DR8)

### Story 2.3: Xem trạng thái 3 mức ngân sách theo danh mục

As a người dùng,
I want thấy trạng thái (trong hạn mức / gần vượt / đã vượt) của mỗi danh mục có ngân sách,
So that tôi chủ động xem lại bất cứ lúc nào, không chỉ khi có cảnh báo mới.

**Acceptance Criteria:**

**Given** tôi có ngân sách cho nhiều danh mục trong chu kỳ hiện tại
**When** tôi mở trang Ngân sách
**Then** mỗi danh mục hiển thị đúng một trong ba trạng thái (`within` <70%, `near` 70–99%, `over` ≥100%) qua `budget-status-pill`, dùng trường `spent/percent/status` mới trả về từ `GET /budgets` (mở rộng, không phải trang tự tính lại)

**Given** trang Ngân sách và Dashboard cùng hiển thị trạng thái một danh mục
**When** tôi so sánh
**Then** hai nơi luôn khớp nhau tuyệt đối — cùng dùng chung hàm tính trong `cycle.go`

**Given** `cycle_start_day` khác ngày 1 (từ Epic 1, Story 1.4)
**When** tôi xem trang Ngân sách
**Then** `budgets.month` được diễn giải theo tháng chứa `cycle_start_date` của chu kỳ hiện tại, không theo tháng dương lịch hệ thống — chỉ có một cách hiểu "kỳ hiện tại" trong toàn app

## Epic 3: Tự nhận ra chi tiêu bất thường

Người dùng thấy một khoản chi bất thường (so với thói quen 3 chu kỳ gần nhất) được gắn cờ ngay trong danh sách giao dịch, có thể xem lý do và đánh dấu đã xem.

### Story 3.1: Gắn cờ giao dịch bất thường khi lưu

As a người dùng,
I want một khoản chi bất thường so với thói quen của tôi được tự động gắn cờ khi tôi lưu giao dịch,
So that tôi được nhắc xem lại mà không cần tự phân tích số liệu.

**Acceptance Criteria:**

**Given** danh mục đã có ít nhất 1 chu kỳ dữ liệu đầy đủ trước đó
**When** tôi lưu một giao dịch khiến tổng chi của danh mục trong chu kỳ hiện tại vượt quá 1.5 lần trung bình 3 chu kỳ gần nhất (tính qua `PreviousCycles()` mới thêm vào `cycle.go`)
**Then** đúng giao dịch vừa lưu — giao dịch khiến tổng vượt ngưỡng lần đầu tiên — được gắn cờ trong bảng `transaction_flags` mới (PK = transaction_id, FK CASCADE về `transactions`), kèm lý do cụ thể

**Given** danh mục mới tạo hoặc chưa đủ 1 chu kỳ dữ liệu lịch sử
**When** tôi lưu giao dịch bất kỳ trong danh mục đó
**Then** không giao dịch nào bị gắn cờ, bất kể số tiền

**Given** một giao dịch đã có row `transaction_flags` (dù đã xem hay chưa)
**When** tôi sửa ghi chú hoặc ngày của giao dịch đó
**Then** hệ thống không đánh giá lại và không gỡ cờ — chỉ giao dịch Create, hoặc Update khi chưa từng có row flag, mới chạy phép kiểm tra bất thường

**And** `anomaly-badge` không chỉ dựa vào màu sắc để truyền đạt ý nghĩa — luôn kèm icon (`Info`) và/hoặc nhãn chữ, đúng nguyên tắc `.chip` hiện có (NFR5/UX-DR8)

### Story 3.2: Xem lý do và đánh dấu đã xem cờ bất thường

As a người dùng,
I want xem lý do một giao dịch bị gắn cờ và đánh dấu đã xem,
So that cờ không còn làm phân tâm sau khi tôi đã xác nhận đây không phải vấn đề.

**Acceptance Criteria:**

**Given** một giao dịch có `anomaly-badge` trong danh sách Giao dịch
**When** tôi nhấn vào badge
**Then** popover hiện lý do ngắn gọn (ví dụ "cao hơn 1.5 lần trung bình 3 chu kỳ gần nhất") — trường `flagged`/`flagReason` lấy từ `GET /transactions` mở rộng

**Given** popover lý do đang mở
**When** tôi nhấn "Đã xem, không vấn đề"
**Then** `acknowledged_at` được ghi qua `POST /transactions/:id/ack-flag`, badge không còn hiển thị nổi bật cho giao dịch đó — kể cả sau khi tải lại trang

**Given** tôi đóng popover mà không nhấn nút
**When** tôi quay lại xem sau
**Then** badge vẫn còn nguyên, không tự biến mất

**And** trường JSON `flagged` = `reason IS NOT NULL`, trường `flagAcknowledged` = `acknowledged_at IS NOT NULL` — không bao giờ suy ra trường này từ trường kia
