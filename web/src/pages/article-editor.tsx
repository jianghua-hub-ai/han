import { useMemo, useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

import { createArticlesApi } from '../api/articles'
import { useSession } from '../app/session'

/** Creates a new Article; a failed save keeps every character the user typed. */
export function ArticleEditorPage() {
  const { api } = useSession()
  const articles = useMemo(() => createArticlesApi(api), [api])
  const navigate = useNavigate()

  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const canSubmit = !pending && title.trim() !== '' && body.trim() !== ''

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canSubmit) {
      return
    }
    setPending(true)
    setError(null)
    try {
      await articles.create({ title, body })
      navigate('/me/articles', { replace: true })
    } catch {
      setError('保存失败，已保留你输入的内容，请稍后重试。')
    } finally {
      setPending(false)
    }
  }

  return (
    <section>
      <h1>新建文章</h1>
      {error !== null ? <p role="alert">{error}</p> : null}
      <form aria-label="新建文章表单" onSubmit={handleSubmit}>
        <label htmlFor="article-title">标题</label>
        <input
          id="article-title"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
        />

        <label htmlFor="article-body">正文</label>
        <textarea
          id="article-body"
          rows={12}
          value={body}
          onChange={(event) => setBody(event.target.value)}
        />

        <button type="submit" disabled={!canSubmit}>
          {pending ? '正在保存…' : '发布'}
        </button>
      </form>
    </section>
  )
}
