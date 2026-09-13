---
title: HL Personal Finance — Cải tổ
created: 2026-09-11
updated: 2026-09-11
status: final
---

# PRD: HL Personal Finance — Cải tổ
*Working title — confirm.*

## 0. Document Purpose

Tài liệu này dành cho chính người xây dựng dự án (đóng vai trò vừa PM vừa dev), và cho các bước tiếp theo (`bmad-ux`, `bmad-architecture`, `bmad-create-epics-and-stories`) sẽ đọc và triển khai từ đây. Quy ước đọc:

- Thuật ngữ được định nghĩa một lần trong Glossary (§3) và dùng nguyên văn xuyên suốt.
- Tính năng được nhóm theo hành vi; mỗi tính năng chứa các Yêu cầu Chức năng (FR) được đánh số toàn cục (FR-1, FR-2, ...) để các bước sau tham chiếu ổn định.
- Giả định được đánh dấu `[ASSUMPTION]` ngay tại chỗ và gom lại ở §10.

Tài liệu này xây trên [`brief.md`](../../briefs/brief-financal_management-2026-09-11/brief.md) — không lặp lại nội dung đó, chỉ tham chiếu khi cần (ví dụ phần "What Makes This Different" — quyền sở hữu và mức độ vừa vặn cá nhân — sống ở brief, không lặp lại ở đây vì không sinh ra FR nào).

## 1. Vision

Mỗi tháng lương về, và mỗi tháng lại hết tiền trước khi lương kế tiếp về. Phiên bản trước của app này (Go + React + MySQL, đã có auth, giao dịch, dashboard, ngân sách) ghi chép được dòng tiền nhưng thụ động — nó cho biết đã tiêu gì, không giúp tránh tiêu quá tay. Kết quả: chưa từng được dùng thật.

Bản cải tổ này biến app từ "sổ ghi chép" thành "trợ lý kiểm soát dòng tiền chủ động", xây trên nền tảng kỹ thuật hiện có. Ba năng lực mới — biết mỗi ngày còn tiêu được bao nhiêu, được cảnh báo trước khi vượt ngân sách, và được gợi ý khi có khoản chi bất thường — cộng với việc làm chắc lại các tính năng nền đã có, tạo thành một công cụ đáng tin cậy, dùng được ngay trong vài tuần tới. Đường mở rộng sang sổ chung gia đình (Giai đoạn 2) được chuẩn bị trước ở tầng dữ liệu nhưng không xây ngay.

Về lâu dài, Safe-to-spend có thể tiến thêm một bước từ chỉ cảnh báo sang chủ động *gợi ý* điều chỉnh — nhưng đó là hướng mở rộng sau v1, không sinh FR nào ở đây. Mục tiêu xuyên suốt vẫn là một công cụ được xây để thực sự dùng hàng ngày, không phải để trưng bày.

## 2. Target User

### 2.1 Jobs To Be Done

- Biết ngay khi mở app: "hôm nay tôi còn tiêu được bao nhiêu mà không ảnh hưởng đến phần còn lại của tháng?"
- Được cảnh báo **trước khi** một khoản chi tiêu đẩy một danh mục vượt ngân sách, không phải sau khi đã lỡ.
- Nhận ra một khoản chi bất thường của chính mình để tự hỏi "khoản này có thực sự cần thiết không?"
- Tin tưởng số liệu hiển thị (giao dịch, ngân sách, dashboard) là chính xác, không bị lỗi vặt làm sai lệch quyết định chi tiêu.

### 2.2 Non-Users (v1)

- Thành viên hộ gia đình/nhóm dùng chung một sổ — đây là người dùng có thật nhưng thuộc Giai đoạn 2, không phải v1.
- Người cần đồng bộ ngân hàng tự động, theo dõi đầu tư, hay tái cấu trúc nợ — ngoài phạm vi hoàn toàn (xem §5 Non-Goals).
- Nhà tuyển dụng/người xem portfolio — sản phẩm không còn nhắm tới nhóm này (xem addendum của brief).

