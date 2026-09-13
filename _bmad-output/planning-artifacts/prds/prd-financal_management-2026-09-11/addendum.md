# Addendum: PRD — HL Personal Finance — Cải tổ

Nội dung bổ trợ cho `prd.md` — căn cứ tham khảo và các lựa chọn đã cân nhắc, hữu ích cho `bmad-architecture` và `bmad-ux`, nhưng không cần thiết trong PRD chính.

## Căn cứ cho các giá trị `[ASSUMPTION]` trong PRD

Nghiên cứu nhanh về cách các app tài chính cá nhân khác triển khai ba năng lực tương tự, dùng để chọn giá trị mặc định hợp lý thay vì tự đặt ra tuỳ tiện:

- **Safe-to-spend**: PocketGuard ("In My Pocket") dùng công thức Thu nhập ước tính − Hoá đơn sắp tới − Mục tiêu tiết kiệm − Ngân sách danh mục đã phân bổ, tính lại theo thời gian thực và chia đều cho số ngày còn lại trong kỳ. Safe to Spend của Rocket Money, dành cho thu nhập không đều, cho phép chọn giữa "tháng thấp nhất gần đây" hoặc "trung bình 3 tháng". → PRD chọn công thức đơn giản hơn (Lương − Chi phí cố định − Mục tiêu tiết kiệm, chia đều số ngày còn lại) vì v1 giả định thu nhập cố định một nguồn.
- **Một con số vs. nhiều "bucket"** (đã cân nhắc và từ chối): Monarch Money tách kế hoạch thành 3 nhóm (Fixed / Flexible / Non-monthly) thay vì một con số duy nhất, để việc điều chỉnh giữa tháng chỉ ảnh hưởng nhóm Flexible. YNAB từ chối hoàn toàn việc dự báo "safe-to-spend" vì lo ngại tạo cảm giác khan hiếm sai lệch, dùng zero-based budgeting theo từng danh mục thay thế. → PRD **chọn một con số Safe-to-spend duy nhất** (đơn giản hơn cho v1, đúng tinh thần "biết ngay không cần tính tay" từ brief) thay vì mô hình nhiều bucket của Monarch; có thể xem xét tách bucket ở giai đoạn sau nếu một con số duy nhất gây hiểu lầm.
- **Ngưỡng cảnh báo ngân sách**: mô hình ba mốc 70%/90%/100% (thay vì một mốc 80%/100%) phổ biến trong tài liệu về budget alerts (Infracost glossary) để giảm cảm giác "bất ngờ" ở mốc cuối. Mint dùng cơ chế gộp cảnh báo theo tuần thay vì báo từng giao dịch — PRD áp dụng nguyên tắc tương tự qua FR-6 (không lặp cảnh báo cùng ngưỡng trong cùng chu kỳ).
- **Gắn cờ bất thường**: Mint dùng heuristic thống kê đơn giản (so tổng chi danh mục hiện tại với trung bình lịch sử, không dùng ML) — đây là cách khả thi cho một dự án cá nhân tự xây, khác với các mô hình ML nặng của các app lớn. PRD dùng ngưỡng 1.5 lần trung bình 3 chu kỳ gần nhất làm điểm khởi đầu.

*Nguồn tham khảo (chỉ mang tính định hướng, không phải benchmark chính thức): PocketGuard, Rocket Money, Money Lover, Monarch Money, YNAB, Mint, Infracost budget-alerts glossary.*

## Quyết định đã giải quyết trong PRD (không còn là Open Question)

- **Chi phí cố định — nhập tay vs. tự suy luận**: v1 chỉ cho phép người dùng tự khai báo Chi phí cố định (FR-3), không tự động suy luận từ lịch sử giao dịch định kỳ. Lý do: giữ v1 đơn giản, tránh logic phát hiện "giao dịch định kỳ" phức tạp không cần thiết cho mục tiêu ban đầu. Có thể xem xét gợi ý tự động ở giai đoạn sau khi đã có đủ dữ liệu thật.

## Ghi chú cho `bmad-architecture`

- Refactor backend đang dở dang (xem addendum của brief) cần được rà soát trước khi thiết kế thêm bảng/luồng dữ liệu cho Lương, Chi phí cố định, Mục tiêu tiết kiệm, Chu kỳ ngân sách, và trạng thái đã cảnh báo và đã gắn cờ/đã xem.
- FR-6 (không lặp cảnh báo cùng ngưỡng/chu kỳ) và FR-10 (không gắn cờ khi thiếu dữ liệu nền) đều cần một dạng trạng thái lưu trữ theo (user, danh mục, chu kỳ, ngưỡng) — đáng để kiến trúc sư thiết kế một bảng trạng thái riêng thay vì suy luận lại mỗi lần từ lịch sử giao dịch thô.
- `TAI_LIEU_KY_THUAT.docx` (chưa được rà soát — xem PRD §9 Open Question 3) nên được kiểm tra ở bước này phòng khi có ràng buộc kỹ thuật chưa được nhắc tới.
