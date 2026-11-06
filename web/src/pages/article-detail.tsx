import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import {
  authorLabel,
  createArticlesApi,
  type ArticleDetail,
} from '../api/articles'
import { ApiError } from '../api/client'
import { useSession } from '../app/session'

type DetailState = 'loading' | 'ready' | 'missing' | 'forbidden' | 'error'

export function ArticleDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { api } = useSession()
  const articles = useMemo(() => createArticlesApi(api), [api])

  const [detail, setDetail] = useState<ArticleDetail | null>(null)
  const [status, setStatus] = useState<DetailState>('loading')
  const [attempt, setAttempt] = useState(0)

  const load = useCallback(async () => {
    if (id === undefined) {
      setStatus('missing')
      return
    }
    setStatus('loading')
    try {
      setDetail(await articles.get(id))
      setStatus('ready')
    } catch (caught) {
      if (caught instanceof ApiError && caught.status === 404) {
        setStatus('missing')
        return
      }
      if (caught instanceof ApiError && caught.status === 403) {
        setStatus('forbidden')
        return
      }
      setStatus('error')
    }
  }, [articles, id])

  useEffect(() => {
    void load()
  }, [load, attempt])

  if (status === 'loading') {
    return (
      <section>
        <h1>文章详情</h1>
        <p role="status">正在加载文章…</p>
      </section>
    )
  }

  if (status === 'missing') {
    return (
      <section>
        <h1>内容不可用</h1>
        <p>这篇文章不存在或已不可见。</p>
        <Link to="/">返回文章列表</Link>
      </section>
    )
  }

  if (status === 'forbidden') {
    return (
      <section>
        <h1>没有权限</h1>
        <p>你没有查看这篇文章的权限。</p>
        <Link to="/">返回文章列表</Link>
      </section>
    )
  }

  if (status === 'error' || detail === null) {
    return (
      <section>
        <h1>文章详情</h1>
        <p role="alert">文章加载失败，请稍后重试。</p>
        <button type="button" onClick={() => setAttempt((value) => value + 1)}>
          重试
        </button>
      </section>
    )
  }

  return (
    <article>
      <h1>{detail.title}</h1>
      <p>
        {authorLabel(detail.author)} · {detail.createdAt.slice(0, 10)}
      </p>
      <p style={{ whiteSpace: 'pre-wrap' }}>{detail.body}</p>
      <Link to="/">返回文章列表</Link>
    </article>
  )
}