### 2.3 Key User Journeys

- **UJ-1. Minh mở app ngay sau khi lương về.**
  - **Persona + context:** Minh, nhân viên văn phòng nhận lương cố định hàng tháng, vừa nhận lương và muốn biết tháng này nên chi tiêu thế nào để không lặp lại cảnh hết tiền như mọi khi.
  - **Entry state:** đã đăng nhập từ trước (JWT còn hiệu lực), mở app trên điện thoại.
  - **Path:** (1) Mở app, màn hình Dashboard hiện ngay; (2) Nếu chưa cập nhật Lương/chi phí cố định/mục tiêu tiết kiệm cho chu kỳ mới, app nhắc xác nhận lại các con số đó trước khi tính; (3) Minh xác nhận hoặc chỉnh sửa Lương nhận được kỳ này; (4) Thấy số "hôm nay còn tiêu được bao nhiêu" đã tính lại cho chu kỳ mới, chia đều cho số ngày còn lại.
  - **Climax:** Minh thấy một con số cụ thể, đáng tin — không phải tự cộng trừ tay.
  - **Resolution:** Minh biết giới hạn chi tiêu hôm nay và có thể quay lại kiểm tra bất cứ lúc nào trong tháng. Realizes FR-1, FR-2, FR-3, FR-4.
  - **Edge case:** nếu Minh chưa nhập chi phí cố định lần nào, app dùng giá trị mặc định là 0 và hiển thị cảnh báo nhẹ rằng Safe-to-spend sẽ không chính xác cho tới khi được cập nhật.

- **UJ-2. Minh sắp vượt ngân sách ăn uống giữa tháng.**
  - **Persona + context:** Minh vừa ghi một giao dịch ăn uống, đây là lần thứ N trong tháng ở danh mục này.
  - **Entry state:** đã đăng nhập, đang ở màn hình Giao dịch, vừa lưu một giao dịch mới.
  - **Path:** (1) Minh lưu giao dịch; (2) Hệ thống tính lại % đã dùng của Ngân sách Danh mục "Ăn uống"; (3) Vì đã vượt ngưỡng cảnh báo (xem FR-6), một thông báo trong app hiện ra ngay: "Ăn uống: đã dùng 92% ngân sách tháng, còn X ngày"; (4) Minh xem thông báo, quyết định giảm chi tiêu ăn ngoài những ngày còn lại.
  - **Climax:** Minh biết ngay tại thời điểm phát sinh chi tiêu, không phải cuối tháng mới biết.
  - **Resolution:** Minh điều chỉnh hành vi còn kịp trong chu kỳ hiện tại. Realizes FR-5, FR-6, FR-7.
  - **Edge case:** nếu Minh nhập nhiều giao dịch liên tiếp trong cùng danh mục sau khi đã vượt ngưỡng, hệ thống không lặp lại cảnh báo cho cùng một ngưỡng trong cùng chu kỳ (tránh làm phiền).

- **UJ-3. Minh thấy một khoản chi bất thường được gắn cờ.**
  - **Persona + context:** Minh vừa chi một khoản lớn bất thường so với thói quen (ví dụ mua sắm đột xuất).
  - **Entry state:** đã đăng nhập, vừa lưu giao dịch hoặc đang xem lại danh sách giao dịch.
  - **Path:** (1) Hệ thống so sánh khoản chi mới với thói quen chi tiêu trước đó của Minh trong cùng Danh mục; (2) Giao dịch vượt Ngưỡng bất thường được gắn cờ trực quan trong danh sách; (3) Minh mở giao dịch, thấy ghi chú ngắn giải thích vì sao bị gắn cờ (ví dụ: "cao hơn 1.5 lần mức trung bình 3 tháng gần nhất của danh mục này"); (4) Minh đánh dấu "đã xem, không vấn đề" hoặc để nguyên như một lời nhắc tự xem lại thói quen.
  - **Climax:** Minh tự nhận ra khoản chi cần xem lại mà không cần tự phân tích số liệu.
  - **Resolution:** giao dịch được đánh dấu đã xem, không bị gắn cờ lặp lại. Realizes FR-8, FR-9, FR-10.
  - **Edge case:** nếu Minh mới dùng app và chưa có đủ lịch sử chi tiêu (dưới 1 chu kỳ dữ liệu), hệ thống không gắn cờ bất thường cho tới khi có đủ dữ liệu tối thiểu để so sánh — tránh gắn cờ sai vì thiếu baseline.

