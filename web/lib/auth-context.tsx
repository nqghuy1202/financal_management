"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as authApi from "./api/auth";
import { restoreSession, setSessionLostHandler } from "./api/client";
import type { User } from "./api/types";

/**
 * Trạng thái phiên có BA giá trị chứ không phải hai.
 *
 * "loading" là trạng thái thật, không phải chi tiết kỹ thuật: khi tải lại
 * trang, access token trong bộ nhớ đã mất và ta phải hỏi backend xem cookie
 * refresh còn dùng được không. Nếu chỉ có đăng-nhập/chưa-đăng-nhập thì
 * trong lúc chờ ta buộc phải đoán là "chưa", và người dùng đang đăng nhập
 * bị đá về trang login mỗi lần bấm F5.
 */
type Status = "loading" | "authenticated" | "anonymous";

type AuthValue = {
  status: Status;
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  register: (input: { email: string; password: string; full_name: string }) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>("loading");
  const [user, setUser] = useState<User | null>(null);
  const queryClient = useQueryClient();

  const clear = useCallback(() => {
    setUser(null);
    setStatus("anonymous");
    // Xoá cache: dữ liệu của người vừa đăng xuất không được hiện ra cho
    // người đăng nhập tiếp theo trên cùng máy.
    queryClient.clear();
  }, [queryClient]);

  useEffect(() => {
    let cancelled = false;

    // Đổi cookie refresh lấy access token mới. Không có cookie thì trả null
    // — người mở trang lần đầu, không phải sự cố.
    restoreSession().then((session) => {
      if (cancelled) return;
      setUser(session?.user ?? null);
      setStatus(session ? "authenticated" : "anonymous");
    });

    // Khi refresh token hỏng giữa chừng, tầng API báo ngược lên đây.
    setSessionLostHandler(() => {
      setUser(null);
      setStatus("anonymous");
    });

    return () => {
      cancelled = true;
      setSessionLostHandler(null);
    };
  }, []);

  const value = useMemo<AuthValue>(
    () => ({
      status,
      user,
      async login(email, password) {
        const session = await authApi.login(email, password);
        queryClient.clear();
        setUser(session.user);
        setStatus("authenticated");
      },
      async register(input) {
        const session = await authApi.register(input);
        queryClient.clear();
        setUser(session.user);
        setStatus("authenticated");
      },
      async logout() {
        await authApi.logout();
        clear();
      },
    }),
    [status, user, queryClient, clear],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth phải nằm trong AuthProvider");
  return value;
}
