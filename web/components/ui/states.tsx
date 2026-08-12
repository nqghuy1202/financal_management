import clsx from "clsx";
import { ApiError } from "@/lib/api/client";

/**
 * Khối xám nhấp nháy thay cho nội dung đang tải.
 *
 * Dùng skeleton chứ không phải spinner vì skeleton giữ nguyên chiều cao
 * của nội dung thật, nên khi dữ liệu về trang không bị nhảy.
 */
export function Skeleton({ className }: { className?: string }) {
  return <div className={clsx("animate-pulse rounded-md bg-surface-2", className)} />;
}

/** Trạng thái rỗng: nói rõ vì sao trống và bước tiếp theo là gì. */
export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-4 py-12 text-center">
      <p className="text-sm font-medium text-fg">{title}</p>
      {description && <p className="max-w-sm text-sm text-fg-muted">{description}</p>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}

/**
 * Hiển thị lỗi bằng chính thông báo của backend.
 *
 * Backend đã sinh thông báo tiếng Việt cho người dùng cuối (xem
 * response/code.go), nên dịch lại ở đây chỉ tạo ra hai bộ chữ lệch nhau.
 * Lỗi không phải ApiError là lỗi ngoài dự tính — nói chung chung thay vì
 * phơi thông điệp kỹ thuật ra màn hình.
 */
export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  return "Đã có lỗi xảy ra, vui lòng thử lại";
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-4 py-12 text-center">
      <p className="text-sm text-danger">{errorMessage(error)}</p>
      {onRetry && (
        <button
          type="button"
          onClick={onRetry}
          className="text-sm font-medium text-brand underline underline-offset-4"
        >
          Thử lại
        </button>
      )}
    </div>
  );
}

/** Thông báo lỗi gọn trong form, đặt ngay trên nút gửi. */
export function FormError({ error }: { error: unknown }) {
  if (!error) return null;
  return (
    <p role="alert" className="rounded-lg bg-danger-soft px-3 py-2 text-sm text-danger">
      {errorMessage(error)}
    </p>
  );
}
