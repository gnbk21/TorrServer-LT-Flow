import React from "react";

export interface BadgeProps {
  children: React.ReactNode;
  variant?: "default" | "success" | "warning" | "danger" | "info" | "purple";
  className?: string;
  icon?: React.ReactNode;
}

export const Badge: React.FC<BadgeProps> = ({
  children,
  variant = "default",
  className = "",
  icon,
}) => {
  const variantStyles = {
    default: "bg-slate-800 text-slate-300 border-slate-700/80",
    success: "bg-emerald-950/70 text-emerald-300 border-emerald-800/60",
    warning: "bg-amber-950/70 text-amber-300 border-amber-800/60",
    danger: "bg-rose-950/70 text-rose-300 border-rose-800/60",
    info: "bg-blue-950/70 text-blue-300 border-blue-800/60",
    purple: "bg-purple-950/70 text-purple-300 border-purple-800/60",
  };

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border ${variantStyles[variant]} ${className}`}
    >
      {icon && (
        <span className="w-3.5 h-3.5 flex items-center justify-center">
          {icon}
        </span>
      )}
      {children}
    </span>
  );
};
