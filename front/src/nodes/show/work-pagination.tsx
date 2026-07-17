import { ChevronLeft, ChevronRight } from "lucide-react";

import { Button } from "@/components/ui/button";

type WorkPaginationProps = {
  loading: boolean;
  hidden?: boolean;
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
};

export function WorkPagination({
  loading,
  hidden = false,
  page,
  pageSize,
  total,
  onPageChange,
}: WorkPaginationProps) {
  if (hidden || total <= 0) return null;

  const safePageSize = Math.max(1, Number(pageSize) || 1);
  const totalPages = Math.max(1, Math.ceil(total / safePageSize));
  const currentPage = Math.min(totalPages, Math.max(1, Number(page) || 1));

  return (
    <nav
      className="flex flex-col items-center gap-3 border-t border-border/70 px-4 py-3 text-xs text-muted-foreground sm:flex-row sm:justify-between"
      aria-label="列表分页"
    >
      <span aria-live="polite">
        第 {currentPage} / {totalPages} 页，共 {total} 条
      </span>
      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="min-w-20"
          disabled={loading || currentPage <= 1}
          onClick={() => onPageChange(currentPage - 1)}
        >
          <ChevronLeft className="h-4 w-4" />
          上一页
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="min-w-20"
          disabled={loading || currentPage >= totalPages}
          onClick={() => onPageChange(currentPage + 1)}
        >
          下一页
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </nav>
  );
}
