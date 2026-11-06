import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'

import { createAuthApi, type AuthenticatedUser, type LoginCredentials } from '../api/auth'
import { ApiClient, ApiError, type FetchLike } from '../api/client'

/** The only three session states the UI renders. */
export type SessionState = 'unknown' | 'anonymous' | 'authenticated'

export interface SessionContextValue {
  state: SessionState
  user: AuthenticatedUser | null
  /** In-memory only; never written to storage. */
  csrfToken: string | null
  /** True when the startup probe failed for a non-401 reason. */
  bootstrapFailed: boolean
  /** True when a live session was ended by a 401 rather than by the user. */
  expired: boolean
  api: ApiClient
  login(credentials: LoginCredentials): Promise<void>
  logout(): Promise<void>
  /** Drop to anonymous without calling the backend (used on 401). */
  invalidate(): void
}

const SessionContext = createContext<SessionContextValue | null>(null)

export interface SessionProviderProps {
  children: ReactNode
  /** Test seams; production always uses the same-origin defaults. */
  fetchImpl?: FetchLike
  basePath?: string
}

/**
 * Holds the session in memory and recovers it on reload through one protected
 * read (401 means anonymous). Nothing is persisted: no localStorage, no
 * sessionStorage, no readable cookie access.
 */
export function SessionProvider({
  children,
  fetchImpl,
  basePath,
}: SessionProviderProps) {
  const csrfRef = useRef<string | null>(null)
  const bootstrappedRef = useRef(false)
  const [csrfToken, setCsrfToken] = useState<string | null>(null)
  const [state, setState] = useState<SessionState>('unknown')
  const [user, setUser] = useState<AuthenticatedUser | null>(null)
  const [bootstrapFailed, setBootstrapFailed] = useState(false)
  const [expired, setExpired] = useState(false)

  const clearSession = useCallback(() => {
    csrfRef.current = null
    setCsrfToken(null)
    setUser(null)
  }, [])

  const invalidate = useCallback(() => {
    clearSession()
    setState('anonymous')
    // A 401 during the startup probe simply means "no session"; only a 401
    // after bootstrap means the user's live session ended.
    if (bootstrappedRef.current) {
      setExpired(true)
    }
  }, [clearSession])

  const api = useMemo(
    () =>
      new ApiClient({
        basePath,
        fetchImpl,
        getCsrfToken: () => csrfRef.current,
        onUnauthorized: invalidate,
      }),
    [basePath, fetchImpl, invalidate],
  )

  const auth = useMemo(() => createAuthApi(api), [api])

  const login = useCallback(
    async (credentials: LoginCredentials) => {
      const result = await auth.login(credentials)
      csrfRef.current = result.csrfToken
      setCsrfToken(result.csrfToken)
      setUser(result.user)
      setBootstrapFailed(false)
      setExpired(false)
      setState('authenticated')
    },
    [auth],
  )

  const logout = useCallback(async () => {
    try {
      await auth.logout()
    } finally {
      clearSession()
      setExpired(false)
      setState('anonymous')
    }
  }, [auth, clearSession])

  useEffect(() => {
    let cancelled = false
    const probe = async () => {
      try {
        await api.request('/articles?limit=1')
        if (!cancelled) {
          setBootstrapFailed(false)
          setState('authenticated')
        }
      } catch (error) {
        if (cancelled) {
          return
        }
        if (error instanceof ApiError && error.status === 401) {
          // onUnauthorized already moved the session to anonymous.
          setState('anonymous')
          return
        }
        // A transport or backend failure must not be reported as "logged out".
        setBootstrapFailed(true)
      }
      finally {
        if (!cancelled) {
          bootstrappedRef.current = true
        }
      }
    }
    void probe()
    return () => {
      cancelled = true
    }
  }, [api])

  const value = useMemo<SessionContextValue>(
    () => ({
      state,
      user,
      csrfToken,
      bootstrapFailed,
      expired,
      api,
      login,
      logout,
      invalidate,
    }),
    [
      state,
      user,
      csrfToken,
      bootstrapFailed,
      expired,
      api,
      login,
      logout,
      invalidate,
    ],
  )

  return (
    <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
  )
}

export function useSession(): SessionContextValue {
  const value = useContext(SessionContext)
  if (value === null) {
    throw new Error('useSession must be used inside SessionProvider')
  }
  return value
}
