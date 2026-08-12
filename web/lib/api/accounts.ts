import { api } from "./client";
import type { Account, AccountType, ListOf } from "./types";

export type AccountInput = {
  name: string;
  type: AccountType;
  icon: string;
};

export async function listAccounts(): Promise<Account[]> {
  const data = await api<ListOf<Account>>("/accounts");
  return data.items;
}

export function createAccount(input: AccountInput): Promise<Account> {
  return api<Account>("/accounts", { method: "POST", body: input });
}

export function updateAccount(id: string, input: AccountInput): Promise<Account> {
  return api<Account>(`/accounts/${id}`, { method: "PUT", body: input });
}

export function deleteAccount(id: string): Promise<unknown> {
  return api(`/accounts/${id}`, { method: "DELETE" });
}
