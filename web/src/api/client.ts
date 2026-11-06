/**
 * Shared HTTP boundary for the SPA.
 *
 * The browser holds nothing durable: the session lives in an HttpOnly cookie
 * and the CSRF token only exists in memory, so every call is same-origin and
 * unsafe methods carry the in-memory token.
 */

/** Stable shape of one field-level validation failure. */
export interface FieldError {
  field: string
  code: string
}

/**
 * Public error codes the SPA reacts to. The union is open on purpose: an
 * unknown code from a newer backend must still surface as a generic failure
 * instead of crashing the client.
 */
export type ApiErrorCode =
  | 'auth.session_required'
  | 'auth.csrf_failed'
  | 'auth.origin_rejected'
  | 'auth.forbidden'
  | 'request.route_not_found'
  | 'request.method_not_allowed'
  | 'request.invalid_json'
  | 'identity.validation_failed'
  | 'platform.internal_error'
  | 'platform.database_unavailable'
  | 'network.error'
  | (string & {})

/** Normalized failure for every non-2xx response and transport error. */
export class ApiError extends Error {
  readonly status: number
  readonly code: ApiErrorCode
  readonly fieldErrors: FieldError[]
  readonly traceId: string | null

  constructor(init: {
    status: number
    code: ApiErrorCode
    message: string
    fieldErrors?: FieldError[]
    traceId?: string | null
  }) {
    super(init.message)
    this.name = 'ApiError'
    this.status = init.status
    this.code = init.code
    this.fieldErrors = init.fieldErrors ?? []
    this.traceId = init.traceId ?? null
  }
}

/** Minimal response surface the client needs; keeps tests free of real HTTP. */
export interface HttpResponseLike {
  ok: boolean
  status: number
  text(): Promise<string>
  headers?: { get(name: string): string | null }
}

/** Injectable transport so tests never touch the network. */
export type FetchLike = (
  input: string,
  init: RequestInit,
) => Promise<HttpResponseLike>

const defaultFetch: FetchLike = async (input, init) => {
  const response = await fetch(input, init)
  return {
    ok: response.ok,
    status: response.status,
    text: () => response.text(),
    headers: response.headers,
  }
}

const unsafeMethods = new Set(['POST', 'PUT', 'PATCH', 'DELETE'])

const genericFailureMessage = '服务器暂时不可用，请稍后重试'

export interface ApiClientOptions {
  /** Same-origin API prefix; overridable only for tests. */
  basePath?: string
  fetchImpl?: FetchLike
  /** Returns the in-memory CSRF token, or null when no session is held. */
  getCsrfToken?: () => string | null
  /** Called once per 401 so the session layer can drop to anonymous. */
  onUnauthorized?: () => void
}

export interface RequestOptions {
  method?: string
  body?: unknown
  signal?: AbortSignal
  /** Extra request headers, e.g. `If-Match` for optimistic writes. */
  headers?: Record<string, string>
  /**
   * Set when a 401 is an expected answer rather than a dead session, e.g. a
   * failed login. The request still rejects; only the global hook is skipped.
   */
  suppressUnauthorized?: boolean
}

export class ApiClient {
  private readonly basePath: string
  private readonly fetchImpl: FetchLike
  private readonly getCsrfToken: () => string | null
  private readonly onUnauthorized: () => void

  constructor(options: ApiClientOptions = {}) {
    this.basePath = options.basePath ?? '/api/v1'
    this.fetchImpl = options.fetchImpl ?? defaultFetch
    this.getCsrfToken = options.getCsrfToken ?? (() => null)
    this.onUnauthorized = options.onUnauthorized ?? (() => {})
  }

  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const result = await this.execute<T>(path, options)
    return result.data
  }

  /** Same contract as `request`, plus the response `ETag` when present. */
  async requestWithEtag<T>(
    path: string,
    options: RequestOptions = {},
  ): Promise<{ data: T; etag: string | null }> {
    return this.execute<T>(path, options)
  }

  private async execute<T>(
    path: string,
    options: RequestOptions,
  ): Promise<{ data: T; etag: string | null }> {
    const method = (options.method ?? 'GET').toUpperCase()
    const headers: Record<string, string> = {
      Accept: 'application/json',
      ...options.headers,
    }
    if (options.body !== undefined) {
      headers['Content-Type'] = 'application/json'
    }
    if (unsafeMethods.has(method)) {
      const token = this.getCsrfToken()
      if (token) {
        headers['X-CSRF-Token'] = token
      }
    }

    let response: HttpResponseLike
    try {
      response = await this.fetchImpl(this.basePath + path, {
        method,
        headers,
        credentials: 'same-origin',
        body:
          options.body === undefined ? undefined : JSON.stringify(options.body),
        signal: options.signal,
      })
    } catch (error) {
      if (isAbortError(error)) {
        throw error
      }
      throw new ApiError({
        status: 0,
        code: 'network.error',
        message: '无法连接服务器，请检查网络后重试',
      })
    }

    if (response.status === 204) {
      return { data: undefined as T, etag: null }
    }

    const payload = parsePayload(await response.text())
    if (response.ok) {
      return {
        data: payload as T,
        etag: response.headers?.get('ETag') ?? null,
      }
    }

    const failure = toApiError(response.status, payload)
    if (response.status === 401 && !options.suppressUnauthorized) {
      this.onUnauthorized()
    }
    throw failure
  }
}

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === 'AbortError'
}

function parsePayload(raw: string): unknown {
  if (raw.trim() === '') {
    return null
  }
  try {
    return JSON.parse(raw) as unknown
  } catch {
    return null
  }
}

function toApiError(status: number, payload: unknown): ApiError {
  if (!isRecord(payload)) {
    return new ApiError({
      status,
      code: 'platform.internal_error',
      message: genericFailureMessage,
    })
  }
  return new ApiError({
    status,
    code: typeof payload.code === 'string' && payload.code
      ? payload.code
      : 'platform.internal_error',
    message: typeof payload.message === 'string' && payload.message
      ? payload.message
      : genericFailureMessage,
    fieldErrors: readFieldErrors(payload.field_errors),
    traceId: typeof payload.trace_id === 'string' ? payload.trace_id : null,
  })
}

function readFieldErrors(value: unknown): FieldError[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value.flatMap((entry) =>
    isRecord(entry) &&
    typeof entry.field === 'string' &&
    typeof entry.code === 'string'
      ? [{ field: entry.field, code: entry.code }]
      : [],
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
