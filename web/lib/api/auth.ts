import { api, setAccessToken } from "./client";
import type { Session, User } from "./types";

/**
 * Mọi endpoint auth đều đặt skipRefresh: 401 ở đây nghĩa là sai mật khẩu
 * hoặc refresh token hỏng, thử refresh thêm một vòng chỉ tốn công.
 */

export async function login(email: string, password: string): Promise<Session> {
  const session = await api<Session>("/auth/login", {
    method: "POST",
    body: { email, password },
    skipRefresh: true,
  });
  setAccessToken(session.access_token);
  return session;
}

export async function register(input: {
  email: string;
  password: string;
  full_name: string;
}): Promise<Session> {
  const session = await api<Session>("/auth/register", {
    method: "POST",
    body: input,
    skipRefresh: true,
  });
  setAccessToken(session.access_token);
  return session;
}

/** Backend luôn trả thành công, kể cả khi không còn cookie. */
export async function logout(): Promise<void> {
  try {
    await api("/auth/logout", { method: "POST", skipRefresh: true });
  } finally {
    // Xoá token phía client dù API có lỗi mạng: người dùng đã bấm đăng
    // xuất thì màn hình phải thoát, không thể kẹt lại vì server bận.
    setAccessToken(null);
  }
}

export function me(): Promise<User> {
  return api<User>("/auth/me");
}