## 3. Glossary

- **Lương** — tổng thu nhập người dùng khai báo cho một Chu kỳ ngân sách; đầu vào đầu tiên của công thức Safe-to-spend (xem FR-2).
- **Safe-to-spend (số tiền an toàn có thể chi tiêu)** — số tiền còn có thể chi tiêu trong ngày hiện tại, tính bằng (Lương − Chi phí cố định − Mục tiêu tiết kiệm) chia đều cho số ngày còn lại trong Chu kỳ ngân sách.
- **Chu kỳ ngân sách** — khoảng thời gian một chu kỳ tính toán ngân sách/safe-to-spend lặp lại, mặc định là tháng dương lịch (ngày 1 đến hết tháng); có thể cấu hình ngày bắt đầu khác (xem FR-4).
- **Chi phí cố định** — các khoản chi định kỳ, số tiền tương đối cố định mỗi chu kỳ (ví dụ tiền nhà, hoá đơn) mà người dùng khai báo, dùng làm đầu vào cho Safe-to-spend.
- **Mục tiêu tiết kiệm** — số tiền người dùng muốn giữ lại mỗi chu kỳ, trừ ra khỏi Safe-to-spend trước khi chia cho số ngày còn lại.
- **Danh mục** — nhóm phân loại một giao dịch (ví dụ Ăn uống, Giải trí); đã tồn tại trong hệ thống hiện tại.
- **Ngân sách** — hạn mức chi tiêu người dùng đặt cho một Danh mục trong một Chu kỳ ngân sách; đã tồn tại trong hệ thống hiện tại.
- **Giao dịch** — một bản ghi thu hoặc chi; đã tồn tại trong hệ thống hiện tại.
- **Cảnh báo ngân sách chủ động** — thông báo hiển thị trong app khi tổng chi của một Danh mục trong Chu kỳ ngân sách hiện tại chạm một Ngưỡng cảnh báo.
- **Ngưỡng cảnh báo** — mốc phần trăm của Ngân sách một Danh mục (70%/90%/100%) tại đó hệ thống phát Cảnh báo ngân sách chủ động.
- **Gắn cờ bất thường** — đánh dấu trực quan trên một Giao dịch mà hệ thống xác định là lệch đáng kể so với thói quen chi tiêu trước đó của người dùng trong cùng Danh mục, theo Ngưỡng bất thường.
- **Ngưỡng bất thường** — mốc (mặc định 1.5 lần trung bình 3 Chu kỳ ngân sách gần nhất của một Danh mục) tại đó một khoản chi được coi là đủ lệch để nhận Gắn cờ bất thường.
- **Sổ chung gia đình** — (Giai đoạn 2, chưa xây trong v1) một Ngân sách và tập Giao dịch được chia sẻ giữa nhiều người dùng thuộc cùng một nhóm.

## 4. Features

### 4.1 Safe-to-spend hàng ngày

**Description:** Ngay khi người dùng mở app, hệ thống hiển thị Safe-to-spend của ngày hiện tại, tính từ Lương, Chi phí cố định, Mục tiêu tiết kiệm, và số ngày còn lại trong Chu kỳ ngân sách. Đây là năng lực trung tâm của bản cải tổ — biến app từ ghi chép thụ động thành công cụ chủ động. Realizes UJ-1.

`[ASSUMPTION]`: v1 giả định thu nhập cố định, một nguồn, hàng tháng (khớp với bối cảnh "nhận lương hàng tháng" trong brief) — không xử lý thu nhập không đều hay nhiều nguồn thu.

**Functional Requirements:**

