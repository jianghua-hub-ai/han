import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { SessionProvider } from '../app/session'
import { MyArticlesPage } from './my-articles'

function json(status: number, body: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  }
}

function summary(id: string, title: string) {
  return {
    id,
    title,
    author: { kind: 'user', id: 'author-1', label: '' },
    created_at: '2026-11-01T08:00:00Z',
    updated_at: '2026-11-02T08:00:00Z',
  }
}

function renderPage(fetchImpl: FetchLike) {
  render(
    <SessionProvider fetchImpl={fetchImpl}>
      <MemoryRouter>
        <MyArticlesPage />
      </MemoryRouter>
    </SessionProvider>,
  )
}

const probe = () => json(200, { items: [], next_cursor: null })

describe('MyArticlesPage', () => {
  it('lists the current user articles and links to edit', async () => {
    const fetchImpl: FetchLike = async (url) => {
      if (url.includes('limit=1')) {
        return probe()
      }
      return json(200, {
        items: [summary('1', '第一篇'), summary('2', '第二篇')],
        next_cursor: null,
      })
    }
    renderPage(fetchImpl)

    await waitFor(() =>
      expect(screen.getAllByRole('listitem')).toHaveLength(2),
    )
    const editLinks = screen.getAllByRole('link', { name: '编辑' })
    expect(editLinks[0]).toHaveAttribute('href', '/me/articles/1/edit')
    expect(screen.getByRole('link', { name: '新建文章' })).toHaveAttribute(
      'href',
      '/me/articles/new',
    )
  })

  it('shows an empty state when the author has no articles', async () => {
    renderPage(async () => json(200, { items: [], next_cursor: null }))

    await waitFor(() =>
      expect(screen.getByText('你还没有文章')).toBeInTheDocument(),
    )
    expect(screen.queryByRole('alert')).toBeNull()
  })

  it('requires confirmation before deleting and removes the row on success', async () => {
    let deleted = false
    const fetchImpl: FetchLike = async (url, init) => {
      if (url.includes('limit=1')) {
        return probe()
      }
      if (init.method === 'DELETE') {
        deleted = true
        return { ok: true, status: 204, text: async () => '' }
      }
      return json(200, {
        items: deleted
          ? [summary('2', '第二篇')]
          : [summary('1', '第一篇'), summary('2', '第二篇')],
        next_cursor: null,
      })
    }
    renderPage(fetchImpl)
    await waitFor(() =>
      expect(screen.getAllByRole('listitem')).toHaveLength(2),
    )

    fireEvent.click(screen.getByRole('button', { name: '删除《第一篇》' }))
    expect(screen.getByRole('dialog')).toHaveTextContent('删除后不可恢复')
    fireEvent.click(screen.getByRole('button', { name: '取消' }))
    expect(screen.getAllByRole('listitem')).toHaveLength(2)

    fireEvent.click(screen.getByRole('button', { name: '删除《第一篇》' }))
    fireEvent.click(screen.getByRole('button', { name: '确认删除' }))

    await waitFor(() =>
      expect(screen.getAllByRole('listitem')).toHaveLength(1),
    )
    expect(screen.queryByText('第一篇')).toBeNull()
  })

  it('keeps the article when deletion fails', async () => {
    const fetchImpl: FetchLike = async (url, init) => {
      if (url.includes('limit=1')) {
        return probe()
      }
      if (init.method === 'DELETE') {
        return json(500, {
          code: 'platform.internal_error',
          message: '服务器暂时不可用',
          trace_id: 'trace-delete',
        })
      }
      return json(200, {
        items: [summary('1', '第一篇')],
        next_cursor: null,
      })
    }
    renderPage(fetchImpl)
    await waitFor(() =>
      expect(screen.getAllByRole('listitem')).toHaveLength(1),
    )

    fireEvent.click(screen.getByRole('button', { name: '删除《第一篇》' }))
    fireEvent.click(screen.getByRole('button', { name: '确认删除' }))

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('删除失败'),
    )
    expect(screen.getAllByRole('listitem')).toHaveLength(1)
    expect(screen.getByRole('link', { name: '第一篇' })).toBeInTheDocument()
  })
})
