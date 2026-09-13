---
name: HL Personal Finance
status: final
sources:
  - {planning_artifacts}/prds/prd-financal_management-2026-09-11/prd.md
  - {planning_artifacts}/briefs/brief-financal_management-2026-09-11/brief.md
updated: 2026-09-11
---

# HL Personal Finance — Experience Spine

> Web app quản lý tài chính cá nhân, React 19 + Tailwind v4, mobile-first responsive. Bản cải tổ thêm ba năng lực chủ động (Safe-to-spend, Cảnh báo ngân sách, Gắn cờ bất thường) lên trên các surface hiện có (Dashboard, Giao dịch, Ngân sách). `DESIGN.md` là nguồn tham chiếu hình ảnh; file này là nguồn tham chiếu hành vi. Khi có mâu thuẫn với bất kỳ mock/wireframe nào, hai file này thắng.

## Foundation

Web responsive, mobile-first `[ASSUMPTION]`: ba User Journey trong PRD đều mô tả cảnh mở app trên điện thoại ngay sau một sự kiện tài chính (nhận lương, vừa chi tiêu) — hành vi kiểm tra nhanh, nhiều lần trong ngày, đúng bản chất một app di động dùng qua trình duyệt/PWA hơn là một công cụ desktop. Không xây app native (kế thừa Non-Goals của PRD). Single-tenant: mỗi người dùng chỉ thấy dữ liệu của chính mình (Giai đoạn 2 sổ chung gia đình sẽ mở rộng mô hình này sau, không nằm trong phạm vi file này). Token màu/kiểu chữ trong file này được tham chiếu bằng `{path.to.token}` trỏ vào `DESIGN.md`.

## Information Architecture

| Surface | Vào từ | Mục đích | Thay đổi so với hiện tại |
|---|---|---|---|
| Dashboard | Đăng nhập / nav chính | Tổng quan: Safe-to-spend, cảnh báo, biểu đồ dòng tiền, trạng thái ngân sách, giao dịch gần đây | **Thêm** hàng trên cùng: `safe-to-spend-hero` + `alert-banner` list (xem Component Patterns) |
| Giao dịch | Nav chính | Xem/thêm/sửa/xoá giao dịch | **Thêm** `anomaly-badge` trên dòng bị gắn cờ + hành động "đã xem, không vấn đề" |
| Ngân sách | Nav chính | Xem/đặt hạn mức theo danh mục | **Thêm** `budget-status-pill` ba trạng thái (thay hiển thị nhị phân cũ) |
| Cập nhật chu kỳ *(mới)* | Icon bút chì trên `safe-to-spend-hero` | Khai báo/sửa Lương, Chi phí cố định, Mục tiêu tiết kiệm, ngày bắt đầu Chu kỳ ngân sách | **Mới** — realizes FR-2, FR-3, FR-4. Hành vi: xem Component Patterns → `cycle-update-sheet`; luồng đầy đủ: Key Flow 1. |
| Đăng nhập / Đăng ký | — | Không đổi | Không đổi |

Không có nav item mới — "Cập nhật chu kỳ" là một sheet/modal mở từ Dashboard, không chiếm một mục điều hướng riêng vì tần suất dùng thấp (vài lần/tháng).

→ Tham chiếu bố cục: `mockups/dashboard.html`. Spine (file này + DESIGN.md) thắng khi có mâu thuẫn.

## Voice and Tone

Microcopy — giọng thương hiệu ở `DESIGN.md → Brand & Style`. Nguyên tắc: **nói sự thật bằng số liệu, không phán xét, không cổ vũ giả tạo**. Đây là công cụ giúp người dùng tự nhìn rõ hành vi chi tiêu của chính mình, không phải một huấn luyện viên "gamify".