#### FR-1: Hiển thị Safe-to-spend khi mở app

Người dùng đã đăng nhập nhìn thấy Safe-to-spend của ngày hiện tại ngay trên Dashboard, không cần thao tác thêm. Realizes UJ-1.

**Consequences (testable):**
- Dashboard hiển thị một con số Safe-to-spend duy nhất, kèm số ngày còn lại trong Chu kỳ ngân sách.
- Giá trị được tính lại (không phải cache cũ) mỗi khi Dashboard được tải.
- Nếu có giao dịch chi tiêu mới trong ngày, Safe-to-spend hiển thị ngay sau đó phản ánh đúng số dư còn lại sau giao dịch đó.

#### FR-2: Khai báo/cập nhật Lương cho chu kỳ hiện tại

Người dùng có thể xem và cập nhật Lương của Chu kỳ ngân sách hiện tại; đây là đầu vào đầu tiên và bắt buộc của công thức Safe-to-spend. v1 chỉ hỗ trợ nhập tay một giá trị Lương duy nhất mỗi chu kỳ — không suy luận từ Giao dịch thu nhập, không hỗ trợ nhiều nguồn thu. Realizes UJ-1.

**Consequences (testable):**
- Nếu chưa khai báo Lương cho chu kỳ hiện tại, Safe-to-spend không hiển thị một con số gây hiểu lầm (ví dụ 0 hoặc âm) mà nhắc người dùng nhập Lương trước (xem edge case UJ-1).
- Thay đổi Lương cập nhật ngay giá trị Safe-to-spend hiển thị, không cần tải lại trang.
- Lương của một chu kỳ không tự động chuyển sang chu kỳ kế tiếp trừ khi người dùng xác nhận lại (chu kỳ mới luôn yêu cầu xác nhận, xem UJ-1 Path bước 2).

#### FR-3: Khai báo Chi phí cố định và Mục tiêu tiết kiệm

Người dùng có thể xem, thêm, sửa, xoá các khoản Chi phí cố định và một Mục tiêu tiết kiệm cho Chu kỳ ngân sách hiện tại; các giá trị này là đầu vào trực tiếp của công thức Safe-to-spend. v1 chỉ hỗ trợ khai báo thủ công — không tự động suy luận chi phí cố định từ lịch sử giao dịch (xem addendum để biết lý do). Realizes UJ-1.

**Consequences (testable):**
- Thay đổi Chi phí cố định hoặc Mục tiêu tiết kiệm cập nhật ngay giá trị Safe-to-spend hiển thị, không cần tải lại trang.
- Nếu chưa từng khai báo, hệ thống dùng giá trị 0 và hiển thị gợi ý nhập liệu (xem edge case UJ-1).

#### FR-4: Cấu hình ngày bắt đầu Chu kỳ ngân sách

Người dùng có thể chọn ngày trong tháng mà Chu kỳ ngân sách bắt đầu (mặc định: ngày 1); toàn bộ tính toán Safe-to-spend và Ngân sách theo Danh mục dùng chu kỳ này thay vì cố định theo tháng dương lịch. Realizes UJ-1.

**Consequences (testable):**
- Đổi ngày bắt đầu chu kỳ tính lại đúng số ngày còn lại và Safe-to-spend tương ứng.
- Chu kỳ hiện tại của mỗi người dùng được lưu và áp dụng nhất quán cho cả Safe-to-spend lẫn Cảnh báo ngân sách (§4.2).

**Notes:** `[NOTE FOR PM]` — nếu ngày nhận lương thực tế của người dùng lệch khỏi ngày 1 dương lịch, FR-4 là yêu cầu quan trọng để công thức không sai lệch; xác nhận với người dùng khi vào `bmad-ux`/khi triển khai xem có cần onboarding hỏi ngày này ngay từ đầu hay để mặc định ngày 1 trước.

### 4.2 Cảnh báo ngân sách chủ động

