import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { FetchLike, HttpResponseLike } from '../api/client'
import { SessionProvider, useSession } from './session'

function json(status: number, body: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  }
}

function noContent(): HttpResponseLike {
  return { ok: true, status: 204, text: async () => '' }
}

const anonymousProbe = () =>
  json(401, {
    code: 'auth.session_required',
    message: '需要重新登录',
    trace_id: 'trace-probe',
  })

const authenticatedProbe = () => json(200, { items: [], next_cursor: null })

const loginPayload = {
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

/**
 * One controllable backend: the startup probe follows `signedIn`, while login,
 * logout and arbitrary reads stay predictable and every call is recorded.
 */
function scenario() {
  const state = { signedIn: false, transportDown: false }
  const calls: { url: string; init: RequestInit }[] = []

  const fetchImpl: FetchLike = async (url, init) => {
    calls.push({ url, init })
    if (url.endsWith('/sessions/logout')) {
      state.signedIn = false
      return noContent()
    }
    if (url.endsWith('/sessions') && init.method === 'POST') {
      state.signedIn = true
      return json(200, loginPayload)
    }
    if (state.transportDown) {
      throw new TypeError('Failed to fetch')
    }
    return state.signedIn ? authenticatedProbe() : anonymousProbe()
  }

  return { state, calls, fetchImpl }
}

function renderSession(fetchImpl: FetchLike) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <SessionProvider fetchImpl={fetchImpl}>{children}</SessionProvider>
  )
  return renderHook(() => useSession(), { wrapper })
}

afterEach(() => {
  localStorage.clear()
  sessionStorage.clear()
})

describe('SessionProvider', () => {
  it('treats a 401 probe as anonymous without persisting anything', async () => {
    const { fetchImpl } = scenario()
    const { result } = renderSession(fetchImpl)

    expect(result.current.state).toBe('unknown')
    await waitFor(() => expect(result.current.state).toBe('anonymous'))
    expect(result.current.csrfToken).toBeNull()
    expect(result.current.user).toBeNull()
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)
  })

  it('recovers an existing session from a successful probe', async () => {
    const backend = scenario()
    backend.state.signedIn = true
    const { result } = renderSession(backend.fetchImpl)

    // A live cookie resolves to authenticated; the probe itself carries no user
    // payload, so the UI must not invent one.
    await waitFor(() => expect(result.current.state).toBe('authenticated'))
    expect(result.current.bootstrapFailed).toBe(false)
    expect(result.current.user).toBeNull()
  })

  it('keeps the session unknown when the probe fails for a transport reason', async () => {
    const backend = scenario()
    backend.state.transportDown = true
    const { result } = renderSession(backend.fetchImpl)

    await waitFor(() => expect(result.current.bootstrapFailed).toBe(true))
    expect(result.current.state).toBe('unknown')
  })

  it('keeps the CSRF token in memory and sends it on writes', async () => {
    const backend = scenario()
    const { result } = renderSession(backend.fetchImpl)
    await waitFor(() => expect(result.current.state).toBe('anonymous'))

    await act(async () => {
      await result.current.login({ username: 'alice', password: 'pw12345' })
    })

    expect(result.current.state).toBe('authenticated')
    expect(result.current.user?.username).toBe('alice')
    expect(result.current.csrfToken).toBe('csrf-in-memory')
    expect(localStorage.length).toBe(0)
    expect(sessionStorage.length).toBe(0)

    await act(async () => {
      await result.current.api.request('/articles', {
        method: 'POST',
        body: { title: 't', body: 'b' },
      })
    })

    const write = backend.calls.find(
      (call) => call.init.method === 'POST' && call.url.endsWith('/articles'),
    )
    expect(write?.init.headers).toMatchObject({
      'X-CSRF-Token': 'csrf-in-memory',
    })
  })

  it('drops to anonymous and forgets the token on any later 401', async () => {
    const backend = scenario()
    const { result } = renderSession(backend.fetchImpl)
    await waitFor(() => expect(result.current.state).toBe('anonymous'))

    await act(async () => {
      await result.current.login({ username: 'alice', password: 'pw12345' })
    })
    expect(result.current.csrfToken).toBe('csrf-in-memory')

    backend.state.signedIn = false
    await act(async () => {
      await expect(result.current.api.request('/articles')).rejects.toThrow()
    })

    expect(result.current.state).toBe('anonymous')
    expect(result.current.csrfToken).toBeNull()
    expect(result.current.user).toBeNull()
  })

  it('clears the session after a successful logout', async () => {
    const backend = scenario()
    const { result } = renderSession(backend.fetchImpl)
    await waitFor(() => expect(result.current.state).toBe('anonymous'))

    await act(async () => {
      await result.current.login({ username: 'alice', password: 'pw12345' })
    })
    expect(result.current.state).toBe('authenticated')

    await act(async () => {
      await result.current.logout()
    })

    expect(result.current.state).toBe('anonymous')
    expect(result.current.csrfToken).toBeNull()
    expect(result.current.user).toBeNull()
  })

  it('exposes a same-origin client for page code', async () => {
    const backend = scenario()
    const spy = vi.fn(backend.fetchImpl)
    const { result } = renderSession(spy)
    await waitFor(() => expect(result.current.state).toBe('anonymous'))

    backend.state.signedIn = true
    await act(async () => {
      await result.current.api.request('/articles?limit=20')
    })

    expect(spy).toHaveBeenCalledWith(
      '/api/v1/articles?limit=20',
      expect.objectContaining({
        method: 'GET',
        credentials: 'same-origin',
      }),
    )
  })
})
