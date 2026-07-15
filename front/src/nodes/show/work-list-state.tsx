import { Inbox, Loader2 } from "lucide-react";

export function WorkListState({
  title,
  description,
  loading = false,
  className = "",
}: {
  title: string;
  description: string;
  loading?: boolean;
  className?: string;
}) {
  const Icon = loading ? Loader2 : Inbox;

  return (
    <div
      className={`flex min-h-[240px] w-full flex-col items-center justify-center px-6 py-12 text-center ${className}`}
    >
      <div className="flex h-11 w-11 items-center justify-center rounded-md bg-muted/60 text-muted-foreground/80">
        <Icon
          className={`h-5 w-5 ${loading ? "animate-spin" : ""}`}
          strokeWidth={1.7}
          aria-hidden="true"
        />
      </div>
      <h3 className="mt-4 text-sm font-semibold text-foreground/90">
        {title}
      </h3>
      <p className="mt-1.5 max-w-xs text-xs leading-5 text-muted-foreground">
        {description}
      </p>
    </div>
  );
}
