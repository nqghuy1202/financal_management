"use client";

import { useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/field";
import { FormError } from "@/components/ui/states";

export default function LoginPage() {
  const { login } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<unknown>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setPending(true);
    try {
      await login(email, password);
      // Không tự chuyển trang ở đây: GuestGuard thấy trạng thái đổi sang
      // "authenticated" sẽ đưa về trang tổng quan. Một nơi quyết định
      // điều hướng thay vì hai nơi tranh nhau.
    } catch (err) {
      setError(err);
      setPending(false);
    }
  }

  return (
    <form
      onSubmit={submit}
      className="space-y-4 rounded-xl border border-line bg-surface p-6"
    >
      <div>
        <h1 className="text-lg font-semibold">Đăng nhập</h1>
        <p className="mt-1 text-sm text-fg-muted">Tiếp tục theo dõi thu chi của bạn.</p>
      </div>

      <Field label="Email">
        {(id) => (
          <Input
            id={id}
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="ban@example.com"
          />
        )}
      </Field>

      <Field label="Mật khẩu">
        {(id) => (
          <Input
            id={id}
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        )}
      </Field>

      <FormError error={error} />

      <Button type="submit" disabled={pending} className="w-full">
        {pending ? "Đang đăng nhập…" : "Đăng nhập"}
      </Button>

      <p className="text-center text-sm text-fg-muted">
        Chưa có tài khoản?{" "}
        <Link href="/register" className="font-medium text-brand underline underline-offset-4">
          Đăng ký
        </Link>
      </p>
    </form>
  );
}
