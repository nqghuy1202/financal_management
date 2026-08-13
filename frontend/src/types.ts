export type TransactionType = 'income' | 'expense'

export interface Category {
  id: string
  name: string
  type: TransactionType
  color: string
  icon: string
}

export interface Transaction {
  id: string
  type: TransactionType
  amount: number
  categoryId: string
  note: string
  date: string // ISO yyyy-mm-dd
}

export interface Budget {
  id: string
  categoryId: string
  limit: number
  month: string // yyyy-mm
}

export interface User {
  id: string
  name: string
  email: string
}