**Description:** Khi chi tiêu của một Danh mục trong Chu kỳ ngân sách hiện tại chạm một Ngưỡng cảnh báo, hệ thống hiển thị thông báo ngay trong app — không cần chờ báo cáo cuối chu kỳ. Realizes UJ-2.

`[ASSUMPTION]`: Ngưỡng cảnh báo mặc định là 70%/90%/100% ngân sách danh mục — mô hình ba mốc phổ biến ở các hệ thống theo dõi ngân sách nói chung (xem addendum), thay vì chỉ một mốc 80%/100%, để giảm cảm giác bị "bất ngờ" ở mốc cuối.

**Functional Requirements:**

#### FR-5: Phát cảnh báo khi chạm Ngưỡng cảnh báo

Hệ thống kiểm tra % đã dùng của Ngân sách mỗi Danh mục sau mỗi Giao dịch chi tiêu mới, và hiển thị Cảnh báo ngân sách chủ động trong app ngay khi một Ngưỡng cảnh báo (70%/90%/100%) bị chạm hoặc vượt. Realizes UJ-2.

**Consequences (testable):**
- Sau khi lưu một giao dịch đẩy một danh mục qua một ngưỡng, cảnh báo tương ứng xuất hiện trong cùng phiên làm việc đó (không cần tải lại trang riêng biệt để thấy).
- Cảnh báo nêu rõ: tên danh mục, % đã dùng, số ngày còn lại trong chu kỳ.

#### FR-6: Không lặp lại cảnh báo cùng ngưỡng trong cùng chu kỳ

`[ASSUMPTION]`: mỗi Ngưỡng cảnh báo chỉ báo một lần cho mỗi (Danh mục, Chu kỳ ngân sách) — lựa chọn để tránh làm phiền, tương tự cách Mint gộp cảnh báo thay vì báo mỗi giao dịch (xem addendum).

Với mỗi Danh mục và mỗi Ngưỡng cảnh báo, hệ thống chỉ hiển thị cảnh báo một lần cho mỗi Chu kỳ ngân sách, kể cả khi có thêm giao dịch mới sau đó vẫn ở trên ngưỡng đó. Realizes UJ-2.

**Consequences (testable):**
- Nhiều giao dịch liên tiếp giữ một Danh mục ở trên 90% chỉ tạo ra một cảnh báo "đã vượt 90%" cho chu kỳ đó, không phải một cảnh báo mỗi giao dịch.
- Khi chu kỳ mới bắt đầu, trạng thái "đã cảnh báo" của mọi ngưỡng được reset.

#### FR-7: Tổng quan trạng thái ngân sách trên Dashboard

Dashboard hiển thị trạng thái hiện tại của từng Danh mục có Ngân sách theo đúng ba mốc của Ngưỡng cảnh báo (FR-5) — **trong hạn mức**: dưới 70%; **gần vượt**: từ 70% đến dưới 100%; **đã vượt**: từ 100% trở lên — không chỉ khi có cảnh báo mới, để người dùng chủ động xem lại bất cứ lúc nào. Realizes UJ-1, UJ-2.

**Consequences (testable):**
- Mỗi danh mục có ngân sách hiển thị % đã dùng và trạng thái màu (trong hạn mức/gần vượt/đã vượt) trên Dashboard hoặc trang Ngân sách, đúng theo ranh giới 70%/100% nêu trên.

### 4.3 Gắn cờ chi tiêu bất thường

> **[CẮT KHỎI V1 — 2026-09-13]** Quyết định của người dùng: dự án một-người-dùng không cần tính năng tự động gắn cờ bất thường (người dùng đã tự biết mình chi tiêu bất thường khi nào). FR-8/FR-9/FR-10 và Epic 3 tương ứng bị xoá khỏi kế hoạch. Giữ nguyên mô tả bên dưới làm lịch sử/tham khảo, không triển khai. Xem `_bmad-output/planning-artifacts/sprint-change-proposal-2026-09-13.md`.

