import { AuthGuard } from "@/components/auth-guard";
import { AppShell } from "@/components/layout/app-shell";

/**
 * Nhóm route (app) gom mọi trang cần đăng nhập.
 *
 * Guard đặt ở layout chứ không ở từng trang: thêm một trang mới sau này
 * được bảo vệ sẵn, không phải nhớ dán thêm dòng nào.
 */
export default function AppLayout({ children }: LayoutProps<"/">) {
  return (
    <AuthGuard>
      <AppShell>{children}</AppShell>
    </AuthGuard>
  );
}
