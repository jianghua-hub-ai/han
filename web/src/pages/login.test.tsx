import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { useEffect } from 'react'
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useRoutes,
} from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { routes } from '../app/router'
import { SessionProvider, useSession } from '../app/session'
import { LoginPage } from './login'

function json(status: number, body: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  }
}

const anonymousProbe = () =>
  json(401, {
    code: 'auth.session_required',
    message: '需要重新登录',
    trace_id: 'trace-login',
  })

const loginSuccess = {
  user: {
    id: 'user-1',
    username: 'alice',
    role: 'member',
    status: 'active',
    is_initial_admin: false,
    must_change_password: false,
    must_acknowledge_privacy: false,
  },
  csrf_token: 'csrf-in-memory',
}

interface Captured {
  pathname: string
}

function LocationProbe({ captured }: { captured: Captured }) {
  const location = useLocation()
  useEffect(() => {
    captured.pathname = location.pathname
  }, [captured, location])
  return null
}

function renderLoginPage(fetchImpl: FetchLike, state: unknown = null) {
  const captured: Captured = { pathname: '/login' }
  render(
    <SessionProvider fetchImpl={fetchImpl}>
      <MemoryRouter initialEntries={[{ pathname: '/login', state }]}>
        <LocationProbe captured={captured} />
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<p>文章列表主页</p>} />
          <Route path="/me/articles" element={<p>我的文章页</p>} />
        </Routes>
      </MemoryRouter>
    </SessionProvider>,
  )
  return captured
}

function fillAndSubmit(username: string, password: string) {
  fireEvent.change(screen.getByLabelText('用户名'), {
    target: { value: username },
  })
  fireEvent.change(screen.getByLabelText('密码'), {
    target: { value: password },
  })
  fireEvent.submit(screen.getByRole('form', { name: '登录表单' }))
}

describe('LoginPage', () => {
  it('renders username and password fields wired to their labels', async () => {
    renderLoginPage(async () => anonymousProbe())

    expect(screen.getByRole('heading', { name: '登录' })).toBeInTheDocument()
    expect(screen.getByLabelText('用户名')).toHaveAttribute(
      'autocomplete',
      'username',
    )
    expect(screen.getByLabelText('密码')).toHaveAttribute('type', 'password')
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })

  it('returns to the protected target after a successful login', async () => {
    const fetchImpl: FetchLike = async (url) =>
      url.endsWith('/sessions') ? json(200, loginSuccess) : anonymousProbe()
    const captured = renderLoginPage(fetchImpl, { from: '/me/articles' })
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '登录' })).toBeInTheDocument(),
    )

    fillAndSubmit('alice', 'pw12345')

    await waitFor(() => expect(captured.pathname).toBe('/me/articles'))
    expect(screen.getByText('我的文章页')).toBeInTheDocument()
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
  })

  it('shows one unified message for any rejected credential', async () => {
    const fetchImpl: FetchLike = async (url) =>
      url.endsWith('/sessions')
        ? json(401, {
            code: 'auth.invalid_credentials',
            message: '用户名或密码不正确',
            trace_id: 'trace-bad-login',
          })
        : anonymousProbe()
    const captured = renderLoginPage(fetchImpl)
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '登录' })).toBeInTheDocument(),
    )

    fillAndSubmit('someone', 'wrong-password')

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        '登录失败，请检查用户名和密码',
      ),
    )
    expect(captured.pathname).toBe('/login')
    // The backend's more specific text must not reach the user.
    expect(screen.queryByText('用户名或密码不正确')).toBeNull()
    // A failed login attempt is not an expired session.
    expect(screen.queryByText('登录已失效，请重新登录。')).toBeNull()
  })

  it('disables the submit button while the request is in flight', async () => {
    let release!: (value: HttpResponseLike) => void
    const pending = new Promise<HttpResponseLike>((resolve) => {
      release = resolve
    })
    const fetchImpl: FetchLike = async (url) =>
      url.endsWith('/sessions') ? pending : anonymousProbe()
    renderLoginPage(fetchImpl)
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '登录' })).toBeInTheDocument(),
    )

    fillAndSubmit('alice', 'pw12345')

    const button = screen.getByRole('button', { name: '正在登录…' })
    expect(button).toBeDisabled()

    await act(async () => {
      release(json(200, loginSuccess))
      await pending
    })
  })

  it('explains an expired session when a live session dies', async () => {
    const captured: Captured = { pathname: '/me/articles' }
    let authorized = true
    const fetchImpl: FetchLike = async () =>
      authorized
        ? json(200, { items: [], next_cursor: null })
        : anonymousProbe()

    function RouteTree() {
      const element = useRoutes(routes)
      return (
        <>
          <LocationProbe captured={captured} />
          {element}
        </>
      )
    }

    function Failer() {
      const { api } = useSession()
      return (
        <button
          type="button"
          onClick={() => {
            void api.request('/articles').catch(() => undefined)
          }}
        >
          模拟失效
        </button>
      )
    }

    render(
      <SessionProvider fetchImpl={fetchImpl}>
        <MemoryRouter initialEntries={['/me/articles']}>
          <RouteTree />
          <Failer />
        </MemoryRouter>
      </SessionProvider>,
    )

    await waitFor(() =>
      expect(
        screen.getByRole('heading', { name: '我的文章' }),
      ).toBeInTheDocument(),
    )

    authorized = false
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '模拟失效' }))
    })

    await waitFor(() => expect(captured.pathname).toBe('/login'))
    expect(screen.getByRole('alert')).toHaveTextContent('登录已失效')
  })
})
