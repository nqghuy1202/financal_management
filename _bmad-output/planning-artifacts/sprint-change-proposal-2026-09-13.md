# Sprint Change Proposal — 2026-09-13

## 1. Issue Summary

Dự án là **một người dùng duy nhất, tự làm cho chính mình dùng** (không phải sản phẩm nhiều người dùng, không có team). Khi rà soát việc còn lại, người dùng đặt câu hỏi: bước nào trong kế hoạch là ceremony/overhead của quy trình nhiều người, không tạo giá trị gì thêm khi chỉ có một mình.

Sau khi phân tích, người dùng quyết định: **cắt hẳn Epic 3 — "Tự nhận ra chi tiêu bất thường"** (Story 3.1, 3.2 — gắn cờ giao dịch bất thường >1.5x trung bình 3 chu kỳ). Lý do: một mình người dùng đã tự biết mình chi tiêu bất thường khi nào; tính năng tự động phát hiện không cần thiết cho việc dùng cá nhân. Đây là quyết định cắt scope vĩnh viễn, không phải hoãn lại.

**Trigger:** Câu hỏi trực tiếp từ người dùng, không phải lỗi kỹ thuật hay phát hiện trong lúc code.

## 2. Impact Analysis

**Epic Impact:**
- Epic 3 (backlog, chưa bắt đầu code) — xoá khỏi kế hoạch.
- Epic 1, Epic 2 — không ảnh hưởng. Epic 3 chỉ phụ thuộc `CycleWindow()` (đọc, không sửa) từ Epic 1; không có epic nào phụ thuộc ngược lại vào Epic 3.

**Story Impact:**
- 3.1, 3.2 — xoá khỏi sprint-status.yaml và epics.md.
- Story 2.3 (budget-status-pill, backlog) — không ảnh hưởng, độc lập với Epic 3.

**Artifact Conflicts:**
- `epics.md` — Epic 3, FR8/FR9/FR10 trong FR Coverage Map, và mọi tham chiếu Epic 3 cần xoá/đánh dấu out-of-scope.
- `prd.md` (FR-8, FR-9, FR-10, §4.3, UJ-3, SM-4, SM-C2) — vẫn còn mô tả đầy đủ tính năng đã bị cắt. Không viết lại toàn bộ PRD (không cần thiết cho dự án cá nhân), nhưng thêm chú thích "cắt khỏi v1" tại đầu §4.3 để không gây hiểu lầm cho lần đọc sau.
- `ARCHITECTURE-SPINE.md` — có nhắc `transaction_flags`, `FlagRepo`, `PreviousCycles` cho FR-8/9/10. Bảng `transaction_flags` và `FlagRepo` **chưa từng được tạo** (chưa có code) nên không cần rollback gì ở tầng DB/code. Riêng `PreviousCycles()` trong `internal/api/cycle.go` đã được viết sẵn làm hạ tầng dùng chung (theo AD-3) nhưng giờ không còn nơi nào gọi tới — trở thành dead code vô hại (không lỗi build vì Go không cảnh báo hàm exported không dùng), có thể dọn sau, không khẩn cấp.

**Technical Impact:** Không có — chưa có dòng code nào của Epic 3 được viết ngoài `PreviousCycles()`.

## 3. Recommended Approach

**Option 1 — Direct Adjustment (chọn):** Xoá Epic 3 khỏi `epics.md` và `sprint-status.yaml`; chú thích ngắn gọn trong `prd.md` và `ARCHITECTURE-SPINE.md` rằng FR-8/9/10 đã bị cắt khỏi v1 theo quyết định của người dùng (giữ nguyên phần mô tả cũ làm lịch sử, không xoá, để không mất ngữ cảnh nếu sau này đổi ý).

- Effort: Thấp (chỉ sửa tài liệu, không đụng code)
- Risk: Rất thấp (chưa có code/DB nào cho Epic 3 để rollback)

Option 2 (Rollback) không áp dụng — không có gì để rollback. Option 3 (PRD MVP review toàn diện) không cần thiết — đây là một cắt giảm cục bộ, không ảnh hưởng MVP goal chính (Epic 1 + Epic 2 vẫn đứng vững độc lập).

## 4. Detailed Change Proposals

### 4.1 `epics.md`
- Xoá mục "Epic 3: Tự nhận ra chi tiêu bất thường" (cả phần tổng quan lẫn Story 3.1/3.2 chi tiết).
- Xoá FR8/FR9/FR10 khỏi "FR Coverage Map" (thay bằng ghi chú "cắt khỏi v1").
- Xoá FR8/FR9/FR10 khỏi "Requirements Inventory" hoặc đánh dấu out-of-scope tại chỗ.

### 4.2 `sprint-status.yaml`
- Xoá 4 dòng: `epic-3`, `3-1-...`, `3-2-...`, `epic-3-retrospective`.

### 4.3 `prd.md`
- Thêm một dòng chú thích ngay trước §4.3 và trước FR-8: *"[CẮT KHỎI V1 — 2026-09-13] Quyết định của người dùng: dự án một-người-dùng không cần tính năng tự động gắn cờ bất thường. Giữ nguyên mô tả bên dưới làm lịch sử/tham khảo, không triển khai."*
- Thêm chú thích tương tự cạnh SM-4 và SM-C2 (đều phụ thuộc FR-8).

### 4.4 `ARCHITECTURE-SPINE.md`
- Thêm chú thích ngắn tại bảng traceability FR-10 và tại định nghĩa `transaction_flags`: *"[CẮT KHỎI V1] — bảng/route này không còn được triển khai."*

## 5. Implementation Handoff

**Scope: Minor** — chỉ sửa tài liệu, không cần backlog reorg hay kiến trúc lại. Thực hiện trực tiếp ngay trong phiên này.

**Success criteria:** `epics.md` và `sprint-status.yaml` không còn liệt kê Epic 3/3.1/3.2 như việc cần làm; `prd.md`/`ARCHITECTURE-SPINE.md` không gây hiểu lầm cho người đọc sau này rằng đây vẫn là việc đang chờ.
