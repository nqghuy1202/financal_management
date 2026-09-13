---
title: "Product Brief: HL Personal Finance — Cải tổ"
status: ready
created: 2026-09-11
updated: 2026-09-11
---

# Product Brief: HL Personal Finance — Cải tổ

## Executive Summary

Mỗi tháng lương về, và mỗi tháng lại hết sạch trước khi lương kế tiếp về — đó là lý do dự án quản lý tài chính cá nhân này tồn tại. Phiên bản trước (Go + React + MySQL, đã có auth, giao dịch, dashboard, ngân sách) ghi chép được dòng tiền nhưng **thụ động**: nó cho biết bạn *đã* tiêu gì, chứ không giúp bạn *tránh* tiêu quá tay trước khi quá muộn. Kết quả là nó chưa từng được dùng thật.

Bản cải tổ này giữ nguyên nền tảng kỹ thuật đã có (Go + React + MySQL, đang được refactor có hệ thống) nhưng đổi trọng tâm sản phẩm: từ "sổ ghi chép" sang "trợ lý kiểm soát dòng tiền chủ động". Ba trụ cột trải nghiệm: biết mỗi ngày còn được tiêu bao nhiêu, được cảnh báo trước khi vượt ngân sách một danh mục, và được gợi ý khi có khoản chi bất thường đáng xem lại — dựng trên một nền tảng kỹ thuật đáng tin cậy để ba điều đó thực sự vận hành đúng. Mục tiêu ban đầu là một công cụ cá nhân đáng tin cậy, dùng được ngay trong vài tuần tới — với đường mở rộng rõ ràng sang sổ chung gia đình ở giai đoạn sau.

## The Problem

Người dùng nhận lương hàng tháng nhưng không có cách nào nhìn thấy trước rằng mình sẽ hết tiền — chỉ nhận ra khi tài khoản đã cạn. Hiện tại không có công cụ nào đang được dùng để theo dõi việc này. Phiên bản app cũ tồn tại nhưng chưa từng được đưa vào dùng thật: nó chỉ ghi lại lịch sử thu-chi, không chủ động cảnh báo hay định hướng hành vi chi tiêu trong lúc còn kịp điều chỉnh.

Hậu quả cụ thể: không biết ngân sách một danh mục (ví dụ ăn uống, giải trí) đã gần cạn cho tới khi đã tiêu quá; không có con số rõ ràng "hôm nay còn được tiêu bao nhiêu" để tự điều chỉnh trong ngày; các khoản chi bất thường trôi qua mà không ai gắn cờ để nhìn lại. Về lâu dài, đây là nguyên nhân trực tiếp khiến người dùng không tiết kiệm được, dù thu nhập ổn định hàng tháng.

## The Solution

Một app quản lý tài chính cá nhân **chủ động**, xây trên nền Go + React + MySQL hiện có nhưng bổ sung một lớp "kiểm soát dòng tiền theo thời gian thực":

- **Safe-to-spend hàng ngày** — công thức lương − chi phí cố định − mục tiêu tiết kiệm = số tiền còn được tiêu, chia đều cho số ngày còn lại trong tháng, hiển thị ngay khi mở app.
- **Cảnh báo ngân sách chủ động** — khi một danh mục chi tiêu sắp/đã vượt hạn mức, hệ thống báo ngay trong app (không cần chờ xem báo cáo cuối tháng).
- **Gắn cờ chi tiêu bất thường** — so sánh khoản chi mới với thói quen chi tiêu trước đó của chính người dùng, gắn cờ những khoản lệch bất thường để tự xem lại có thực sự cần thiết không.
- **Nền tảng kỹ thuật đáng tin cậy** — refactor lại kiến trúc backend để dữ liệu chính xác, ổn định, không còn các lỗi vặt khiến người dùng mất niềm tin vào số liệu hiển thị.

## What Makes This Different

