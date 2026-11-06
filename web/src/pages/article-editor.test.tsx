import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { SessionProvider } from '../app/session'
import { ArticleEditorPage } from './article-editor'

function json(status: number, body: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  }
}

function renderEditor(fetchImpl: FetchLike) {
  render(
    <SessionProvider fetchImpl={fetchImpl}>
      <MemoryRouter initialEntries={['/me/articles/new']}>
        <Routes>
          <Route path="/me/articles/new" element={<ArticleEditorPage />} />
          <Route path="/me/articles" element={<p>我的文章页</p>} />
        </Routes>
      </MemoryRouter>
    </SessionProvider>,
  )
}

function fillAndSubmit(title: string, body: string) {
  fireEvent.change(screen.getByLabelText('标题'), {
    target: { value: title },
  })
  fireEvent.change(screen.getByLabelText('正文'), {
    target: { value: body },
  })
  fireEvent.submit(screen.getByRole('form', { name: '新建文章表单' }))
}

describe('ArticleEditorPage', () => {
  it('creates the article and returns to my articles', async () => {
    const posts: unknown[] = []
    const fetchImpl: FetchLike = async (url, init) => {
      if (url.includes('limit=1')) {
        return json(200, { items: [], next_cursor: null })
      }
      posts.push(init.body)
      return json(201, { id: 'article-1', title: '标题', body: '正文' })
    }
    renderEditor(fetchImpl)
    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '新建文章' }),
      ).toBeInTheDocument(),
    )

    fillAndSubmit('标题', '正文')

    await waitFor(() =>
      expect(screen.getByText('我的文章页')).toBeInTheDocument(),
    )
    expect(posts).toEqual([JSON.stringify({ title: '标题', body: '正文' })])
  })

  it('keeps the typed content when the save fails', async () => {
    const fetchImpl: FetchLike = async (url) => {
      if (url.includes('limit=1')) {
        return json(200, { items: [], next_cursor: null })
      }
      return json(500, {
        code: 'platform.internal_error',
        message: '服务器暂时不可用',
        trace_id: 'trace-create',
      })
    }
    renderEditor(fetchImpl)
    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '新建文章' }),
      ).toBeInTheDocument(),
    )

    fillAndSubmit('保留的标题', '保留的正文')

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('保存失败'),
    )
    expect(screen.getByLabelText('标题')).toHaveValue('保留的标题')
    expect(screen.getByLabelText('正文')).toHaveValue('保留的正文')
  })
})
