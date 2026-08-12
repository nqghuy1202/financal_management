"use client";

import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Lock, Pencil, Plus, Trash2 } from "lucide-react";
import {
  createCategory,
  deleteCategory,
  updateCategory,
} from "@/lib/api/categories";
import type { Category, TransactionType } from "@/lib/api/types";
import { useCategories } from "@/lib/hooks";
import { Button } from "@/components/ui/button";
import { Card, CardTitle } from "@/components/ui/card";
import { Field, Input, Select } from "@/components/ui/field";
import { ConfirmDialog, Modal } from "@/components/ui/modal";
import { EmptyState, ErrorState, FormError, Skeleton } from "@/components/ui/states";

export default function CategoriesPage() {
  const categories = useCategories();
  const [editing, setEditing] = useState<Category | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [deleting, setDeleting] = useState<Category | null>(null);

  const queryClient = useQueryClient();
  const remove = useMutation({
    mutationFn: (id: string) => deleteCategory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["categories"] });
      setDeleting(null);
    },
  });

  const byType = (type: TransactionType) =>
    (categories.data ?? []).filter((c) => c.type === type);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold">Danh mục</h1>
          <p className="mt-0.5 text-sm text-fg-muted">
            Danh mục quyết định báo cáo &ldquo;tiền đi đâu&rdquo; được nhóm như thế nào.
          </p>
        </div>
        <Button
          onClick={() => {
            setEditing(null);
            setFormOpen(true);
          }}
        >
          <Plus size={16} />
          Thêm danh mục
        </Button>
      </div>

      {categories.isPending ? (
        <div className="grid gap-4 lg:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      ) : categories.isError ? (
        <Card>
          <ErrorState error={categories.error} onRetry={() => categories.refetch()} />
        </Card>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          <CategoryList
            title="Khoản chi"
            items={byType("expense")}
            onEdit={(c) => {
              setEditing(c);
              setFormOpen(true);
            }}
            onDelete={setDeleting}
          />
          <CategoryList
            title="Khoản thu"
            items={byType("income")}
            onEdit={(c) => {
              setEditing(c);
              setFormOpen(true);
            }}
            onDelete={setDeleting}
          />
        </div>
      )}

      <CategoryForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        category={editing ?? undefined}
      />

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.id)}
        pending={remove.isPending}
        title={`Xoá danh mục "${deleting?.name ?? ""}"?`}
        description="Danh mục đang được giao dịch sử dụng sẽ không xoá được — hãy chuyển các giao dịch đó sang danh mục khác trước."
      />
    </div>
  );
}

function CategoryList({
  title,
  items,
  onEdit,
  onDelete,
}: {
  title: string;
  items: Category[];
  onEdit: (category: Category) => void;
  onDelete: (category: Category) => void;
}) {
  return (
    <Card>
      <CardTitle>{title}</CardTitle>
      {items.length === 0 ? (
        <EmptyState title="Chưa có danh mục nào" />
      ) : (
        <ul className="divide-y divide-line">
          {items.map((category) => (
            <li key={category.id} className="flex items-center gap-3 py-2.5">
              <span aria-hidden className="text-base">
                {category.icon || "•"}
              </span>
              <span className="min-w-0 flex-1 truncate text-sm">{category.name}</span>

              {category.is_system ? (
                // Danh mục hệ thống backend không cho sửa hay xoá. Hiện
                // biểu tượng khoá thay vì để nút bấm được rồi báo lỗi.
                <span
                  title="Danh mục mặc định của hệ thống"
                  className="flex items-center gap-1 text-xs text-fg-faint"
                >
                  <Lock size={12} aria-hidden />
                  Mặc định
                </span>
              ) : (
                <span className="flex gap-0.5">
                  <button
                    type="button"
                    onClick={() => onEdit(category)}
                    aria-label={`Sửa ${category.name}`}
                    className="rounded-md p-1.5 text-fg-faint hover:bg-surface-2 hover:text-fg"
                  >
                    <Pencil size={15} />
                  </button>
                  <button
                    type="button"
                    onClick={() => onDelete(category)}
                    aria-label={`Xoá ${category.name}`}
                    className="rounded-md p-1.5 text-fg-faint hover:bg-surface-2 hover:text-fg"
                  >
                    <Trash2 size={15} />
                  </button>
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}

function CategoryForm({
  open,
  onClose,
  category,
}: {
  open: boolean;
  onClose: () => void;
  category?: Category;
}) {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [type, setType] = useState<TransactionType>("expense");
  const [icon, setIcon] = useState("");
  const [color, setColor] = useState("#2a78d6");

  useEffect(() => {
    if (!open) return;
    setName(category?.name ?? "");
    setType(category?.type ?? "expense");
    setIcon(category?.icon ?? "");
    setColor(category?.color || "#2a78d6");
  }, [open, category]);

  const save = useMutation({
    mutationFn: () =>
      category
        ? updateCategory(category.id, { name, icon, color })
        : createCategory({ name, type, icon, color }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["categories"] });
      // Báo cáo theo danh mục hiển thị tên và biểu tượng lấy từ đây.
      queryClient.invalidateQueries({ queryKey: ["by-category"] });
      onClose();
    },
  });

  return (
    <Modal open={open} onClose={onClose} title={category ? "Sửa danh mục" : "Thêm danh mục"}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate();
        }}
        className="space-y-4"
      >
        <Field label="Tên danh mục">
          {(id) => (
            <Input
              id={id}
              required
              maxLength={100}
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Ăn uống"
            />
          )}
        </Field>

        {/* Sửa danh mục không đổi được loại: mọi giao dịch cũ đã gắn với
            nó sẽ nhảy từ cột chi sang cột thu, làm sai toàn bộ báo cáo. */}
        <Field
          label="Loại"
          hint={category ? "Không đổi được loại sau khi đã tạo." : undefined}
        >
          {(id) => (
            <Select
              id={id}
              value={type}
              disabled={Boolean(category)}
              onChange={(e) => setType(e.target.value as TransactionType)}
            >
              <option value="expense">Khoản chi</option>
              <option value="income">Khoản thu</option>
            </Select>
          )}
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Biểu tượng" hint="Một emoji, ví dụ 🍜">
            {(id) => (
              <Input
                id={id}
                maxLength={50}
                value={icon}
                onChange={(e) => setIcon(e.target.value)}
                placeholder="🍜"
              />
            )}
          </Field>
          <Field label="Màu">
            {(id) => (
              <Input
                id={id}
                type="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                className="h-10 p-1"
              />
            )}
          </Field>
        </div>

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
