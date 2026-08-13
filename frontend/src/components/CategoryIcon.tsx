import {
  Wallet,
  Gift,
  TrendingUp,
  Utensils,
  Car,
  ShoppingBag,
  Receipt,
  HeartPulse,
  Gamepad2,
  MoreHorizontal,
  Landmark,
  Plane,
  GraduationCap,
  Home,
  Tag,
  Coffee,
  Dumbbell,
  type LucideIcon,
} from 'lucide-react'

const ICONS: Record<string, LucideIcon> = {
  Wallet,
  Gift,
  TrendingUp,
  Utensils,
  Car,
  ShoppingBag,
  Receipt,
  HeartPulse,
  Gamepad2,
  MoreHorizontal,
  Landmark,
  Plane,
  GraduationCap,
  Home,
  Tag,
  Coffee,
  Dumbbell,
}

export const iconNames = Object.keys(ICONS)

export function CategoryIcon({
  name,
  color,
  size = 18,
  className,
}: {
  name: string
  color?: string
  size?: number
  className?: string
}) {
  const Icon = ICONS[name] ?? MoreHorizontal
  return <Icon size={size} color={color} className={className} strokeWidth={2} />
}
