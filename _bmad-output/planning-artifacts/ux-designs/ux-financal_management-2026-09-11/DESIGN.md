---
name: HL Personal Finance
status: final
updated: 2026-09-11
description: App quản lý tài chính cá nhân chủ động. Tailwind CSS v4 tuỳ biến qua @theme trên React 19 + Vite; DESIGN.md này chỉ đặc tả phần delta thêm mới cho ba năng lực của bản cải tổ (Safe-to-spend, cảnh báo ngân sách, gắn cờ bất thường) trên nền hệ thống hiện có.
colors:
  # Toàn bộ token bên dưới đã tồn tại trong frontend/src/index.css (Tailwind @theme) — liệt kê lại ở đây để DESIGN.md là nguồn tham chiếu đầy đủ, không phải vì thay đổi giá trị.
  brand-50: '#ecfdf5'
  brand-500: '#10b981'
  brand-600: '#059669'
  brand-700: '#047857'
  ink-50: '#f8fafc'
  ink-100: '#f1f5f9'
  ink-400: '#94a3b8'
  ink-500: '#64748b'
  ink-800: '#1e293b'
  ink-900: '#0f172a'
  # Delta thật sự cho bản cải tổ: một accent hoàn toàn mới cho "gắn cờ bất thường",
  # vì brand/rose/sky/amber đã có ý nghĩa cố định khác trong hệ thống hiện tại.
  anomaly-50: '#f5f3ff'
  anomaly-500: '#8b5cf6'
  anomaly-700: '#6d28d9'
typography:
  # Kế thừa nguyên vẹn font-sans (Inter) và ramp hiện có. Chỉ thêm một vai trò mới (xem lý do ở phần Typography bên dưới).
  display-number:
    fontFamily: '{typography.font-sans}'
    fontSize: 40px
    fontWeight: '800'
    lineHeight: '1.1'
    letterSpacing: -0.01em
rounded:
  # Kế thừa nguyên vẹn — không đổi.
  md: 0.75rem
  xl: 0.75rem
  '2xl': 1rem
  full: 9999px
spacing:
  # Kế thừa thang spacing Tailwind mặc định (4, 8, 12, 16, 20, 24, 32...) — không override, không token nào ở đây cần định nghĩa riêng.
  note: 'inherits Tailwind default scale, no overrides'
components:
  safe-to-spend-hero:
    background: '{colors.brand-50}'
    border: '{colors.brand-500}/20'
    radius: '{rounded.2xl}'
    numberStyle: '{typography.display-number}'
  budget-status-pill:
    within: 'bg-brand-50 text-brand-700'
    near: 'bg-amber-50 text-amber-700'
    over: 'bg-rose-50 text-rose-700'
  anomaly-badge:
    background: '{colors.anomaly-50}'
    foreground: '{colors.anomaly-700}'
    radius: '{rounded.full}'
  alert-banner:
    near:
      background: 'bg-amber-50'
      border: 'border-amber-200'
      foreground: 'text-amber-800'
    over:
      background: 'bg-rose-50'
      border: 'border-rose-200'
      foreground: 'text-rose-800'
  cycle-update-sheet:
    background: 'bg-white'
    radius: '{rounded.2xl}'
    note: 'Không có token mới — tái sử dụng .input/.label/.btn-primary hiện có; chrome sheet/modal kế thừa mặc định trình duyệt + Tailwind.'
---

## Brand & Style

HL Personal Finance là một công cụ, không phải một sản phẩm tiêu dùng cần "gây ấn tượng". Phong cách hiện có (Tailwind, thẻ bo góc `rounded-2xl`, màu xanh lá `brand` làm điểm nhấn, nền trung tính `ink`) đã đúng tinh thần: sạch, nghiêm túc vừa đủ cho một app tiền bạc, không nặng nề như một app ngân hàng doanh nghiệp. Bản cải tổ **không làm lại nhận diện thương hiệu** — brief đã nói rõ đây không phải dự án portfolio cần gây ấn tượng thị giác. Toàn bộ token màu, chữ, bo góc, component hiện có được giữ nguyên; DESIGN.md này chỉ thêm những gì ba năng lực mới thực sự cần và hệ thống cũ chưa có.