Đây không phải một sản phẩm thương mại cạnh tranh với Money Lover, Spendee hay YNAB — những app đó đã làm tốt việc lập ngân sách và cảnh báo. Sự khác biệt ở đây là **quyền sở hữu và mức độ vừa vặn cá nhân**: đây là công cụ được xây đúng theo chu kỳ lương – chi tiêu – tiết kiệm của chính người dùng, dữ liệu do mình kiểm soát hoàn toàn, và có thể tuỳ biến logic "safe-to-spend" hay quy tắc phát hiện bất thường theo đúng thói quen thật của bản thân (và sau này của gia đình) — điều mà một app đóng gói sẵn không thể linh hoạt bằng. Đây là lợi thế của việc tự xây, không phải một lợi thế cạnh tranh khó sao chép (moat) về kỹ thuật hay thị trường.

## Who This Serves

**Người dùng chính**: chính người xây dựng dự án — nhân viên nhận lương hàng tháng, có xu hướng tiêu hết trước cuối tháng vì thiếu tín hiệu cảnh báo sớm. Thành công với họ là: mở app bất kỳ lúc nào trong tháng và biết ngay mình đang ổn hay sắp vượt tầm kiểm soát.

**Người dùng phụ (Giai đoạn 2)**: thành viên trong một hộ gia đình/nhóm nhỏ, cùng theo dõi và đóng góp vào một sổ chi tiêu chung (một quỹ, một ngân sách chia sẻ) — chưa nằm trong phạm vi bản đầu tiên.

## Success Criteria

Bản cải tổ được xem là thành công khi:

- Người dùng nhận được cảnh báo **trước khi** vượt ngân sách một danh mục, không phải sau.
- Người dùng biết ngay trong app con số "hôm nay còn tiêu được bao nhiêu" mà không cần tự tính tay.
- Ít nhất các khoản chi bất thường rõ rệt (lệch nhiều so với thói quen) được gắn cờ để xem lại.
- Dữ liệu và tính năng hiện có (giao dịch, dashboard, ngân sách, danh mục, đăng nhập) hoạt động ổn định, không còn lỗi vặt của bản cũ.
- Người dùng thực sự **dùng app hàng ngày/hàng tuần** trong ít nhất một chu kỳ lương trọn vẹn — thước đo thật là hành vi dùng, không phải số liệu kỹ thuật.

## Scope

**Trong phạm vi bản cải tổ đầu tiên (vài tuần tới):**
- Hoàn thiện refactor backend đã bắt đầu (repository layer, config tách riêng) trên nền Go/Gin/MySQL hiện có.
- Giữ và làm chắc lại các tính năng nền: auth, giao dịch, danh mục, ngân sách, dashboard.
- Tính năng mới: safe-to-spend hàng ngày; cảnh báo vượt/gần vượt ngân sách theo danh mục (hiển thị trong app); gắn cờ chi tiêu bất thường so với thói quen cá nhân.
- Cảnh báo hiển thị **trong app khi mở lên** — chưa cần email/push notification.

**Ngoài phạm vi (để sau, Giai đoạn 2+):**
- Sổ chung nhiều người dùng / ngân sách chia sẻ gia đình.
- Kênh cảnh báo qua email, push notification, hay bot (Telegram/Zalo).
- Bất kỳ tính năng hướng đến portfolio (dự án trưng bày cho nhà tuyển dụng)/kinh doanh/thương mại hoá.

## Vision

Nếu thành công, đây trở thành trung tâm tài chính của không chỉ một người mà một hộ gia đình nhỏ: mỗi thành viên đóng góp và theo dõi trên cùng một sổ chung, ngân sách được thoả thuận và giám sát cùng nhau, và logic "safe-to-spend" đủ thông minh để không chỉ cảnh báo mà còn gợi ý điều chỉnh (ví dụ dồn tiết kiệm cho một mục tiêu cụ thể). Về kỹ thuật, đây là một hệ thống nhỏ nhưng vững chắc — dễ bảo trì, dữ liệu đáng tin cậy — phản ánh đúng tinh thần ban đầu: một công cụ được xây để thực sự dùng, không phải để trưng bày.

---

**Bước tiếp theo**: brief này là đầu vào cho PRD (`bmad-prd`), sau đó là kiến trúc kỹ thuật (`bmad-architecture`) — xem chi tiết ràng buộc kỹ thuật và các lựa chọn đã cân nhắc trong `addendum.md`.
