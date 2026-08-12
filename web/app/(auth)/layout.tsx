import { GuestGuard } from "@/components/auth-guard";

export default function AuthLayout({ children }: LayoutProps<"/">) {
  return (
    <GuestGuard>
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="w-full max-w-sm">
          <div className="mb-8 flex items-center justify-center gap-2">
            <span className="grid h-9 w-9 place-items-center rounded-lg bg-brand font-bold text-brand-fg">
              F
            </span>
            <span className="text-lg font-semibold">FinTrack</span>
          </div>
          {children}
        </div>
      </div>
    </GuestGuard>
  );
}