| Nên | Không nên |
|---|---|
| "Hôm nay còn tiêu được 185.000đ" | "Bạn đang làm rất tốt! 🎉" |
| "Ăn uống: đã dùng 92% ngân sách, còn 6 ngày" | "Cẩn thận, bạn sắp hết tiền rồi đó!" |
| "Cao hơn 1.5 lần trung bình 3 chu kỳ gần nhất" | "Chi tiêu bất thường! Có gì đó không ổn." |
| "Đã xem, không vấn đề" (hành động trung tính) | "Đánh dấu đã sửa" (ngụ ý đây là lỗi cần sửa) |
| Cùng một giọng cho mọi cảnh báo — nêu sự kiện + con số | Thêm emoji/dấu chấm than để tạo cảm giác khẩn cấp |

## Component Patterns

Hành vi — đặc tả hình ảnh ở `DESIGN.md → Components`.

| Component | Dùng ở | Quy tắc hành vi |
|---|---|---|
| `safe-to-spend-hero` | Dashboard | Luôn hiển thị đầu tiên, không thể ẩn. Nếu Lương/Chi phí cố định/Mục tiêu tiết kiệm chưa được xác nhận cho chu kỳ hiện tại, hero hiển thị trạng thái "chưa đầy đủ" (xem State Patterns) thay vì một con số có thể sai. Nhấn icon bút chì mở modal/sheet "Cập nhật chu kỳ". Realizes FR-1. |
| `alert-banner` | Dashboard, ngay dưới hero | Một banner cho mỗi (Danh mục, Ngưỡng) đã kích hoạt và **chưa bị đóng** trong chu kỳ hiện tại (FR-5, FR-6). Đóng một banner không xoá cảnh báo khỏi hệ thống — chỉ ẩn khỏi Dashboard; trạng thái "đã cảnh báo" của FR-6 vẫn được giữ để không hiện lại. `[ASSUMPTION]`: tối đa 3 banner cùng lúc. "Over" (rose) luôn ưu tiên hiển thị trước "near" (amber); phần dư gộp thành "+N cảnh báo khác, xem Ngân sách". Đây là quyết định UX mà PRD không quy định con số cụ thể — cần xem lại ngưỡng 3 khi có dữ liệu ngân sách thật. |
| `budget-status-pill` | Ngân sách, Dashboard (tóm tắt) | Ba trạng thái ánh xạ đúng FR-7 (within <70%, near 70–99%, over ≥100%). Không có trạng thái thứ tư; 0% chi tiêu vẫn là "within". |
| `anomaly-badge` | Giao dịch (mỗi dòng bị gắn cờ) | Nhấn badge mở popover/sheet nhỏ hiển thị lý do (FR-8 consequence) + nút "Đã xem, không vấn đề" (FR-9). Sau khi bấm, badge biến mất khỏi dòng đó vĩnh viễn cho giao dịch đó (không hiện lại dù mở lại trang). |
| `cycle-update-sheet` | Mở từ icon bút chì trên `safe-to-spend-hero` | Trường Lương điền sẵn giá trị chu kỳ trước làm gợi ý, không tự áp dụng — người dùng luôn phải xác nhận (realizes FR-2). Chi phí cố định/Mục tiêu tiết kiệm hiển thị mặc định, sửa trực tiếp (FR-3). Mục "Nâng cao" (ngày bắt đầu Chu kỳ, FR-4) gấp lại mặc định. Validate Lương > 0 tại chỗ, không cho Lưu nếu sai (xem State Patterns). Sheet trên mobile (`< md`), modal căn giữa trên desktop (`≥ md`). |
| Bảng Giao dịch (đã có) | Giao dịch | Không đổi cấu trúc — chỉ thêm cột/indicator cho `anomaly-badge` ở đầu dòng, trước cột Danh mục. |

## State Patterns