**Description:** Hệ thống so sánh mỗi giao dịch mới với thói quen chi tiêu trước đó của người dùng trong cùng Danh mục, và gắn cờ những giao dịch lệch đáng kể để người dùng tự xem lại. Đây là một gợi ý để suy ngẫm, không phải một lỗi cần sửa. Realizes UJ-3.

`[ASSUMPTION]`: Ngưỡng bất thường ban đầu = tổng chi danh mục trong chu kỳ hiện tại vượt quá 1.5 lần trung bình 3 chu kỳ gần nhất của danh mục đó (mô hình "rolling-average-plus-threshold" như Mint dùng) — đây là quy tắc đơn giản, không dùng machine learning, phù hợp quy mô dự án cá nhân.

**Functional Requirements:**

#### FR-8: Gắn cờ giao dịch/danh mục bất thường

Sau mỗi giao dịch mới, hệ thống tính tổng chi hiện tại của Danh mục trong Chu kỳ ngân sách và so với trung bình 3 chu kỳ gần nhất của cùng Danh mục; nếu vượt Ngưỡng bất thường, giao dịch hoặc Danh mục được gắn cờ trực quan. Realizes UJ-3.

**Consequences (testable):**
- Giao dịch bị gắn cờ có chỉ báo trực quan khác biệt trong danh sách Giao dịch.
- Giao dịch bị gắn cờ hiển thị lý do ngắn gọn (ví dụ: "cao hơn 1.5 lần trung bình 3 chu kỳ gần nhất").

**Out of Scope:** *(FR-8)*
- Không phát hiện gian lận/giao dịch trái phép (không phải mục tiêu bảo mật, chỉ là gợi ý xem lại hành vi chi tiêu của chính mình).

#### FR-9: Đánh dấu đã xem xét cờ bất thường

Người dùng có thể đánh dấu một giao dịch đã bị gắn cờ là "đã xem, không vấn đề" để cờ không còn hiển thị nổi bật cho giao dịch đó nữa. Realizes UJ-3.

**Consequences (testable):**
- Sau khi đánh dấu, giao dịch không còn hiển thị chỉ báo cờ nổi bật (có thể vẫn giữ log nội bộ rằng đã từng bị gắn cờ).

#### FR-10: Không gắn cờ khi thiếu dữ liệu nền

`[ASSUMPTION]`: cần tối thiểu 1 Chu kỳ ngân sách đầy đủ dữ liệu trước khi bắt đầu Gắn cờ bất thường cho một Danh mục — tránh gắn cờ sai vì thiếu baseline so sánh.

Hệ thống không gắn cờ bất thường cho một Danh mục cho tới khi có đủ lịch sử tối thiểu (ít nhất 1 Chu kỳ ngân sách đầy đủ dữ liệu) để tính trung bình so sánh. Realizes UJ-3 (edge case).

**Consequences (testable):**
- Người dùng mới hoặc danh mục mới tạo không nhận cờ bất thường nào cho tới khi đủ dữ liệu nền.

**Notes:** `[NOTE FOR PM]` — ngưỡng 1.5 lần và cửa sổ 3 chu kỳ là điểm khởi đầu hợp lý, không phải con số cố định; nên xem lại sau khi có dữ liệu sử dụng thật (xem §9 Open Questions).

## 5. Non-Goals (Explicit)

*Danh sách dưới đây áp dụng cho cả v1 lẫn MVP — hai khái niệm trùng nhau trong tài liệu này.*

