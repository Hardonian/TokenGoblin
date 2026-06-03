import { type ReactNode } from "react";

export function Card({
  children,
  className = "",
  featured = false,
  ...props
}: {
  children: ReactNode;
  className?: string;
  featured?: boolean;
} & React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`
        bg-white p-8
        ${featured ? "border-2 border-[#426b51] relative shadow-lg shadow-[#426b51]/5" : "border border-[#d7dccf]"}
        ${className}
      `}
      {...props}
    >
      {children}
    </div>
  );
}

export function Badge({
  children,
  variant = "default",
}: {
  children: ReactNode;
  variant?: "default" | "accent" | "warning" | "danger" | "success";
}) {
  const styles = {
    default: "bg-[#edf0e8] text-[#61705a]",
    accent: "bg-[#d7dccf] text-[#2d4a3e]",
    warning: "bg-[#fff3cd] text-[#b38600]",
    danger: "bg-[#f8d7da] text-[#9f2f2f]",
    success: "bg-[#d4edda] text-[#2d6a4f]",
  };
  return (
    <span
      className={`inline-block px-3 py-1 text-xs font-semibold tracking-wider uppercase rounded ${styles[variant]}`}
    >
      {children}
    </span>
  );
}

export function Button({
  children,
  variant = "primary",
  size = "md",
  className = "",
  disabled = false,
  ...props
}: {
  children: ReactNode;
  variant?: "primary" | "secondary" | "outline" | "danger" | "ghost";
  size?: "sm" | "md" | "lg";
  className?: string;
  disabled?: boolean;
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const base =
    "inline-flex items-center justify-center font-semibold transition-all duration-150 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed";

  const sizes = {
    sm: "px-3 py-1.5 text-xs",
    md: "px-5 py-2.5 text-sm",
    lg: "px-8 py-3.5 text-base",
  };

  const variants = {
    primary: "bg-[#171915] text-white border border-[#171915] hover:bg-[#2a2d28]",
    secondary:
      "bg-[#426b51] text-white border border-[#426b51] hover:bg-[#2d6a4f]",
    outline:
      "bg-white text-[#171915] border border-[#c5cdbb] hover:bg-[#f5f7f0]",
    danger:
      "bg-white text-[#9f2f2f] border border-[#9f2f2f] hover:bg-[#9f2f2f] hover:text-white",
    ghost:
      "bg-transparent text-[#52604e] border border-transparent hover:bg-[#edf0e8]",
  };

  return (
    <button
      className={`${base} ${sizes[size]} ${variants[variant]} ${className}`}
      disabled={disabled}
      {...props}
    >
      {children}
    </button>
  );
}

export function Input({
  label,
  error,
  className = "",
  ...props
}: {
  label?: string;
  error?: string;
  className?: string;
} & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <div className={`flex flex-col gap-1.5 ${className}`}>
      {label && (
        <label className="text-sm font-medium text-[#171915]">{label}</label>
      )}
      <input
        className={`
          h-11 px-4 text-sm border bg-white outline-none transition-colors
          ${error ? "border-[#9f2f2f] focus:border-[#9f2f2f]" : "border-[#c5cdbb] focus:border-[#426b51]"}
        `}
        {...props}
      />
      {error && <p className="text-xs text-[#9f2f2f]">{error}</p>}
    </div>
  );
}

export function AlertBox({
  variant = "info",
  title,
  children,
}: {
  variant?: "info" | "warning" | "error" | "success";
  title?: string;
  children: ReactNode;
}) {
  const styles = {
    info: "border-[#c5cdbb] bg-[#fbfcf8]",
    warning: "border-[#b38600] bg-[#fffbec]",
    error: "border-[#9f2f2f] bg-[#fef2f2]",
    success: "border-[#2d6a4f] bg-[#f0f9f4]",
  };
  const iconStyles = {
    info: "text-[#426b51]",
    warning: "text-[#b38600]",
    error: "text-[#9f2f2f]",
    success: "text-[#2d6a4f]",
  };
  const icons = {
    info: "ℹ",
    warning: "⚠",
    error: "✕",
    success: "✓",
  };
  return (
    <div className={`border p-4 ${styles[variant]}`}>
      <div className="flex gap-3 items-start">
        <span className={`text-base mt-0.5 ${iconStyles[variant]}`}>
          {icons[variant]}
        </span>
        <div className="flex-1 min-w-0">
          {title && (
            <p className="text-sm font-semibold text-[#171915]">{title}</p>
          )}
          <p className={`text-sm text-[#52604e] ${title ? "mt-1" : ""}`}>
            {children}
          </p>
        </div>
      </div>
    </div>
  );
}

export function Panel({
  title,
  children,
  className = "",
}: {
  title?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={`border border-[#d7dccf] bg-white p-6 ${className}`}
    >
      {title && <h2 className="text-lg font-semibold mb-4">{title}</h2>}
      {children}
    </section>
  );
}

export function formatMoney(value: number) {
  return value.toFixed(value >= 10 ? 2 : 4);
}

export function formatInt(value: number) {
  return new Intl.NumberFormat("en-US").format(value);
}

export function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function formatDateTime(dateStr: string) {
  return new Date(dateStr).toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