| Trạng thái | Surface | Cách xử lý |
|---|---|---|
| Chưa khai báo Lương cho chu kỳ mới | Dashboard (`safe-to-spend-hero`) | Thay vì một con số, hero hiển thị: "Cập nhật Lương để xem Safe-to-spend" + nút mở modal "Cập nhật chu kỳ" ngay trong hero — không chặn phần còn lại của Dashboard. Realizes UJ-1 edge case, FR-2. |
| Đã khai báo Lương nhưng chưa xác nhận Chi phí cố định/Mục tiêu tiết kiệm | Dashboard (`safe-to-spend-hero`) | Hero vẫn hiển thị con số (dùng giá trị 0 cho phần chưa khai báo, theo FR-3) kèm dòng phụ nhỏ màu `ink-500`: "Chưa cập nhật chi phí cố định — số có thể chưa chính xác." Không dùng banner cảnh báo (đây là gợi ý, không phải cảnh báo ngân sách). |
| Không có cảnh báo/gắn cờ nào đang hoạt động | Dashboard | Không hiển thị `alert-banner` nào — im lặng hoàn toàn (đúng nguyên tắc DESIGN.md "im lặng khi không có gì bất thường"). |
| Danh mục/Giao dịch mới, chưa đủ dữ liệu nền (FR-10) | Giao dịch | Không hiển thị `anomaly-badge` cho danh mục đó; không có trạng thái "đang học" hiển thị ra ngoài — im lặng cho tới khi đủ dữ liệu. |
| Tải lần đầu (cold load) | Dashboard | Khung skeleton cho hero + stat card, giữ đúng layout để tránh nhảy bố cục khi dữ liệu về. Ngân sách và Giao dịch **kế thừa nguyên vẹn** hành vi tải hiện có của hai trang đó — không đổi. |
| Nhập sai trong `cycle-update-sheet` (Lương ≤ 0 hoặc bỏ trống) | `cycle-update-sheet` | Báo lỗi ngay tại trường ("Lương phải lớn hơn 0"), không đóng sheet, giữ nguyên các trường khác đã nhập. Realizes FR-2 (Consequences). |
| Lỗi tải dữ liệu ngân sách/giao dịch | Dashboard, Ngân sách, Giao dịch | Toast lỗi hiện có (đã có sẵn theo README) — không dùng `alert-banner` cho lỗi kỹ thuật, chỉ dùng cho cảnh báo ngân sách nghiệp vụ. |

## Interaction Primitives

- **Chạm/nhấn là chính** — đây là app mobile-first, không có phím tắt bàn phím nào là bắt buộc (khác các sản phẩm desktop-first).
- Đóng `alert-banner`: nhấn nút X trên banner — không có "hoàn tác", banner không hiện lại trong chu kỳ này (đã được ghi nhận theo FR-6).
- Mở "Cập nhật chu kỳ": nhấn icon bút chì trên `safe-to-spend-hero`. Đóng bằng nút Lưu hoặc vuốt xuống (sheet trên mobile) / nhấn ra ngoài (modal trên desktop).
- Đánh dấu `anomaly-badge` đã xem: nhấn badge → popover → nhấn "Đã xem, không vấn đề". Không có thao tác vuốt ẩn ngoài luồng này để tránh vô tình bỏ qua một khoản chi cần xem lại.
- **Cấm:** tự động ẩn `alert-banner` sau một khoảng thời gian (khác Toast) — người dùng phải chủ động đóng, vì đây là thông tin cần hành động, không phải xác nhận thao tác vừa làm.

## Accessibility Floor

Hành vi — độ tương phản màu sắc nằm ở `DESIGN.md`.

- Toàn bộ màu mới (`anomaly`, ba trạng thái `budget-status-pill`) đạt tối thiểu WCAG AA cho text-trên-nền theo đúng cặp nhạt/đậm đã dùng trong hệ thống hiện có (`*-50` nền, `*-700` chữ).
- `anomaly-badge` và `budget-status-pill` không dựa vào màu sắc một mình để truyền đạt ý nghĩa — luôn kèm icon và/hoặc nhãn chữ (kế thừa nguyên tắc `.chip` hiện có).
- `alert-banner` có `role="alert"` để trình đọc màn hình thông báo ngay khi xuất hiện; nút đóng có `aria-label` rõ ràng ("Đóng cảnh báo {danh mục}").
- Giữ nguyên chuẩn touch-target tối thiểu 40×40px đã áp dụng cho `.btn-icon` hiện có, áp dụng cho icon bút chì trên hero và nút đóng banner.
- Tôn trọng `prefers-reduced-motion` đã có trong `index.css` — banner mới xuất hiện không dùng animation trượt/nảy, chỉ fade nhẹ kế thừa `fadeIn` hiện có.

