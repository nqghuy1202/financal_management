"use client";

import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import clsx from "clsx";
import { createTransaction, updateTransaction } from "@/lib/api/transactions";
import type { Transaction, TransactionType } from "@/lib/api/types";
import { useAccounts, useCategories } from "@/lib/hooks";
import { inputToRFC3339, nowLocalInput, rfc3339ToInput } from "@/lib/time";
import { Button } from "@/components/ui/button";
import { Field, Input, Select, Textarea } from "@/components/ui/field";
import { Modal } from "@/components/ui/modal";
import { FormError } from "@/components/ui/states";

/**
 * Số tiền là chuỗi từ đầu tới cuối.
 *
 * Ô nhập để type="text" chứ không phải type="number": trình duyệt đọc
 * giá trị của ô số thành float64, nên "9007199254740993" bị làm tròn
 * ngay trong ô nhập, trước cả khi ta gửi đi. inputMode="decimal" vẫn cho
 * bàn phím số hiện lên trên điện thoại.
 */
const AMOUNT_PATTERN = /^\d+([.,]\d{1,4})?$/;

type FormState = {
  type: TransactionType;
  amount: string;
  category_id: string;
  account_id: string;
  occurred_at: string;
  note: string;
};

function emptyForm(): FormState {
  return {
    type: "expense",
    amount: "",
    category_id: "",
    account_id: "",
    occurred_at: nowLocalInput(),
    note: "",
  };
}

function formFrom(tx: Transaction): FormState {
  return {
    type: tx.type,
    amount: tx.amount.amount,
    category_id: tx.category_id ?? "",
    account_id: tx.account_id ?? "",
    occurred_at: rfc3339ToInput(tx.occurred_at),
    note: tx.note,
  };
}

export function TransactionForm({
  open,
  onClose,
  transaction,
}: {
  open: boolean;
  onClose: () => void;
  /** Có thì là sửa, không có thì là thêm mới. */
  transaction?: Transaction;
}) {
  const queryClient = useQueryClient();
  const categories = useCategories();
  const accounts = useAccounts();

  const [form, setForm] = useState<FormState>(emptyForm);
  const [amountError, setAmountError] = useState<string | null>(null);

  // Nạp lại giá trị mỗi lần mở: mở form sửa của giao dịch khác mà vẫn
  // giữ state cũ thì người dùng sửa nhầm sang số của giao dịch trước.
  useEffect(() => {
    if (!open) return;
    setForm(transaction ? formFrom(transaction) : emptyForm());
    setAmountError(null);
  }, [open, transaction]);

  const save = useMutation({
    mutationFn: (state: FormState) => {
      const input = {
        type: state.type,
        // Người Việt gõ dấu phẩy làm dấu thập phân; backend chờ dấu chấm.
        amount: state.amount.replace(",", "."),
        category_id: state.category_id || null,
        account_id: state.account_id || null,
        note: state.note,
        occurred_at: inputToRFC3339(state.occurred_at),
      };
      return transaction ? updateTransaction(transaction.id, input) : createTransaction(input);
    },
    onSuccess: () => {
      // Một giao dịch mới làm sai lệch cả danh sách lẫn mọi báo cáo, nên
      // xoá hiệu lực toàn bộ thay vì cố sửa từng cache một.
      queryClient.invalidateQueries({ queryKey: ["transactions"] });
      queryClient.invalidateQueries({ queryKey: ["summary"] });
      queryClient.invalidateQueries({ queryKey: ["by-category"] });
      queryClient.invalidateQueries({ queryKey: ["cash-flow"] });
      onClose();
    },
  });

  function submit(event: React.FormEvent) {
    event.preventDefault();

    if (!AMOUNT_PATTERN.test(form.amount.trim())) {
      setAmountError("Số tiền phải là số dương, tối đa 4 chữ số thập phân");
      return;
    }
    setAmountError(null);
    save.mutate(form);
  }

  // Danh mục thu và danh mục chi là hai tập riêng: chọn "Lương" cho một
  // khoản chi sẽ làm hỏng mọi báo cáo theo danh mục.
  const options = (categories.data ?? []).filter((c) => c.type === form.type);

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={transaction ? "Sửa giao dịch" : "Thêm giao dịch"}
    >
      <form onSubmit={submit} className="space-y-4">
        <TypeToggle
          value={form.type}
          onChange={(type) =>
            // Đổi loại thì bỏ danh mục đã chọn, vì nó thuộc loại cũ.
            setForm((f) => ({ ...f, type, category_id: "" }))
          }
        />

        <Field label="Số tiền" error={amountError ?? undefined}>
          {(id) => (
            <Input
              id={id}
              inputMode="decimal"
              autoFocus
              required
              placeholder="0"
              value={form.amount}
              onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))}
              className="tabular text-lg"
            />
          )}
        </Field>

        <Field label="Danh mục">
          {(id) => (
            <Select
              id={id}
              value={form.category_id}
              onChange={(e) => setForm((f) => ({ ...f, category_id: e.target.value }))}
            >
              <option value="">Chưa phân loại</option>
              {options.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.icon} {category.name}
                </option>
              ))}
            </Select>
          )}
        </Field>

        <Field label="Nguồn tiền">
          {(id) => (
            <Select
              id={id}
              value={form.account_id}
              onChange={(e) => setForm((f) => ({ ...f, account_id: e.target.value }))}
            >
              <option value="">Không ghi nguồn</option>
              {(accounts.data ?? []).map((account) => (
                <option key={account.id} value={account.id}>
                  {account.icon} {account.name}
                </option>
              ))}
            </Select>
          )}
        </Field>

        <Field label="Thời điểm" hint="Ngày khoản tiền thực sự phát sinh.">
          {(id) => (
            <Input
              id={id}
              type="datetime-local"
              required
              value={form.occurred_at}
              onChange={(e) => setForm((f) => ({ ...f, occurred_at: e.target.value }))}
            />
          )}
        </Field>

        <Field label="Ghi chú">
          {(id) => (
            <Textarea
              id={id}
              rows={2}
              maxLength={500}
              value={form.note}
              onChange={(e) => setForm((f) => ({ ...f, note: e.target.value }))}
              placeholder="Ăn trưa với đồng nghiệp"
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

function TypeToggle({
  value,
  onChange,
}: {
  value: TransactionType;
  onChange: (type: TransactionType) => void;
}) {
  return (
    <div role="group" aria-label="Loại giao dịch" className="grid grid-cols-2 gap-2">
      {(["expense", "income"] as const).map((type) => (
        <button
          key={type}
          type="button"
          onClick={() => onChange(type)}
          aria-pressed={value === type}
          className={clsx(
            "h-10 rounded-lg border text-sm font-medium transition-colors",
            value === type
              ? "border-brand bg-brand-soft text-brand-on-soft"
              : "border-line text-fg-muted hover:bg-surface-2",
          )}
        >
          {type === "expense" ? "Khoản chi" : "Khoản thu"}
        </button>
      ))}
    </div>
  );
}
