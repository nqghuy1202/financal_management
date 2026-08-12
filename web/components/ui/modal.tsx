"use client";

import { useEffect, useRef } from "react";
import { X } from "lucide-react";

/**
 * Hộp thoại dựng trên thẻ <dialog> của trình duyệt.
 *
 * Dùng thẻ có sẵn thay vì tự làm bằng <div> vì trình duyệt đã lo hết
 * phần khó: nhốt focus bên trong, đóng khi bấm Esc, che phần còn lại của
 * trang khỏi trình đọc màn hình. Tự làm lại những thứ đó thường thiếu
 * một hai chỗ, và chỗ thiếu luôn là chỗ người dùng bàn phím cần nhất.
 */
export function Modal({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;

    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  if (!open) return null;

  return (
    <dialog
      ref={ref}
      // Esc do trình duyệt xử lý và phát ra sự kiện close; lắng nghe ở đây
      // để trạng thái React không lệch với thứ đang hiện trên màn hình.
      onClose={onClose}
      onClick={(event) => {
        // Bấm ra ngoài thì đóng. Sự kiện trên phần nền lại có target chính
        // là <dialog>, nên so sánh này phân biệt được trong với ngoài.
        if (event.target === ref.current) onClose();
      }}
      className="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-xl border border-line bg-surface p-0 text-fg backdrop:bg-black/50"
    >
      <div className="flex items-center justify-between border-b border-line px-5 py-3">
        <h2 className="text-sm font-semibold">{title}</h2>
        <button
          type="button"
          onClick={onClose}
          aria-label="Đóng"
          className="rounded-md p-1 text-fg-muted hover:bg-surface-2 hover:text-fg"
        >
          <X size={16} />
        </button>
      </div>
      <div className="max-h-[70vh] overflow-y-auto p-5">{children}</div>
    </dialog>
  );
}

/** Hỏi lại trước khi xoá. Xoá là việc người dùng khó tự sửa lại. */
export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  description,
  confirmLabel = "Xoá",
  pending,
}: {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: string;
  confirmLabel?: string;
  pending?: boolean;
}) {
  return (
    <Modal open={open} onClose={onClose} title={title}>
      <p className="text-sm text-fg-muted">{description}</p>
      <div className="mt-5 flex justify-end gap-2">
        <button
          type="button"
          onClick={onClose}
          className="h-9 rounded-lg border border-line px-4 text-sm font-medium hover:bg-surface-2"
        >
          Huỷ
        </button>
        <button
          type="button"
          onClick={onConfirm}
          disabled={pending}
          className="h-9 rounded-lg bg-danger px-4 text-sm font-medium text-white disabled:opacity-50"
        >
          {pending ? "Đang xoá…" : confirmLabel}
        </button>
      </div>
    </Modal>
  );
}
