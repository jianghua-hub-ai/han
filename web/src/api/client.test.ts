import { describe, expect, it, vi } from 'vitest'

import {
  ApiClient,
  ApiError,
  type FetchLike,
  type HttpResponseLike,
} from './client'

function response(status: number, body?: unknown): HttpResponseLike {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (body === undefined ? '' : JSON.stringify(body)),
  }
}

function spyFetch(result: HttpResponseLike): FetchLike {
  return vi.fn(async () => result)
}

function lastCall(spy: FetchLike): [string, RequestInit] {
  const calls = vi.mocked(spy).mock.calls
  const call = calls[calls.length - 1]
  return [call[0], call[1] as RequestInit]
}

describe('ApiClient', () => {
  it('reads with same-origin credentials and never invents a CSRF header', async () => {
    const spy = spyFetch(response(200, { items: [], next_cursor: null }))
    const client = new ApiClient({
      fetchImpl: spy,
      getCsrfToken: () => 'csrf-token',
    })

    await client.request('/articles?limit=1')

    const [url, init] = lastCall(spy)
    expect(url).toBe('/api/v1/articles?limit=1')
    expect(init.method).toBe('GET')
    expect(init.credentials).toBe('same-origin')
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })

  it('injects the in-memory CSRF token on unsafe methods', async () => {
    const spy = spyFetch(response(204))
    const client = new ApiClient({
      fetchImpl: spy,
      getCsrfToken: () => 'csrf-token',
    })

    const result = await client.request('/sessions/logout', { method: 'POST' })

    expect(result).toBeUndefined()
    const [url, init] = lastCall(spy)
    expect(url).toBe('/api/v1/sessions/logout')
    expect(init.method).toBe('POST')
    expect(init.headers).toMatchObject({ 'X-CSRF-Token': 'csrf-token' })
  })

  it('omits the CSRF header when no session is held', async () => {
    const spy = spyFetch(response(401, {
      code: 'auth.session_required',
      message: '需要重新登录',
      trace_id: 'trace-1',
    }))
    const client = new ApiClient({ fetchImpl: spy })

    await expect(client.request('/sessions/logout', { method: 'POST' })).rejects
      .toThrow(ApiError)

    const [, init] = lastCall(spy)
    expect(init.headers).not.toHaveProperty('X-CSRF-Token')
  })

  it('normalizes an error envelope into ApiError fields', async () => {
    const spy = spyFetch(response(422, {
      code: 'identity.validation_failed',
      message: '请求字段不符合要求',
      field_errors: [{ field: 'password', code: 'password_too_short' }],
      trace_id: 'trace-42',
    }))
    const client = new ApiClient({ fetchImpl: spy })

    const error = await client.request('/users').catch((caught: unknown) => caught)

    expect(error).toBeInstanceOf(ApiError)
    const failure = error as ApiError
    expect(failure.status).toBe(422)
    expect(failure.code).toBe('identity.validation_failed')
    expect(failure.message).toBe('请求字段不符合要求')
    expect(failure.fieldErrors).toEqual([
      { field: 'password', code: 'password_too_short' },
    ])
    expect(failure.traceId).toBe('trace-42')
  })

  it('notifies the session layer exactly once on 401', async () => {
    const onUnauthorized = vi.fn()
    const client = new ApiClient({
      fetchImpl: spyFetch(response(401, {
        code: 'auth.session_required',
        message: '需要重新登录',
        trace_id: 'trace-401',
      })),
      onUnauthorized,
    })

    await expect(client.request('/articles')).rejects.toBeInstanceOf(ApiError)

    expect(onUnauthorized).toHaveBeenCalledTimes(1)
  })

  it('reports transport failures as a network error instead of a 401', async () => {
    const client = new ApiClient({
      fetchImpl: async () => {
        throw new TypeError('Failed to fetch')
      },
    })

    const error = await client.request('/articles').catch((caught: unknown) => caught)

    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).status).toBe(0)
    expect((error as ApiError).code).toBe('network.error')
  })

  it('falls back to a generic failure when the error body is not JSON', async () => {
    const client = new ApiClient({
      fetchImpl: async () => ({
        ok: false,
        status: 502,
        text: async () => '<html>bad gateway</html>',
      }),
    })

    const error = await client.request('/articles').catch((caught: unknown) => caught)

    expect((error as ApiError).code).toBe('platform.internal_error')
    expect((error as ApiError).message).toContain('服务器')
  })
})
