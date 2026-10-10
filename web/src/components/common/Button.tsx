import React from "react";
import { Loader2 } from "lucide-react";

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?:
    "primary" | "secondary" | "danger" | "ghost" | "outline" | "emerald";
  size?: "sm" | "md" | "lg";
  isLoading?: boolean;
  icon?: React.ReactNode;
}

export const Button: React.FC<ButtonProps> = ({
  children,
  variant = "secondary",
  size = "md",
  isLoading = false,
  icon,
  className = "",
  disabled,
  ...props
}) => {
  const baseStyles =
    "inline-flex items-center justify-center font-medium rounded-xl transition-all duration-150 select-none active:scale-[0.98] disabled:opacity-50 disabled:pointer-events-none disabled:active:scale-100 min-h-[44px]";

  const sizeStyles = {
    sm: "text-xs px-3 py-1.5 gap-1.5 min-h-[44px]",
    md: "text-sm px-4 py-2 gap-2 min-h-[44px]",
    lg: "text-base px-5 py-2.5 gap-2.5 min-h-[48px]",
  };

  const variantStyles = {
    primary:
      "bg-blue-600 hover:bg-blue-700 text-white shadow-lg shadow-blue-600/20 active:bg-blue-700",
    emerald:
      "bg-emerald-700 hover:bg-emerald-800 text-white shadow-lg shadow-emerald-600/20 active:bg-emerald-800",
    secondary:
      "bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700/80 active:bg-slate-800",
    outline:
      "bg-transparent hover:bg-slate-800/80 text-slate-300 border border-slate-700 active:bg-slate-800",
    danger:
      "bg-rose-600 hover:bg-rose-700 text-white shadow-lg shadow-rose-600/20 active:bg-rose-700",
    ghost:
      "bg-transparent hover:bg-slate-800/60 text-slate-300 active:bg-slate-800",
  };

  return (
    <button
      type="button"
      className={`${baseStyles} ${sizeStyles[size]} ${variantStyles[variant]} ${className}`}
      disabled={disabled || isLoading}
      {...props}
    >
      {isLoading ? (
        <Loader2 className="w-4 h-4 animate-spin text-current" />
      ) : icon ? (
        <span className="flex items-center justify-center shrink-0">
          {icon}
        </span>
      ) : null}
      {children}
    </button>
  );
};