## Key Flows

### Flow 1 — Minh cập nhật chu kỳ mới sau khi lương về (UJ-1)

1. Minh mở app, Dashboard tải. `safe-to-spend-hero` phát hiện chu kỳ mới đã bắt đầu nhưng Lương chưa được xác nhận cho chu kỳ này.
2. Hero hiển thị trạng thái "Cập nhật Lương để xem Safe-to-spend" thay vì một con số.
3. Minh nhấn nút trong hero, sheet "Cập nhật chu kỳ" mở từ dưới lên (mobile). Trường Lương được điền sẵn giá trị chu kỳ trước làm gợi ý, không tự áp dụng.
4. Minh nhập Lương thật của kỳ này, xem lại Chi phí cố định (giữ nguyên hoặc sửa), xác nhận Mục tiêu tiết kiệm, nhấn Lưu.
5. **Climax:** Sheet đóng lại, Dashboard tự cập nhật ngay tại chỗ — `safe-to-spend-hero` giờ hiển thị con số `display-number` lớn, kèm số ngày còn lại trong chu kỳ. Minh không cần điều hướng đi đâu khác.

Thất bại: Minh nhập Lương âm hoặc bỏ trống → trường báo lỗi ngay tại chỗ ("Lương phải lớn hơn 0"), sheet không đóng, không mất dữ liệu các trường khác đã nhập.

### Flow 2 — Minh nhận cảnh báo ngân sách giữa chu kỳ (UJ-2)

1. Minh lưu một giao dịch ăn uống mới trong màn hình Giao dịch.
2. Toast xác nhận lưu thành công hiện ra như bình thường (không đổi).
3. Hệ thống tính lại % ngân sách danh mục "Ăn uống", phát hiện vừa vượt 90%.
4. Minh điều hướng về Dashboard (hoặc app tự làm mới nếu đang mở sẵn): một `alert-banner` màu amber xuất hiện ngay dưới `safe-to-spend-hero`: "Ăn uống: đã dùng 92% ngân sách, còn 6 ngày."
5. **Climax:** Minh thấy banner ngay khi mở Dashboard lần tiếp theo, không phải tìm trong trang Ngân sách. Anh đóng banner sau khi đã ghi nhận, quyết định giảm ăn ngoài.

Thất bại: nếu Minh tiếp tục chi tiêu vượt 100% trước khi đóng banner 90%, một banner "over" (rose) mới xuất hiện thêm — không thay thế banner amber cho tới khi Minh tự đóng nó, vì hai ngưỡng là hai sự kiện độc lập theo FR-6.

### Flow 3 — Minh xem lại một khoản chi bị gắn cờ (UJ-3)

1. Minh mở màn hình Giao dịch, thấy một dòng có `anomaly-badge` màu violet cạnh tên danh mục.
2. Nhấn badge, popover nhỏ hiện ra: "Cao hơn 1.5 lần trung bình 3 chu kỳ gần nhất của danh mục này."
3. Minh nhớ lại đây là một khoản mua sắm đột xuất có chủ đích, không phải sai sót.
4. **Climax:** Minh nhấn "Đã xem, không vấn đề" — badge biến mất khỏi dòng đó, Minh tiếp tục xem các giao dịch khác mà không bị badge này làm phân tâm nữa.

Thất bại: nếu Minh không chắc và muốn xem lại sau, anh chỉ cần đóng popover mà không nhấn nút — badge vẫn còn nguyên ở đó cho lần xem tiếp theo, không có hạn để phải xử lý.
