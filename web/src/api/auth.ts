import type { ApiClient } from './client'

/** The authenticated identity as the backend projects it on login. */
export interface AuthenticatedUser {
  id: string
  username: string
  role: string
  status: string
  isInitialAdmin: boolean
  mustChangePassword: boolean
  mustAcknowledgePrivacy: boolean
}

export interface LoginCredentials {
  username: string
  password: string
}

export interface LoginResult {
  user: AuthenticatedUser
  csrfToken: string
}

export interface AuthApi {
  login(credentials: LoginCredentials): Promise<LoginResult>
  logout(): Promise<void>
}

interface LoginUserPayload {
  id: string
  username: string
  role: string
  status: string
  is_initial_admin: boolean
  must_change_password: boolean
  must_acknowledge_privacy: boolean
}

interface LoginPayload {
  user: LoginUserPayload
  csrf_token: string
}

/**
 * Auth calls over the shared client. The session cookie is set by the server;
 * only the CSRF token crosses back into memory.
 */
export function createAuthApi(client: ApiClient): AuthApi {
  return {
    async login(credentials: LoginCredentials): Promise<LoginResult> {
      const payload = await client.request<LoginPayload>('/sessions', {
        method: 'POST',
        body: credentials,
        // A rejected login is not an expired session.
        suppressUnauthorized: true,
      })
      return {
        user: {
          id: payload.user.id,
          username: payload.user.username,
          role: payload.user.role,
          status: payload.user.status,
          isInitialAdmin: payload.user.is_initial_admin,
          mustChangePassword: payload.user.must_change_password,
          mustAcknowledgePrivacy: payload.user.must_acknowledge_privacy,
        },
        csrfToken: payload.csrf_token,
      }
    },

    async logout(): Promise<void> {
      await client.request<void>('/sessions/logout', { method: 'POST' })
    },
  }
}
