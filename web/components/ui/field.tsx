import clsx from "clsx";
import { useId } from "react";

const CONTROL = clsx(
  "w-full rounded-lg border border-line bg-surface px-3 py-2 text-sm text-fg",
  "placeholder:text-fg-muted disabled:opacity-50",
);

/**
 * Field bọc nhãn quanh ô nhập và tự nối chúng bằng id.
 *
 * Viết tay `htmlFor` ở từng chỗ thì sớm muộn cũng có chỗ quên, và khi đó
 * bấm vào nhãn không focus được ô nhập, còn trình đọc màn hình đọc ô đó
 * là "edit text" không tên.
 */
export function Field({
  label,
  hint,
  error,
  children,
}: {
  label: string;
  hint?: string;
  error?: string;
  children: (id: string) => React.ReactNode;
}) {
  const id = useId();

  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-sm font-medium text-fg">
        {label}
      </label>
      {children(id)}
      {hint && !error && <p className="text-xs text-fg-muted">{hint}</p>}
      {error && (
        <p role="alert" className="text-xs text-danger">
          {error}
        </p>
      )}
    </div>
  );
}

export function Input({ className, ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input className={clsx(CONTROL, className)} {...props} />;
}

export function Select({ className, ...props }: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={clsx(CONTROL, "pr-8", className)} {...props} />;
}

export function Textarea({
  className,
  ...props
}: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={clsx(CONTROL, "resize-y", className)} {...props} />;
}
