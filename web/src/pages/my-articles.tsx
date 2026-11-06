import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'

import {
  createArticlesApi,
  type ArticleSummary,
} from '../api/articles'
import { useSession } from '../app/session'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { Pagination } from '../components/Pagination'

const pageSize = 20

export function MyArticlesPage() {
  const { api } = useSession()
  const articles = useMemo(() => createArticlesApi(api), [api])

  const [items, setItems] = useState<ArticleSummary[]>([])
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pendingDelete, setPendingDelete] = useState<ArticleSummary | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const started = useRef(false)

  const load = useCallback(
    async (cursor: string | null, append: boolean) => {
      setLoading(true)
      setError(null)
      try {
        const page = await articles.listMine({ cursor, limit: pageSize })
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

  async function confirmDelete() {
    if (pendingDelete === null) {
      return
    }
    setDeleting(true)
    setDeleteError(null)
    try {
      await articles.remove(pendingDelete.id)
      setItems((current) =>
        current.filter((item) => item.id !== pendingDelete.id),
      )
      setPendingDelete(null)
    } catch {
      // Keep the row exactly as it was; nothing is removed optimistically.
      setDeleteError('删除失败，文章仍保留在列表中，请稍后重试。')
    } finally {
      setDeleting(false)
    }
  }

  return (
    <section>
      <h1>我的文章</h1>
      <p>
        <Link to="/me/articles/new">新建文章</Link>
      </p>
      {deleteError !== null ? <p role="alert">{deleteError}</p> : null}
      {error !== null ? (
        <>
          <p role="alert">{error}</p>
          <button
            type="button"
            onClick={() => {
              void load(items.length === 0 ? null : nextCursor, items.length > 0)
            }}
          >
            重试
          </button>
        </>
      ) : loading && items.length === 0 ? (
        <p role="status">正在加载文章…</p>
      ) : items.length === 0 ? (
        <p>你还没有文章</p>
      ) : (
        <ul>
          {items.map((item) => (
            <li key={item.id}>
              <Link to={`/articles/${item.id}`}>{item.title}</Link>
              <Link to={`/me/articles/${item.id}/edit`}>编辑</Link>
              <button
                type="button"
                onClick={() => {
                  setDeleteError(null)
                  setPendingDelete(item)
                }}
              >
                删除《{item.title}》
              </button>
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
      <ConfirmDialog
        open={pendingDelete !== null}
        title="确认删除"
        message="删除后不可恢复，确定要删除这篇文章吗？"
        confirmLabel="确认删除"
        pending={deleting}
        onConfirm={() => {
          void confirmDelete()
        }}
        onCancel={() => setPendingDelete(null)}
      />
    </section>
  )
}

function mergeArticles(
  existing: ArticleSummary[],
  incoming: ArticleSummary[],
): ArticleSummary[] {
  const seen = new Set(existing.map((item) => item.id))
  return [...existing, ...incoming.filter((item) => !seen.has(item.id))]
}
