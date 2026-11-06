import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { SessionProvider } from '../app/session'
import { RegisterPage, privacyNoticeText } from './register'

function json(status: number, body: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  }
}

const probe = () => json(200, { items: [], next_cursor: null })

function renderRegister(fetchImpl: FetchLike) {
  const calls: { url: string; init: RequestInit }[] = []
  const recording: FetchLike = async (url, init) => {
    calls.push({ url, init })
    return fetchImpl(url, init)
  }
  render(
    <SessionProvider fetchImpl={recording}>
      <MemoryRouter>
        <RegisterPage />
      </MemoryRouter>
    </SessionProvider>,
  )
  return calls
}

function fillForm() {
  fireEvent.change(screen.getByLabelText('用户名'), {
    target: { value: 'alice' },
  })
  fireEvent.change(screen.getByLabelText('邮箱'), {
    target: { value: 'alice@example.test' },
  })
  fireEvent.change(screen.getByLabelText('手机号'), {
    target: { value: '13800000000' },
  })
  fireEvent.change(screen.getByLabelText('密码'), {
    target: { value: 'pw12345' },
  })
}

function submit() {
  fireEvent.submit(screen.getByRole('form', { name: '注册表单' }))
}

describe('RegisterPage', () => {
  it('renders the four fields and the current privacy notice', async () => {
    renderRegister(async () => probe())

    expect(screen.getByRole('heading', { name: '注册' })).toBeInTheDocument()
    for (const label of ['用户名', '邮箱', '手机号', '密码']) {
      expect(screen.getByLabelText(label)).toBeInTheDocument()
    }
    expect(
      screen.getByRole('heading', { name: '用户隐私声明（v1.0.0）' }),
    ).toBeInTheDocument()
    expect(screen.getByText(privacyNoticeText)).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
  })

  it('blocks submission and explains why when the notice is not accepted', async () => {
    const calls = renderRegister(async () => probe())
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '注册' })).toBeInTheDocument(),
    )
    fillForm()

    submit()

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        '请先阅读并同意《用户隐私声明》再提交。',
      ),
    )
    expect(calls.some((call) => call.url.endsWith('/users'))).toBe(false)
  })

  it('sends the acknowledged version and guides the user to login', async () => {
    const calls = renderRegister(async (url) =>
      url.endsWith('/users')
        ? json(201, {
            id: 'user-1',
            username: 'alice',
            role: 'member',
            status: 'active',
            must_change_password: false,
            created_at: '2026-11-06T08:00:00Z',
          })
        : probe(),
    )
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '注册' })).toBeInTheDocument(),
    )
    fillForm()
    fireEvent.click(
      screen.getByLabelText('我已阅读并同意《用户隐私声明》（v1.0.0）'),
    )

    submit()

    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '注册成功' })).toBeInTheDocument(),
    )
    expect(screen.getByRole('link', { name: '前往登录' })).toHaveAttribute(
      'href',
      '/login',
    )
    // Never treated as logged in: no login call, and the password is gone from
    // the DOM (and was never persisted anywhere).
    expect(calls.some((call) => call.url.includes('/sessions'))).toBe(false)
    expect(screen.queryByLabelText('密码')).toBeNull()
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)

    const registration = calls.find((call) => call.url.endsWith('/users'))
    expect(registration?.init.method).toBe('POST')
    expect(JSON.parse(String(registration?.init.body))).toEqual({
      username: 'alice',
      email: 'alice@example.test',
      phone: '13800000000',
      password: 'pw12345',
      privacy_notice_version: 'v1.0.0',
      privacy_acknowledged: true,
    })
  })

  it('reports a conflict without revealing which account already exists', async () => {
    renderRegister(async (url) =>
      url.endsWith('/users')
        ? json(409, {
            code: 'identity.identifier_conflict',
            message: '身份标识已被使用',
            trace_id: 'trace-conflict',
          })
        : probe(),
    )
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '注册' })).toBeInTheDocument(),
    )
    fillForm()
    fireEvent.click(
      screen.getByLabelText('我已阅读并同意《用户隐私声明》（v1.0.0）'),
    )

    submit()

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        '用户名、邮箱或手机号已被使用，请更换后重试。',
      ),
    )
    // The form survives the failure so the user keeps their input.
    expect(screen.getByLabelText('用户名')).toHaveValue('alice')
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
  })

  it('maps a stale privacy notice to a refresh instruction', async () => {
    renderRegister(async (url) =>
      url.endsWith('/users')
        ? json(409, {
            code: 'privacy.notice_outdated',
            message: '隐私声明已更新，请重新确认',
            trace_id: 'trace-notice',
          })
        : probe(),
    )
    await waitFor(() =>
      expect(screen.getByRole('heading', { name: '注册' })).toBeInTheDocument(),
    )
    fillForm()
    fireEvent.click(
      screen.getByLabelText('我已阅读并同意《用户隐私声明》（v1.0.0）'),
    )

    submit()

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        '隐私声明已更新，请刷新页面后重新确认。',
      ),
    )
  })
})