- Không xây sổ chung/ngân sách chia sẻ nhiều người dùng trong v1 (Giai đoạn 2 — xem §2.2). `[NOTE FOR PM]` mô hình dữ liệu nên chuẩn bị trước cho việc này (xem addendum của brief), dù không xây UI/logic nhóm ngay.
- Không gửi cảnh báo qua email, push notification, hay bot (Telegram/Zalo) trong v1 — chỉ trong app. Để sau, khi sổ chung gia đình khiến kênh trong-app không còn đủ (nhiều người không mở app cùng lúc).
- Không đồng bộ ngân hàng tự động (Open Banking/bank sync) — nhập giao dịch vẫn là thủ công.
- Không theo dõi đầu tư, tài sản, hay nợ/vay — phạm vi chỉ là thu nhập/chi tiêu/ngân sách cá nhân.
- Không xây tính năng thương lượng hoá đơn, huỷ subscription hộ người dùng, hay bất kỳ tính năng nào của các app tài chính thương mại không liên quan trực tiếp đến ba năng lực cốt lõi (§4.1–4.3).
- Không nhắm tới mục đích portfolio/tuyển dụng hay thương mại hoá — không đầu tư công sức vào các yếu tố chỉ phục vụ mục đích đó (xem §9 Open Question 4 về README hiện tại).
- Không phát hiện gian lận/bảo mật giao dịch — Gắn cờ bất thường (§4.3) là công cụ tự phản ánh thói quen, không phải bảo mật.

## 6. MVP Scope

### 6.1 In Scope

- Hoàn thiện refactor backend đã bắt đầu (repository layer, config tách riêng) trên nền Go/Gin/MySQL hiện có.
- Giữ và làm chắc lại các tính năng nền: auth, giao dịch, danh mục, ngân sách, dashboard — không hồi quy so với hành vi hiện có.
- Safe-to-spend hàng ngày (FR-1 đến FR-4).
- Cảnh báo ngân sách chủ động, hiển thị trong app (FR-5 đến FR-7).
- Gắn cờ chi tiêu bất thường (FR-8 đến FR-10).

### 6.2 Out of Scope for MVP

Xem §5 Non-Goals — toàn bộ các loại trừ liệt kê ở đó áp dụng nguyên vẹn cho MVP.

## 7. Cross-Cutting NFRs

- **Reliability:** sau khi hoàn thiện refactor backend, toàn bộ hành vi CRUD hiện có (giao dịch, danh mục, ngân sách, auth, dashboard) phải cho kết quả nhất quán với trước refactor — không mất dữ liệu, không sai lệch số liệu hiển thị. Test: chạy lại các kịch bản thao tác thủ công hiện có trước/sau refactor và so sánh kết quả.
- **Performance:** Safe-to-spend và trạng thái ngân sách phải tải xong dưới khoảng 1 giây trên kết nối di động thông thường khi mở Dashboard `[ASSUMPTION: con số mềm, không đo benchmark chính thức]` — dữ liệu cá nhân, quy mô nhỏ, không cần tối ưu cho tải lớn.
- **Security & Privacy:** dữ liệu tài chính của một người dùng chỉ thuộc về người dùng đó (kế thừa cơ chế JWT + bcrypt + scoping theo user hiện có); khi Giai đoạn 2 (sổ chung) được xây, việc chia sẻ dữ liệu phải là hành động rõ ràng (tham gia nhóm), không mặc định. Không thêm bên thứ ba theo dõi/phân tích dữ liệu người dùng.
- **Cost:** không giới thiệu hạ tầng trả phí mới — tiếp tục self-host/free-tier như hiện tại.

## 8. Success Metrics

*Đơn vị tiền tệ trong toàn bộ tài liệu này là VND `[ASSUMPTION: kế thừa từ brief theo bối cảnh người dùng tại Việt Nam, chưa hỏi lại trực tiếp]`.*

**Primary**
- **SM-1**: Trong một chu kỳ ngân sách đầy đủ, người dùng nhận được ít nhất một Cảnh báo ngân sách chủ động *trước khi* một danh mục bị vượt hoàn toàn (100%). Validates FR-5, FR-6.
- **SM-2**: Người dùng mở app và xem Safe-to-spend ít nhất 3 lần/tuần trong suốt một chu kỳ ngân sách trọn vẹn `[ASSUMPTION: tần suất cụ thể do mình đề xuất làm ngưỡng đo được, chưa xác nhận với người dùng]` — thước đo hành vi dùng thật, không phải số liệu kỹ thuật. Validates FR-1.

