export interface PaginationProps {
  hasMore: boolean
  loading: boolean
  onLoadMore(): void
}

/**
 * Cursor pagination has no page numbers: the only action is "load the next
 * slice", and the control disappears once the server stops returning a cursor.
 */
export function Pagination({ hasMore, loading, onLoadMore }: PaginationProps) {
  if (!hasMore) {
    return null
  }
  return (
    <button type="button" onClick={onLoadMore} disabled={loading}>
      {loading ? '正在加载…' : '加载更多'}
    </button>
  )
}
