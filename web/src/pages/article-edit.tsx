import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react'
import { Link, useParams } from 'react-router-dom'

import { createArticlesApi, type AuthorArticle } from '../api/articles'
import { ApiError } from '../api/client'
import { useSession } from '../app/session'

type EditState = 'loading' | 'ready' | 'missing' | 'forbidden' | 'error'

/**
 * Edits one Article through the author view. The backend requires a strong
 * `If-Match`, so the ETag from the prefill response is held in state and sent
 * back on every save; a stale ETag (412) asks the user to reload.
 */
export function ArticleEditPage() {
  const { id } = useParams<{ id: string }>()
  const { api } = useSession()
  const articles = useMemo(() => createArticlesApi(api), [api])

  const [status, setStatus] = useState<EditState>('loading')
  const [article, setArticle] = useState<AuthorArticle | null>(null)
  const [etag, setEtag] = useState<string | null>(null)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    if (id === undefined) {
      setStatus('missing')
      return
    }
    setStatus('loading')
    try {
      const result = await articles.getAuthor(id)
      setArticle(result.article)
      setEtag(result.etag)
      setTitle(result.article.title)
      setBody(result.article.body)
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
  }, [load])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (id === undefined || etag === null || article?.editable !== true) {
      return
    }
    setPending(true)
    setError(null)
    setSaved(false)
    try {
      const result = await articles.update(id, { title, body }, etag)
      setArticle(result.article)
      setEtag(result.etag)
      setSaved(true)
    } catch (caught) {
      if (
        caught instanceof ApiError &&
        (caught.status === 412 || caught.status === 409)
      ) {
        setError('内容已被修改，请刷新后重试。')
      } else {
        setError('保存失败，已保留你输入的内容，请稍后重试。')
      }
    } finally {
      setPending(false)
    }
  }

  if (status === 'loading') {
    return (
      <section>
        <h1>编辑文章</h1>
        <p role="status">正在加载文章…</p>
      </section>
    )
  }

  if (status === 'missing') {
    return (
      <section>
        <h1>内容不可用</h1>
        <Link to="/me/articles">返回我的文章</Link>
      </section>
    )
  }

  if (status === 'forbidden') {
    return (
      <section>
        <h1>没有权限</h1>
        <Link to="/me/articles">返回我的文章</Link>
      </section>
    )
  }

  if (status === 'error' || article === null) {
    return (
      <section>
        <h1>编辑文章</h1>
        <p role="alert">文章加载失败，请稍后重试。</p>
        <button type="button" onClick={() => void load()}>
          重试
        </button>
      </section>
    )
  }

  return (
    <section>
      <h1>编辑文章</h1>
      {article.editable ? null : (
        <p role="alert">这篇文章当前不可编辑。</p>
      )}
      {error !== null ? <p role="alert">{error}</p> : null}
      {saved ? <p role="status">已保存。</p> : null}
      <form aria-label="编辑文章表单" onSubmit={handleSubmit}>
        <label htmlFor="edit-title">标题</label>
        <input
          id="edit-title"
          value={title}
          readOnly={!article.editable}
          onChange={(event) => setTitle(event.target.value)}
        />

        <label htmlFor="edit-body">正文</label>
        <textarea
          id="edit-body"
          rows={12}
          value={body}
          readOnly={!article.editable}
          onChange={(event) => setBody(event.target.value)}
        />

        <button type="submit" disabled={pending || !article.editable}>
          {pending ? '正在保存…' : '保存'}
        </button>
      </form>
    </section>
  )
}
