import { api } from "./client";
import type { Transaction, TransactionInput, TransactionPage, TransactionType } from "./types";

export type TransactionFilter = {
  type?: TransactionType | "";
  category_id?: string;
  account_id?: string;
  /** Dạng ngày "2026-08-01"; backend hiểu theo múi giờ ứng dụng. */
  from?: string;
  to?: string;
  page_size?: number;
  /** Nguyên văn `next_cursor` của trang trước. */
  cursor?: string;
};

export function listTransactions(filter: TransactionFilter): Promise<TransactionPage> {
  return api<TransactionPage>("/transactions", { query: filter });
}

export function getTransaction(id: string): Promise<Transaction> {
  return api<Transaction>(`/transactions/${id}`);
}

export function createTransaction(input: TransactionInput): Promise<Transaction> {
  return api<Transaction>("/transactions", { method: "POST", body: input });
}

export function updateTransaction(id: string, input: TransactionInput): Promise<Transaction> {
  return api<Transaction>(`/transactions/${id}`, { method: "PUT", body: input });
}

export function deleteTransaction(id: string): Promise<unknown> {
  return api(`/transactions/${id}`, { method: "DELETE" });
}
