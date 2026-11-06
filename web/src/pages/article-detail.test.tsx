import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { SessionProvider } from '../app/session'
import { ArticleDetailPage } from './article-detail'

function json(status: number, body: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  }
}

const probe = () => json(200, { items: [], next_cursor: null })

function detailPayload(author: { kind: string; id: string | null; label?: string }) {
  return {
    id: 'article-1',
    title: '第一篇',
    body: '第一段\n第二段',
    author,
    publication_state: 'published',
    violation_state: 'clear',
    created_at: '2026-11-01T08:00:00Z',
    updated_at: '2026-11-02T08:00:00Z',
  }
}

function renderDetail(fetchImpl: FetchLike) {
  render(
    <SessionProvider fetchImpl={fetchImpl}>
      <MemoryRouter initialEntries={['/articles/article-1']}>
        <Routes>
          <Route path="/articles/:id" element={<ArticleDetailPage />} />
        </Routes>
      </MemoryRouter>
    </SessionProvider>,
  )
}

describe('ArticleDetailPage', () => {
  it('renders title, body and the anonymous author label', async () => {
    const fetchImpl: FetchLike = async (url) =>
      url.includes('limit=1')
        ? probe()
        : json(200, detailPayload({ kind: 'anonymous', id: null, label: '匿名' }))
    renderDetail(fetchImpl)

    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '第一篇' }),
      ).toBeInTheDocument(),
    )
    expect(screen.getByText(/第一段\s+第二段/)).toBeInTheDocument()
    expect(screen.getByText(/匿名 · 2026-11-01/)).toBeInTheDocument()
  })

  it('shows one unavailable message for a 404', async () => {
    const fetchImpl: FetchLike = async (url) =>
      url.includes('limit=1')
        ? probe()
        : json(404, {
            code: 'article.not_found',
            message: 'not found',
            trace_id: 'trace-404',
          })
    renderDetail(fetchImpl)

    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '内容不可用' }),
      ).toBeInTheDocument(),
    )
    expect(screen.getByText('这篇文章不存在或已不可见。')).toBeInTheDocument()
  })

  it('distinguishes a 403 from a 404', async () => {
    const fetchImpl: FetchLike = async (url) =>
      url.includes('limit=1')
        ? probe()
        : json(403, {
            code: 'auth.forbidden',
            message: '无权执行该操作',
            trace_id: 'trace-403',
          })
    renderDetail(fetchImpl)

    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '没有权限' }),
      ).toBeInTheDocument(),
    )
    expect(screen.queryByText('内容不可用')).toBeNull()
  })

  it('offers a retry after a server failure', async () => {
    let failed = false
    const fetchImpl: FetchLike = async (url) => {
      if (url.includes('limit=1')) {
        return probe()
      }
      if (!failed) {
        failed = true
        return json(500, {
          code: 'platform.internal_error',
          message: '服务器暂时不可用',
          trace_id: 'trace-500',
        })
      }
      return json(200, detailPayload({ kind: 'user', id: 'author-1' }))
    }
    renderDetail(fetchImpl)

    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: '重试' }))

    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '第一篇' }),
      ).toBeInTheDocument(),
    )
    expect(screen.getByText(/社区成员 · 2026-11-01/)).toBeInTheDocument()
  })
})
