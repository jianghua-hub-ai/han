import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { SessionProvider } from '../app/session'
import { ArticleListPage } from './article-list'

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

const probe = () => json(200, { items: [], next_cursor: null })

function renderList(fetchImpl: FetchLike) {
  render(
    <SessionProvider fetchImpl={fetchImpl}>
      <MemoryRouter>
        <ArticleListPage />
      </MemoryRouter>
    </SessionProvider>,
  )
}

describe('ArticleListPage', () => {
  it('pages by cursor without duplicates or skips, then stops at the end', async () => {
    const urls: string[] = []
    const fetchImpl: FetchLike = async (url) => {
      urls.push(url)
      if (url.includes('limit=1')) {
        return probe()
      }
      if (url.includes('cursor=cursor-1')) {
        return json(200, {
          items: [summary('3', '第三篇'), summary('4', '第四篇')],
          next_cursor: null,
        })
      }
      return json(200, {
        items: [summary('1', '第一篇'), summary('2', '第二篇')],
        next_cursor: 'cursor-1',
      })
    }
    renderList(fetchImpl)

    await waitFor(() =>
      expect(screen.getAllByRole('listitem')).toHaveLength(2),
    )
    expect(
      screen.getByRole('link', { name: '第一篇' }),
    ).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '加载更多' }))

    await waitFor(() =>
      expect(screen.getAllByRole('listitem')).toHaveLength(4),
    )
    expect(
      screen
        .getAllByRole('listitem')
        .map((item) => item.textContent?.split('社区成员')[0]),
    ).toEqual([
      '第一篇',
      '第二篇',
      '第三篇',
      '第四篇',
    ])
    expect(screen.queryByRole('button', { name: '加载更多' })).toBeNull()

    const listCalls = urls.filter((url) => !url.includes('limit=1'))
    expect(listCalls[0]).toContain('limit=20')
    expect(listCalls[0]).not.toContain('cursor=')
    expect(listCalls[1]).toContain('cursor=cursor-1')
  })

  it('shows an empty state instead of an error when there are no articles', async () => {
    const fetchImpl: FetchLike = async () =>
      json(200, { items: [], next_cursor: null })
    renderList(fetchImpl)

    await waitFor(() => expect(screen.getByText('还没有文章')).toBeInTheDocument())
    expect(screen.queryByRole('alert')).toBeNull()
    expect(screen.queryByRole('button', { name: '加载更多' })).toBeNull()
  })

  it('reports a load failure and recovers on retry', async () => {
    let failed = false
    const fetchImpl: FetchLike = async (url) => {
      if (url.includes('limit=1')) {
        return probe()
      }
      if (!failed) {
        failed = true
        return json(503, {
          code: 'platform.database_unavailable',
          message: '暂时不可用',
          trace_id: 'trace-list',
        })
      }
      return json(200, {
        items: [summary('1', '第一篇')],
        next_cursor: null,
      })
    }
    renderList(fetchImpl)

    await waitFor(() => expect(screen.getByRole('alert')).toBeInTheDocument())
    expect(screen.queryByText('第一篇')).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: '重试' }))

    await waitFor(() =>
      expect(screen.getByRole('link', { name: '第一篇' })).toBeInTheDocument(),
    )
    expect(screen.queryByRole('alert')).toBeNull()
  })
})
