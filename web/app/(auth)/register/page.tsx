"use client";

import { useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { Button } from "@/components/ui/button";
import { Field, Input } from "@/components/ui/field";
import { FormError } from "@/components/ui/states";

/** Khớp với ràng buộc của backend: min=8, max=72 (giới hạn của bcrypt). */
const MIN_PASSWORD = 8;

export default function RegisterPage() {
  const { register } = useAuth();
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<unknown>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setPending(true);
    try {
      await register({ email, password, full_name: fullName });
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
        <h1 className="text-lg font-semibold">Tạo tài khoản</h1>
        <p className="mt-1 text-sm text-fg-muted">
          Bắt đầu ghi lại thu chi trong chưa tới một phút.
        </p>
      </div>

      <Field label="Họ và tên">
        {(id) => (
          <Input
            id={id}
            required
            maxLength={100}
            autoComplete="name"
            value={fullName}
            onChange={(e) => setFullName(e.target.value)}
            placeholder="Nguyễn Văn A"
          />
        )}
      </Field>

      <Field label="Email">
        {(id) => (
          <Input
            id={id}
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="ban@example.com"
          />
        )}
      </Field>

      <Field label="Mật khẩu" hint={`Tối thiểu ${MIN_PASSWORD} ký tự.`}>
        {(id) => (
          <Input
            id={id}
            type="password"
            required
            minLength={MIN_PASSWORD}
            maxLength={72}
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        )}
      </Field>

      <FormError error={error} />

      <Button type="submit" disabled={pending} className="w-full">
        {pending ? "Đang tạo tài khoản…" : "Đăng ký"}
      </Button>

      <p className="text-center text-sm text-fg-muted">
        Đã có tài khoản?{" "}
        <Link href="/login" className="font-medium text-brand underline underline-offset-4">
          Đăng nhập
        </Link>
      </p>
    </form>
  );
}
