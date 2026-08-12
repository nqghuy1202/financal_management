"use client";

import clsx from "clsx";
import type { Transaction } from "@/lib/api/types";
import { useLookups } from "@/lib/hooks";
import { formatMoney } from "@/lib/money";
import { formatDate } from "@/lib/time";

/**
 * Nội dung một dòng giao dịch. Nơi gọi tự bọc trong <li> của mình, để
 * component này dùng được cả trong danh sách lẫn ngoài danh sách.
 *
 * Dấu + hoặc − ở đầu số tiền là tín hiệu CHÍNH cho biết đây là thu hay
 * chi; màu chỉ nhấn thêm. Nếu chỉ dựa vào màu thì người không phân biệt
 * được xanh với đỏ đọc bảng này thành một dãy số không dấu.
 */
export function TransactionRow({
  transaction,
  onClick,
}: {
  transaction: Transaction;
  onClick?: () => void;
}) {
  const { categories, accounts } = useLookups();

  const category = transaction.category_id ? categories.get(transaction.category_id) : undefined;
  const account = transaction.account_id ? accounts.get(transaction.account_id) : undefined;
  const income = transaction.type === "income";

  const content = (
    <>
      <span
        aria-hidden
        className="grid h-9 w-9 shrink-0 place-items-center rounded-lg bg-surface-2 text-base"
      >
        {category?.icon || (income ? "↓" : "↑")}
      </span>

      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm text-fg">
          {category?.name ?? "Chưa phân loại"}
        </span>
        <span className="block truncate text-xs text-fg-faint">
          {[formatDate(transaction.occurred_at), account?.name, transaction.note]
            .filter(Boolean)
            .join(" · ")}
        </span>
      </span>

      <span
        className={clsx(
          "shrink-0 tabular text-sm font-medium",
          income ? "text-income" : "text-expense",
        )}
      >
        {income ? "+" : "−"}
        {formatMoney(transaction.amount)}
        <span className="sr-only">{income ? " (khoản thu)" : " (khoản chi)"}</span>
      </span>
    </>
  );

  if (!onClick) {
    return <div className="flex items-center gap-3 py-3">{content}</div>;
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className="flex w-full items-center gap-3 rounded-lg px-2 py-3 text-left hover:bg-surface-2"
    >
      {content}
    </button>
  );
}
