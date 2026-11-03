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
    "flow-button inline-flex min-w-0 max-w-full [overflow-wrap:anywhere] items-center justify-center font-medium select-none disabled:opacity-50 disabled:pointer-events-none min-h-[44px]";

  const sizeStyles = {
    sm: "text-xs px-3 py-1.5 gap-1.5 min-h-[44px]",
    md: "text-sm px-4 py-2 gap-2 min-h-[44px]",
    lg: "text-base px-5 py-2.5 gap-2.5 min-h-[48px]",
  };

  const variantStyles = {
    primary: "flow-button-primary",
    emerald: "flow-button-success",
    secondary: "flow-button-tonal",
    outline: "flow-button-outline",
    danger: "flow-button-danger",
    ghost: "flow-button-ghost",
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
