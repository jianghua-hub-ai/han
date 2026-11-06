import { act, render, screen, waitFor } from '@testing-library/react'
import { useEffect } from 'react'
import { MemoryRouter, useLocation, useRoutes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { routes } from './router'
import { SessionProvider } from './session'

function json(status: number, body: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  }
}

const authenticated: FetchLike = async (url) => {
  if (url.includes('/users/me/articles/')) {
    return json(200, {
      title: '第一篇',
      body: '正文',
      moderation_status: 'normal',
      editable: true,
    })
  }
  if (url.includes('/articles/article-1')) {
    return json(200, {
        id: 'article-1',
        title: '第一篇',
        body: '正文',
        author: { kind: 'user', id: 'author-1', label: '' },
        publication_state: 'published',
        violation_state: 'clear',
        created_at: '2026-11-01T08:00:00Z',
        updated_at: '2026-11-02T08:00:00Z',
      })
  }
  return json(200, { items: [], next_cursor: null })
}

const anonymous: FetchLike = async () =>
  json(401, {
    code: 'auth.session_required',
    message: '需要重新登录',
    trace_id: 'trace-router',
  })

interface CapturedLocation {
  pathname: string
  state: unknown
}

function Harness({ captured }: { captured: CapturedLocation }) {
  const element = useRoutes(routes)
  const location = useLocation()
  useEffect(() => {
    captured.pathname = location.pathname
    captured.state = location.state
  }, [captured, location])
  return element
}

function renderRouter(initialPath: string, fetchImpl: FetchLike) {
  const captured: CapturedLocation = { pathname: initialPath, state: null }
  render(
    <SessionProvider fetchImpl={fetchImpl}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Harness captured={captured} />
      </MemoryRouter>
    </SessionProvider>,
  )
  return captured
}

describe('routes', () => {
  it('redirects an anonymous visitor to login and keeps the target path', async () => {
    const captured = renderRouter('/me/articles?page=2', anonymous)

    await waitFor(() => expect(captured.pathname).toBe('/login'))
    expect(captured.state).toMatchObject({ from: '/me/articles?page=2' })
    expect(screen.getByRole('heading', { name: '登录' })).toBeInTheDocument()
  })

  it('shows the recovery skeleton while the startup probe is running', async () => {
    let release!: (value: HttpResponseLike) => void
    const pending = new Promise<HttpResponseLike>((resolve) => {
      release = resolve
    })

    renderRouter('/', () => pending)

    expect(screen.getByRole('status')).toHaveTextContent('正在恢复登录状态')

    await act(async () => {
      release(json(401, {
        code: 'auth.session_required',
        message: '需要重新登录',
        trace_id: 'trace-pending',
      }))
      await pending
    })
  })

  it.each([
    ['/', '文章列表'],
    ['/articles/article-1', '第一篇'],
    ['/me/articles', '我的文章'],
    ['/me/articles/new', '新建文章'],
    ['/me/articles/article-1/edit', '编辑文章'],
  ])('renders %s for an authenticated session', async (path, heading) => {
    renderRouter(path, authenticated)

    await waitFor(() =>
      expect(screen.getByRole('heading', { name: heading })).toBeInTheDocument(),
    )
  })

  it.each([
    ['/login', '登录'],
    ['/register', '注册'],
  ])('renders the public route %s without a session', async (path, heading) => {
    renderRouter(path, anonymous)

    await waitFor(() =>
      expect(screen.getByRole('heading', { name: heading })).toBeInTheDocument(),
    )
  })

  it('falls back to the not-found page for unknown paths', async () => {
    renderRouter('/does-not-exist', authenticated)

    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '页面不存在' }),
      ).toBeInTheDocument(),
    )
  })
})
