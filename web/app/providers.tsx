"use client";

import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "@/lib/auth-context";
import { ApiError } from "@/lib/api/client";

/**
 * Toàn bộ dữ liệu được lấy ở phía trình duyệt, không phải trên server
 * Next.js.
 *
 * Lý do nằm ở thiết kế token của backend: access token chỉ sống trong bộ
 * nhớ tab, còn cookie refresh bị giới hạn ở đường dẫn /api/v1/auth của
 * API. Server Next không cầm được cái nào trong hai thứ đó, nên nó không
 * thể gọi API thay người dùng. Cố render dữ liệu trên server sẽ chỉ nhận
 * về 401.
 */
export function Providers({ children }: { children: React.ReactNode }) {
  // Tạo trong useState để mỗi lần render lại không dựng client mới, làm
  // mất sạch cache.
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // Số liệu tài chính không đổi từng giây; một phút là đủ tươi
            // mà tránh gọi lại API mỗi lần chuyển tab.
            staleTime: 60_000,
            retry(failureCount, error) {
              // Lỗi nghiệp vụ thì thử lại vô ích: sai tham số vẫn sai,
              // không tìm thấy vẫn không tìm thấy. Chỉ thử lại lỗi mạng
              // hoặc lỗi phía server.
              if (error instanceof ApiError && error.status < 500 && error.status !== 0) {
                return false;
              }
              return failureCount < 2;
            },
          },
          mutations: {
            retry: false,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>{children}</AuthProvider>
    </QueryClientProvider>
  );
}
