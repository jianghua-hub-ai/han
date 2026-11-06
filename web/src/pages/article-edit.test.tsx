import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { SessionProvider } from '../app/session'
import { ArticleEditPage } from './article-edit'

function json(
  status: number,
  body: unknown,
  etag?: string,
): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
    headers: {
      get: (name: string) =>
        name.toLowerCase() === 'etag' && etag !== undefined ? etag : null,
    },
  }
}

function authorPayload(editable = true) {
  return {
    title: '原标题',
    body: '原正文',
    moderation_status: 'normal',
    editable,
  }
}

const probe = () => json(200, { items: [], next_cursor: null })

function renderEdit(fetchImpl: FetchLike) {
  render(
    <SessionProvider fetchImpl={fetchImpl}>
      <MemoryRouter initialEntries={['/me/articles/article-1/edit']}>
        <Routes>
          <Route
            path="/me/articles/:id/edit"
            element={<ArticleEditPage />}
          />
        </Routes>
      </MemoryRouter>
    </SessionProvider>,
  )
}

describe('ArticleEditPage', () => {
  it('prefills from the author view and sends the prefill ETag as If-Match', async () => {
    const puts: RequestInit[] = []
    const fetchImpl: FetchLike = async (url, init) => {
      if (url.includes('limit=1')) {
        return probe()
      }
      if (init.method === 'PUT') {
        puts.push(init)
        return json(
          200,
          { title: '新标题', body: '新正文', moderation_status: 'normal', editable: true },
          '"v2"',
        )
      }
      return json(200, authorPayload(), '"v1"')
    }
    renderEdit(fetchImpl)

    await waitFor(() =>
      expect(screen.getByLabelText('标题')).toHaveValue('原标题'),
    )
    expect(screen.getByLabelText('正文')).toHaveValue('原正文')

    fireEvent.change(screen.getByLabelText('标题'), {
      target: { value: '新标题' },
    })
    fireEvent.change(screen.getByLabelText('正文'), {
      target: { value: '新正文' },
    })
    fireEvent.submit(screen.getByRole('form', { name: '编辑文章表单' }))

    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('已保存'))
    expect(puts).toHaveLength(1)
    expect(puts[0].headers).toMatchObject({ 'If-Match': '"v1"' })
    expect(puts[0].body).toBe(JSON.stringify({ title: '新标题', body: '新正文' }))
  })

  it('asks the user to reload when the ETag is stale', async () => {
    const fetchImpl: FetchLike = async (url, init) => {
      if (url.includes('limit=1')) {
        return probe()
      }
      if (init.method === 'PUT') {
        return json(412, {
          code: 'article.precondition_failed',
          message: '前置条件不满足',
          trace_id: 'trace-412',
        })
      }
      return json(200, authorPayload(), '"v1"')
    }
    renderEdit(fetchImpl)
    await waitFor(() =>
      expect(screen.getByLabelText('标题')).toHaveValue('原标题'),
    )

    fireEvent.submit(screen.getByRole('form', { name: '编辑文章表单' }))

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        '内容已被修改，请刷新后重试。',
      ),
    )
  })

  it('keeps a restricted article read-only', async () => {
    const fetchImpl: FetchLike = async (url) =>
      url.includes('limit=1') ? probe() : json(200, authorPayload(false), '"v1"')
    renderEdit(fetchImpl)

    await waitFor(() =>
      expect(screen.getByLabelText('标题')).toHaveValue('原标题'),
    )
    expect(screen.getByLabelText('标题')).toHaveAttribute('readonly')
    expect(screen.getByRole('button', { name: '保存' })).toBeDisabled()
    expect(screen.getByRole('alert')).toHaveTextContent('这篇文章当前不可编辑')
  })
})
