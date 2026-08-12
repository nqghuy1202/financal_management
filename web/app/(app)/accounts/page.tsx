"use client";

import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { createAccount, deleteAccount, updateAccount } from "@/lib/api/accounts";
import { ACCOUNT_TYPES, type Account, type AccountType } from "@/lib/api/types";
import { useAccounts } from "@/lib/hooks";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Field, Input, Select } from "@/components/ui/field";
import { ConfirmDialog, Modal } from "@/components/ui/modal";
import { EmptyState, ErrorState, FormError, Skeleton } from "@/components/ui/states";

const TYPE_LABELS: Record<AccountType, string> = {
  cash: "Tiền mặt",
  bank: "Ngân hàng",
  ewallet: "Ví điện tử",
  credit_card: "Thẻ tín dụng",
};

export default function AccountsPage() {
  const accounts = useAccounts();
  const queryClient = useQueryClient();

  const [editing, setEditing] = useState<Account | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [deleting, setDeleting] = useState<Account | null>(null);

  const remove = useMutation({
    mutationFn: (id: string) => deleteAccount(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["accounts"] });
      setDeleting(null);
    },
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">Nguồn tiền</h1>
          {/* Nói rõ ngay từ đầu để không ai đi tìm nút chuyển tiền: hệ
              thống cố tình không có tính năng đó, vì chuyển tiền giữa hai
              nguồn của chính mình không làm thay đổi tổng thu hay tổng chi. */}
          <p className="mt-0.5 text-sm text-fg-muted">
            Nhãn để biết khoản tiền đi qua đâu. Không theo dõi số dư từng nguồn.
          </p>
        </div>
        <Button
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus size={16} />
          Thêm nguồn tiền
        </Button>
      </div>

      {accounts.isPending ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }, (_, i) => (
            <Skeleton key={i} className="h-20" />
          ))}
        </div>
      ) : accounts.isError ? (
        <Card>
          <ErrorState error={accounts.error} onRetry={() => accounts.refetch()} />
        </Card>
      ) : accounts.data.length === 0 ? (
        <Card>
          <EmptyState
            title="Chưa có nguồn tiền nào"
            description="Ví dụ: Tiền mặt, Techcombank, Momo."
            action={
              <Button
                onClick={() => {
                  setEditing(null);
                  setFormOpen(true);
                }}
              >
                Thêm nguồn tiền
              </Button>
            }
          />
        </Card>
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {accounts.data.map((account) => (
            <li
              key={account.id}
              className="flex items-center gap-3 rounded-xl border border-line bg-surface p-4"
            >
              <span
                aria-hidden
                className="grid h-10 w-10 shrink-0 place-items-center rounded-lg bg-surface-2 text-lg"
              >
                {account.icon || "💰"}
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{account.name}</p>
                <p className="text-xs text-fg-faint">{TYPE_LABELS[account.type]}</p>
              </div>
              <div className="flex shrink-0 gap-0.5">
                <button
                  type="button"
                  onClick={() => {
                    setEditing(account);
                    setFormOpen(true);
                  }}
                  aria-label={`Sửa ${account.name}`}
                  className="rounded-md p-1.5 text-fg-faint hover:bg-surface-2 hover:text-fg"
                >
                  <Pencil size={15} />
                </button>
                <button
                  type="button"
                  onClick={() => setDeleting(account)}
                  aria-label={`Xoá ${account.name}`}
                  className="rounded-md p-1.5 text-fg-faint hover:bg-surface-2 hover:text-fg"
                >
                  <Trash2 size={15} />
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      <AccountForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        account={editing ?? undefined}
      />

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.id)}
        pending={remove.isPending}
        title={`Xoá nguồn tiền "${deleting?.name ?? ""}"?`}
        description="Nguồn tiền đang gắn với giao dịch sẽ không xoá được, để số liệu cũ không bị mất tham chiếu."
      />
    </div>
  );
}

function AccountForm({
  open,
  onClose,
  account,
}: {
  open: boolean;
  onClose: () => void;
  account?: Account;
}) {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [type, setType] = useState<AccountType>("cash");
  const [icon, setIcon] = useState("");

  useEffect(() => {
    if (!open) return;
    setName(account?.name ?? "");
    setType(account?.type ?? "cash");
    setIcon(account?.icon ?? "");
  }, [open, account]);

  const save = useMutation({
    mutationFn: () =>
      account
        ? updateAccount(account.id, { name, type, icon })
        : createAccount({ name, type, icon }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["accounts"] });
      onClose();
    },
  });

  return (
    <Modal open={open} onClose={onClose} title={account ? "Sửa nguồn tiền" : "Thêm nguồn tiền"}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate();
        }}
        className="space-y-4"
      >
        <Field label="Tên">
          {(id) => (
            <Input
              id={id}
              required
              maxLength={100}
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Techcombank"
            />
          )}
        </Field>

        <Field label="Loại">
          {(id) => (
            <Select
              id={id}
              value={type}
              onChange={(e) => setType(e.target.value as AccountType)}
            >
              {ACCOUNT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {TYPE_LABELS[t]}
                </option>
              ))}
            </Select>
          )}
        </Field>

        <Field label="Biểu tượng" hint="Một emoji, ví dụ 🏦">
          {(id) => (
            <Input
              id={id}
              maxLength={50}
              value={icon}
              onChange={(e) => setIcon(e.target.value)}
              placeholder="🏦"
            />
          )}
        </Field>

        <FormError error={save.error} />

        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>
            Huỷ
          </Button>
          <Button type="submit" disabled={save.isPending}>
            {save.isPending ? "Đang lưu…" : "Lưu"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
