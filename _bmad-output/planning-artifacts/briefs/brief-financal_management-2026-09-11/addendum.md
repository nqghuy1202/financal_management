# Addendum: HL Personal Finance — Cải tổ

Nội dung bổ trợ cho brief.md — hữu ích cho PRD và kiến trúc, nhưng không cần thiết trong bản brief chính.

## Ràng buộc kỹ thuật hiện có

- Stack giữ nguyên: Go 1.26 (Gin, `database/sql`), React 19 + TypeScript + Vite 6 + Tailwind v4, MySQL 8, JWT (bcrypt), Docker/Docker Compose. Một binary duy nhất (`cmd/web`) phục vụ cả API (`/api/*`) và SPA đã build.
- Refactor backend **đã bắt đầu trước brief này**: các file `internal/api/repo_budget.go`, `repo_category.go`, `repo_helpers.go`, `repo_transaction.go`, `repo_user.go`, `config.go` đã được thêm (chưa commit) để tách repository layer ra khỏi các file `*.go` gốc (`api.go`, `auth.go`, `budgets.go`, `categories.go`, `db.go`, `transactions.go`). Kiến trúc sư (bmad-agent-architect / bmad-architecture) nên rà soát phần refactor dở dang này trước khi thiết kế tiếp, thay vì giả định toàn bộ code vẫn ở trạng thái cũ.
- File `TAI_LIEU_KY_THUAT.docx` tồn tại trong repo (chưa được Git theo dõi) nhưng **không được dùng làm input cho brief này** — người dùng xác nhận không có tài liệu nguồn nào khác. Nếu tài liệu này có nội dung liên quan, nên rà soát riêng ở bước kiến trúc/PRD.

## Quyết định đã cân nhắc — Sổ chung gia đình (Giai đoạn 2)

**Quyết định**: Sổ chung một nhóm kiểu gia đình — một quỹ/ngân sách chung nhiều người cùng xem/nhập — được chọn, nhưng dời sang Giai đoạn 2 (sau MVP cá nhân).

Ba lựa chọn đã cân nhắc khi hỏi về hình thức "dùng chung hội nhóm":
1. **Sổ chung một nhóm (gia đình)** — một quỹ/ngân sách chung nhiều người cùng xem/nhập. → Được chọn.
2. Chia tiền nhóm kiểu Splitwise (mỗi người tự quản lý, chỉ chia đều chi phí chung) — không chọn vì không phù hợp nhu cầu "hộ gia đình" mà người dùng mô tả.
3. Chỉ cá nhân, không có nhóm — không chọn vì người dùng có ý định dùng chung sau này.

Quyết định này ảnh hưởng đến mô hình dữ liệu. Kiến trúc sư nên tính trước khả năng mở rộng từ "user sở hữu dữ liệu" sang "nhóm sở hữu dữ liệu, user thuộc nhóm" để không phải viết lại từ đầu ở Giai đoạn 2. Tuy nhiên, **không cần xây multi-tenant/nhóm ngay trong MVP**.

## Kênh cảnh báo — quyết định và lý do

**Quyết định**: chỉ cảnh báo trong app cho MVP — không xây email/push notification ngay từ đầu.

Các kênh đã cân nhắc: trong app / email / push notification / kết hợp. Trong app được chọn vì tránh phải xây hạ tầng gửi email/push ngay từ đầu; có thể bổ sung sau nếu thấy cần — ví dụ khi có sổ chung gia đình, push notification sẽ hữu ích hơn vì nhiều người không mở app cùng lúc.

## Không phải trọng tâm của bản cải tổ này

- Không có mục tiêu thương mại hoá, không có áp lực "portfolio để xin việc".
- README hiện tại của repo có ngôn ngữ hướng tới nhà tuyển dụng ("recruiters can explore") — nội dung đó nên được xem lại/loại bỏ vì không còn phản ánh đúng mục đích dự án.
- Không đặt ngân sách cho hạ tầng trả phí — giả định tiếp tục self-host/free-tier như hiện tại.