Nguyên tắc cho phần thêm mới: **im lặng khi không có gì bất thường, nổi bật đúng lúc cần chú ý**. Safe-to-spend luôn hiện diện nhưng điềm tĩnh (dùng chính màu brand nền nhạt, không phải một màu báo động); cảnh báo ngân sách dùng đúng ngôn ngữ màu cảnh báo đã có (amber = sắp, rose = đã vượt); gắn cờ bất thường dùng một màu hoàn toàn mới — xem Colors → `anomaly`.

## Colors

- **`brand` (xanh lá, đã có)** — màu tích cực: số dư dương, thu nhập, trạng thái ngân sách "trong hạn mức", nút hành động chính. Safe-to-spend hero dùng `brand-50` làm nền — một sắc thái rất nhạt để con số lớn không bị nền cạnh tranh, nhưng vẫn ngầm nói "đây là điều tốt cần theo dõi", không phải một cảnh báo.
- **`amber` (đã có, tái sử dụng có chủ đích)** — trước đây chỉ dùng cho một stat card (`budgetsTracked`); bản cải tổ mở rộng ý nghĩa của nó thành trạng thái ngân sách "gần vượt" (70–99%, theo FR-7 của PRD) và banner cảnh báo mức "sắp vượt". Không tạo màu warning mới vì amber đã đúng vai trò này trong hệ màu Tailwind.
- **`rose` (đã có)** — giữ nguyên vai trò cảnh báo mạnh nhất: chi tiêu, ngân sách "đã vượt", banner cảnh báo mức vượt hoàn toàn. Không đổi.
- **`anomaly` (violet, MỚI — `[ASSUMPTION]`)** — màu duy nhất thêm mới cho toàn bộ bản cải tổ, dùng riêng cho Gắn cờ bất thường. Cố tình không dùng rose/amber vì PRD (§4.3) nói rõ đây không phải cảnh báo lỗi/bảo mật mà là gợi ý tự xem lại — violet đọc là "đáng chú ý", không phải "có gì sai". Cặp nền/chữ `anomaly-50`/`anomaly-700` giữ cùng tỉ lệ tương phản với các cặp `*-50`/`*-700` khác trong hệ thống (đã đạt AA cho văn bản thường).
- **`ink` (trung tính, đã có)** — không đổi, dùng cho toàn bộ text/border/nền phụ như hiện tại.

**Không dùng:** không dùng gradient (các quy tắc khác về `anomaly`/`display-number` xem Do's and Don'ts).

## Typography

Kế thừa nguyên vẹn `font-sans: Inter` và toàn bộ ramp cỡ chữ hiện có (`text-sm`, `text-2xl font-bold` cho StatCard, v.v.) cùng lớp tiện ích `.tnum` (tabular nums) cho mọi cột số tiền — không đổi.

Thêm một vai trò mới duy nhất: **`display-number`** (40px, weight 800, tabular nums kế thừa từ `.tnum`) — dành riêng cho con số Safe-to-spend trên Dashboard, con số duy nhất người dùng cần thấy trong 2 giây đầu mở app. Đây là con số lớn hơn `text-2xl` — cỡ hiện đang là mức lớn nhất trong app (StatCard). Quy tắc sử dụng: xem Do's and Don'ts.

## Layout & Spacing

Kế thừa nguyên vẹn thang spacing Tailwind mặc định và bố cục hiện có của Dashboard (`space-y-6`, grid `sm:grid-cols-2 xl:grid-cols-4` cho stat cards). Thay đổi bố cục duy nhất: một hàng mới ở **trên cùng** Dashboard, phía trên hàng stat card hiện tại, dành cho `safe-to-spend-hero` + (khi có) `alert-banner` — xem `EXPERIENCE.md → Information Architecture`. Hero luôn full-width ở mọi breakpoint (không chia cột — đây là con số duy nhất cần chiếm toàn bộ chú ý). Các `alert-banner` xếp chồng theo chiều dọc (`space-y-2`) ngay dưới hero, full-width như nhau ở mọi kích thước màn hình — không chuyển sang lưới nhiều cột kể cả trên desktop, vì đây là danh sách cần đọc tuần tự (rose trước amber), không phải nội dung duyệt song song.

