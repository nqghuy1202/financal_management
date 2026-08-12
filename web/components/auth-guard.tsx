"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { Skeleton } from "@/components/ui/states";

/**
 * Chặn route ở phía trình duyệt, không phải ở proxy.ts.
 *
 * Next.js chạy proxy trên server của nó, mà server đó không nhìn thấy
 * phiên: access token chỉ nằm trong bộ nhớ tab, còn cookie refresh bị
 * giới hạn ở đường dẫn /api/v1/auth của API nên trình duyệt không gửi
 * kèm khi tải trang Next. Một proxy đọc cookie ở đây sẽ luôn thấy trống
 * và đá cả người đang đăng nhập ra ngoài.
 *
 * Điều đó không làm dữ liệu kém an toàn: mọi endpoint đều tự kiểm tra
 * token ở backend. Guard này chỉ để người dùng không nhìn thấy một màn
 * hình rỗng rồi mới bị chuyển hướng.
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (status === "anonymous") router.replace("/login");
  }, [status, router]);

  if (status === "authenticated") return <>{children}</>;

  // Cả lúc đang kiểm tra phiên lẫn lúc đang chuyển hướng đều hiện khung
  // chờ, để không loé lên nội dung của trang mình sắp rời đi.
  return <ShellSkeleton />;
}

/** Ngược lại: người đã đăng nhập không cần nhìn thấy trang đăng nhập. */
export function GuestGuard({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (status === "authenticated") router.replace("/");
  }, [status, router]);

  if (status === "anonymous") return <>{children}</>;
  return null;
}

function ShellSkeleton() {
  return (
    <div className="mx-auto max-w-6xl space-y-4 p-6">
      <Skeleton className="h-8 w-48" />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
      <Skeleton className="h-64" />
    </div>
  );
}
