"use client";

import { useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { deleteTransaction, listTransactions, type TransactionFilter } from "@/lib/api/transactions";
import type { Transaction } from "@/lib/api/types";
import { useAccounts, useCategories } from "@/lib/hooks";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input, Select } from "@/components/ui/field";
import { ConfirmDialog } from "@/components/ui/modal";
import { EmptyState, ErrorState, Skeleton } from "@/components/ui/states";
import { TransactionForm } from "@/components/transactions/transaction-form";
import { TransactionRow } from "@/components/transactions/transaction-row";

const PAGE_SIZE = 20;

export default function TransactionsPage() {
  const queryClient = useQueryClient();
  const categories = useCategories();
  const accounts = useAccounts();

  const [filter, setFilter] = useState<TransactionFilter>({});
  const [editing, setEditing] = useState<Transaction | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [deleting, setDeleting] = useState<Transaction | null>(null);

  /**
   * Phân trang bằng con trỏ chứ không phải số trang.
   *
   * Backend trả về `next_cursor`; ta gửi lại nguyên văn để lấy trang kế.
   * Nhờ vậy một giao dịch mới được thêm giữa hai lần bấm "Tải thêm" cũng
   * không đẩy các dòng cũ trôi sang trang sau, thứ mà OFFSET luôn mắc phải.
   */
  const list = useInfiniteQuery({
    queryKey: ["transactions", "list", filter],
    queryFn: ({ pageParam }) =>
      listTransactions({ ...filter, page_size: PAGE_SIZE, cursor: pageParam }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteTransaction(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      queryClient.invalidateQueries({ queryKey: ["summary"] });
      queryClient.invalidateQueries({ queryKey: ["by-category"] });
      queryClient.invalidateQueries({ queryKey: ["cash-flow"] });
      setDeleting(null);
    },
  });

  const items = list.data?.pages.flatMap((page) => page.items) ?? [];
  const total = list.data?.pages[0]?.total ?? 0;
  const filtered = Object.values(filter).some(Boolean);

  function openNew() {
    setEditing(null);
    setFormOpen(true);
  }

  function openEdit(transaction: Transaction) {
    setEditing(transaction);
    setFormOpen(true);
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">Giao dịch</h1>
          {!list.isPending && (
            <p className="mt-0.5 text-sm text-fg-muted">{total} giao dịch khớp bộ lọc</p>
          )}
        </div>
        <Button onClick={openNew}>
          <Plus size={16} />
          Thêm giao dịch
        </Button>
      </div>

      {/* Bộ lọc nằm trên một hàng ngay phía trên danh sách, để thấy ngay
          cái gì đang lọc mà không phải mở panel. */}
      <Card className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        <Field label="Loại">
          {(id) => (
            <Select
              id={id}
              value={filter.type ?? ""}
              onChange={(e) =>
                setFilter((f) => ({ ...f, type: e.target.value as TransactionFilter["type"] }))
              }
            >
              <option value="">Tất cả</option>
              <option value="expense">Khoản chi</option>
              <option value="income">Khoản thu</option>
            </Select>
          )}
        </Field>

        <Field label="Danh mục">
          {(id) => (
            <Select
              id={id}
              value={filter.category_id ?? ""}
              onChange={(e) => setFilter((f) => ({ ...f, category_id: e.target.value }))}
            >
              <option value="">Tất cả</option>
              {(categories.data ?? [])
                .filter((c) => !filter.type || c.type === filter.type)
                .map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.icon} {c.name}
                  </option>
                ))}
            </Select>
          )}
        </Field>

        <Field label="Nguồn tiền">
          {(id) => (
            <Select
              id={id}
              value={filter.account_id ?? ""}
              onChange={(e) => setFilter((f) => ({ ...f, account_id: e.target.value }))}
            >
              <option value="">Tất cả</option>
              {(accounts.data ?? []).map((a) => (
                <option key={a.id} value={a.id}>
                  {a.icon} {a.name}
                </option>
              ))}
            </Select>
          )}
        </Field>

        <Field label="Từ ngày">
          {(id) => (
            <Input
              id={id}
              type="date"
              value={filter.from ?? ""}
              onChange={(e) => setFilter((f) => ({ ...f, from: e.target.value }))}
            />
          )}
        </Field>

        {/* Backend hiểu khoảng [from, to) — "đến ngày" không bao gồm chính
            ngày đó, nên nói rõ ra thay vì để người dùng đoán. */}
        <Field label="Đến ngày" hint="Không tính ngày này">
          {(id) => (
            <Input
              id={id}
              type="date"
              value={filter.to ?? ""}
              onChange={(e) => setFilter((f) => ({ ...f, to: e.target.value }))}
            />
          )}
        </Field>
      </Card>

      <Card className="p-2 sm:p-3">
        {list.isPending ? (
          <div className="space-y-2 p-2">
            {Array.from({ length: 6 }, (_, i) => (
              <Skeleton key={i} className="h-14" />
            ))}
          </div>
        ) : list.isError ? (
          <ErrorState error={list.error} onRetry={() => list.refetch()} />
        ) : items.length === 0 ? (
          <EmptyState
            title={filtered ? "Không có giao dịch nào khớp" : "Chưa có giao dịch nào"}
            description={
              filtered
                ? "Thử nới bộ lọc, ví dụ bỏ khoảng ngày."
                : "Ghi lại khoản thu hoặc chi đầu tiên để bắt đầu."
            }
            action={
              filtered ? (
                <Button variant="secondary" onClick={() => setFilter({})}>
                  Xoá bộ lọc
                </Button>
              ) : (
                <Button onClick={openNew}>Thêm giao dịch</Button>
              )
            }
          />
        ) : (
          <>
            <ul className="divide-y divide-line">
              {items.map((tx) => (
                <li key={tx.id} className="flex items-center gap-1">
                  <div className="min-w-0 flex-1">
                    <TransactionRow transaction={tx} onClick={() => openEdit(tx)} />
                  </div>
                  <div className="flex shrink-0 gap-0.5">
                    <IconAction label="Sửa" onClick={() => openEdit(tx)}>
                      <Pencil size={16} />
                    </IconAction>
                    <IconAction label="Xoá" onClick={() => setDeleting(tx)}>
                      <Trash2 size={16} />
                    </IconAction>
                  </div>
                </li>
              ))}
            </ul>

            {list.hasNextPage && (
              <div className="p-3 text-center">
                <Button
                  variant="secondary"
                  onClick={() => list.fetchNextPage()}
                  disabled={list.isFetchingNextPage}
                >
                  {list.isFetchingNextPage ? "Đang tải…" : "Tải thêm"}
                </Button>
              </div>
            )}
          </>
        )}
      </Card>

      <TransactionForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        transaction={editing ?? undefined}
      />

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.id)}
        pending={remove.isPending}
        title="Xoá giao dịch?"
        description="Giao dịch sẽ biến mất khỏi mọi báo cáo của bạn. Thao tác này không hoàn tác được."
      />
    </div>
  );
}

function IconAction({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      className="rounded-md p-2 text-fg-faint hover:bg-surface-2 hover:text-fg"
    >
      {children}
    </button>
  );
}