**Secondary**
- **SM-3**: Không phát sinh lỗi dữ liệu/hồi quy nào trên các tính năng hiện có sau khi hoàn thiện refactor. Validates NFR Reliability (§7).
- ~~**SM-4**: Ít nhất một khoản chi bất thường thực sự đáng chú ý được gắn cờ và người dùng xác nhận là hữu ích (không phải false positive) trong chu kỳ đầu tiên sử dụng. Validates FR-8.~~ **[CẮT KHỎI V1]** phụ thuộc FR-8, đã cắt.

**Counter-metrics (không tối ưu quá mức)**
- **SM-C1**: Số lượng cảnh báo ngân sách mỗi tuần không vượt quá ~5-7 lần dù có nhiều danh mục — nếu vượt, đây là dấu hiệu Ngưỡng cảnh báo (§4.2) cần điều chỉnh, không phải thành công. Counterbalances SM-1.
- ~~**SM-C2**: Tỷ lệ giao dịch bị gắn cờ bất thường không vượt quá một phần nhỏ tổng giao dịch (ước lượng ban đầu <10%/chu kỳ) — gắn cờ quá nhiều làm mất giá trị cảnh báo. Counterbalances SM-4.~~ **[CẮT KHỎI V1]** phụ thuộc SM-4/FR-8, đã cắt.

## 9. Open Questions

1. ~~Có cần hỏi ngày bắt đầu Chu kỳ ngân sách (FR-4) ngay trong onboarding...~~ **Đã giải quyết tại `bmad-ux`**: không có bước onboarding riêng; ngày bắt đầu chu kỳ nằm trong mục "Nâng cao" (gấp lại mặc định) của sheet "Cập nhật chu kỳ", mở từ Safe-to-spend hero khi cần — xem EXPERIENCE.md §Information Architecture, §Key Flows Flow 1.
2. Ngưỡng cảnh báo 70/90/100% và Ngưỡng bất thường 1.5x trung bình 3 chu kỳ (đều là `[ASSUMPTION]`) có phù hợp với thói quen chi tiêu thật của người dùng không — nên xem lại sau khi có dữ liệu sử dụng thực tế. *(Không chặn tiến độ — revisit sau khi có ít nhất 1 chu kỳ dữ liệu thật.)*
3. `TAI_LIEU_KY_THUAT.docx` trong repo chưa được rà soát — có nội dung nào ảnh hưởng tới PRD này không? *(Không chặn PRD, nhưng nên kiểm tra khi vào `bmad-architecture` — xem addendum.)*
4. README hiện tại của repo còn ngôn ngữ hướng nhà tuyển dụng ("recruiters can explore...") không còn khớp mục đích dự án — nên dọn lại. *(Việc vặt, không chặn PRD/kiến trúc, nhưng nên làm trước khi công bố bản cải tổ.)*

## 10. Assumptions Index

- §4.1 (FR-2) — Lương giả định cố định, một nguồn, hàng tháng, nhập tay mỗi chu kỳ (không xử lý thu nhập không đều/nhiều nguồn trong v1).
- §4.2 — Ngưỡng cảnh báo ngân sách mặc định: 70% / 90% / 100%.
- §4.2 (FR-6) — Mỗi Ngưỡng cảnh báo chỉ cảnh báo một lần mỗi (Danh mục, Chu kỳ ngân sách).
- §4.3 — Ngưỡng bất thường: chi danh mục trong chu kỳ hiện tại > 1.5 lần trung bình 3 chu kỳ gần nhất của danh mục đó.
- §4.3 (FR-10) — Cần tối thiểu 1 chu kỳ dữ liệu lịch sử trước khi bắt đầu Gắn cờ bất thường.
- §7 (Performance) — Ngưỡng tải Dashboard <1 giây là con số mềm, chưa đo benchmark chính thức.
- §8 (SM-2) — Tần suất sử dụng "≥3 lần/tuần" là ngưỡng đo được do mình đề xuất, chưa xác nhận trực tiếp với người dùng.
- Toàn bộ tài liệu — đơn vị tiền tệ VND, theo bối cảnh người dùng tại Việt Nam (kế thừa từ brief, chưa hỏi lại trực tiếp).
