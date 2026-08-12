"use client";

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { listAccounts } from "./api/accounts";
import { listCategories } from "./api/categories";
import type { Account, Category } from "./api/types";

/**
 * Danh mục và nguồn tiền đổi rất hiếm, mà gần như trang nào cũng cần để
 * đổi id thành tên. Để chung một query key nên TanStack Query chỉ gọi API
 * một lần rồi chia sẻ kết quả cho mọi component.
 */

export function useCategories() {
  return useQuery({
    queryKey: ["categories"],
    queryFn: () => listCategories(),
    staleTime: 5 * 60_000,
  });
}

export function useAccounts() {
  return useQuery({
    queryKey: ["accounts"],
    queryFn: () => listAccounts(),
    staleTime: 5 * 60_000,
  });
}

/** Bảng tra id → danh mục và id → nguồn tiền, dùng khi hiển thị giao dịch. */
export function useLookups() {
  const categories = useCategories();
  const accounts = useAccounts();

  return useMemo(
    () => ({
      categories: toMap(categories.data),
      accounts: toMap(accounts.data),
    }),
    [categories.data, accounts.data],
  );
}

function toMap<T extends { id: string }>(items: T[] | undefined): Map<string, T> {
  return new Map((items ?? []).map((item) => [item.id, item]));
}

export type Lookups = {
  categories: Map<string, Category>;
  accounts: Map<string, Account>;
};
