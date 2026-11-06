import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'

import {
  authorLabel,
  createArticlesApi,
  type ArticleSummary,
} from '../api/articles'
import { useSession } from '../app/session'
import { Pagination } from '../components/Pagination'

const pageSize = 20

export function ArticleListPage() {
  const { api } = useSession()
  const articles = useMemo(() => createArticlesApi(api), [api])

  const [items, setItems] = useState<ArticleSummary[]>([])
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const started = useRef(false)

  const load = useCallback(
    async (cursor: string | null, append: boolean) => {
      setLoading(true)
      setError(null)
      try {
        const page = await articles.list({ cursor, limit: pageSize })
        setItems((current) =>
          append ? mergeArticles(current, page.items) : page.items,
        )
        setNextCursor(page.nextCursor)
      } catch {
        setError('文章列表加载失败，请稍后重试。')
      } finally {
        setLoading(false)
      }
    },
    [articles],
  )

  useEffect(() => {
    if (started.current) {
      return
    }
    started.current = true
    void load(null, false)
  }, [load])

  if (error !== null) {
    return (
      <section>
        <h1>文章列表</h1>
        <p role="alert">{error}</p>
        <button
          type="button"
          onClick={() => {
            void load(items.length === 0 ? null : nextCursor, items.length > 0)
          }}
        >
          重试
        </button>
      </section>
    )
  }

  if (loading && items.length === 0) {
    return (
      <section>
        <h1>文章列表</h1>
        <p role="status">正在加载文章…</p>
      </section>
    )
  }

  return (
    <section>
      <h1>文章列表</h1>
      {items.length === 0 ? (
        <p>还没有文章</p>
      ) : (
        <ul>
          {items.map((item) => (
            <li key={item.id}>
              <Link to={`/articles/${item.id}`}>{item.title}</Link>
              <p>
                {authorLabel(item.author)} · {formatDay(item.createdAt)}
              </p>
            </li>
          ))}
        </ul>
      )}
      <Pagination
        hasMore={nextCursor !== null}
        loading={loading}
        onLoadMore={() => {
          void load(nextCursor, true)
        }}
      />
    </section>
  )
}

/** Appends a page without duplicating rows if the server repeats an item. */
function mergeArticles(
  existing: ArticleSummary[],
  incoming: ArticleSummary[],
): ArticleSummary[] {
  const seen = new Set(existing.map((item) => item.id))
  return [...existing, ...incoming.filter((item) => !seen.has(item.id))]
}

/** Stable day-level rendering; avoids locale/time-zone drift in tests. */
function formatDay(value: string): string {
  return value.slice(0, 10)
}