## Elevation & Depth

Kế thừa nguyên vẹn — `.card` hiện tại dùng `shadow-sm`, không có hệ thống elevation nhiều lớp. Safe-to-spend hero và alert banner đều dùng cùng mức nông này; không thêm shadow nặng hơn để tạo phân cấp — phân cấp thị giác đến từ kích thước chữ (`display-number`) và vị trí (trên cùng), không phải từ đổ bóng.

## Shapes

Kế thừa nguyên vẹn: `rounded-2xl` cho card/hero, `rounded-xl` cho button/input/icon-box, `rounded-full` cho chip/pill/badge (bao gồm `budget-status-pill` và `anomaly-badge` mới).

## Components

Component mới, xây trên các lớp `.card`/`.chip` đã có trong `frontend/src/index.css`:

- **`safe-to-spend-hero`** — biến thể lớn hơn của `.card`, nền `brand-50`, viền `brand-500/20`. Chứa: nhãn nhỏ ("Hôm nay còn tiêu được"), con số lớn (`display-number`, `.tnum`), dòng phụ nhỏ (số ngày còn lại trong chu kỳ). Xuất hiện đúng một lần, trên cùng Dashboard.
- **`budget-status-pill`** — biến thể của `.chip` đã có, ba trạng thái màu (within=brand, near=amber, over=rose) ánh xạ đúng ranh giới 70%/100% của FR-7. Thay thế cách hiển thị nhị phân hiện tại (chỉ brand/rose) trên thanh tiến trình ngân sách.
- **`alert-banner`** — mới, dạng card viền trái dày (`border-l-4`) màu amber hoặc rose tương ứng mức ngưỡng, có nút đóng (dismiss). Xuất hiện thành danh sách ngay dưới `safe-to-spend-hero` khi có cảnh báo chưa xem trong chu kỳ hiện tại (xem EXPERIENCE.md → State Patterns).
- **`anomaly-badge`** — biến thể của `.chip`, nền `anomaly-50`, chữ `anomaly-700`, icon lucide `Info` (không dùng `Sparkles` — quá gợi cảm giác "thành tích/ăn mừng", trái với nguyên tắc "không gamify" của brief; cũng không dùng icon tam giác cảnh báo vì đó là ngôn ngữ thị giác của amber/rose). Gắn cạnh dòng giao dịch bị gắn cờ trong bảng Giao dịch hiện có.
- **`cycle-update-sheet`** — sheet (mobile) / modal (desktop) mở từ icon bút chì trên `safe-to-spend-hero`. Không có token hình ảnh mới — dựng hoàn toàn từ `.input`, `.label`, `.btn-primary`/`.btn-ghost` đã có; mục gấp lại dùng một link text nhỏ `text-ink-500`. Hành vi chi tiết: xem `EXPERIENCE.md → Component Patterns`.

**Không đổi:** `.btn-primary/.btn-ghost/.btn-outline/.btn-icon`, `.input`, `.label`, `StatCard`, `CategoryIcon`, bảng Giao dịch (cấu trúc cột/hàng), biểu đồ Recharts trên Dashboard — toàn bộ giữ nguyên như hiện tại.

## Do's and Don'ts

| Làm | Không làm |
|---|---|
| Tái sử dụng brand/amber/rose đúng ý nghĩa đã có (tích cực/gần vượt/đã vượt) | Tạo thêm màu cảnh báo mới ngoài amber/rose |
| Dùng `anomaly` (violet) chỉ cho gắn cờ bất thường | Dùng violet cho bất kỳ mục đích trang trí hay trạng thái nào khác |
| Giữ `display-number` chỉ cho Safe-to-spend | Phóng to con số khác (thu nhập, chi tiêu...) lên cùng cỡ — sẽ làm loãng phân cấp |
| Đặt `safe-to-spend-hero` cố định ở vị trí đầu tiên trên Dashboard | Di chuyển hero xuống dưới stat card, hoặc nhân bản nó ở trang khác |
| Alert banner có nút đóng rõ ràng, không tự động biến mất như Toast | Dùng Toast (hiện có, dùng cho create/update/delete) cho cảnh báo ngân sách — hai loại thông báo có vòng đời khác nhau |
