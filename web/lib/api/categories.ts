import { api } from "./client";
import type { Category, ListOf, TransactionType } from "./types";

export type CreateCategoryInput = {
  name: string;
  type: TransactionType;
  icon: string;
  color: string;
};

/** Sửa danh mục không đổi được `type`: đổi loại sẽ làm sai mọi báo cáo cũ. */
export type UpdateCategoryInput = {
  name: string;
  icon: string;
  color: string;
};

export async function listCategories(type?: TransactionType): Promise<Category[]> {
  const data = await api<ListOf<Category>>("/categories", { query: { type } });
  return data.items;
}

export function createCategory(input: CreateCategoryInput): Promise<Category> {
  return api<Category>("/categories", { method: "POST", body: input });
}

export function updateCategory(id: string, input: UpdateCategoryInput): Promise<Category> {
  return api<Category>(`/categories/${id}`, { method: "PUT", body: input });
}

export function deleteCategory(id: string): Promise<unknown> {
  return api(`/categories/${id}`, { method: "DELETE" });
}
