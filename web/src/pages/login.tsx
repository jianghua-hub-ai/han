import { useEffect, useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { ApiError } from '../api/client'
import { useSession } from '../app/session'

const credentialFailure = '登录失败，请检查用户名和密码'
const retryFailure = '登录失败，请稍后重试'

export function LoginPage() {
  const { state, login, expired } = useSession()
  const navigate = useNavigate()
  const location = useLocation()
  const from = readTarget(location.state)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (state === 'authenticated') {
      navigate(from ?? '/', { replace: true })
    }
  }, [state, from, navigate])

  const canSubmit = !pending && username !== '' && password !== ''

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!canSubmit) {
      return
    }
    setPending(true)
    setError(null)
    try {
      await login({ username, password })
    } catch (caught) {
      // Credential, unknown-account, suspended and deleted users all share one
      // message on purpose (US-WEB-01.2.2).
      setError(
        caught instanceof ApiError &&
          caught.status >= 400 &&
          caught.status < 500
          ? credentialFailure
          : retryFailure,
      )
    } finally {
      setPending(false)
    }
  }

  return (
    <section>
      <h1>登录</h1>
      {expired ? <p role="alert">登录已失效，请重新登录。</p> : null}
      {error ? <p role="alert">{error}</p> : null}
      <form aria-label="登录表单" onSubmit={handleSubmit}>
        <label htmlFor="login-username">用户名</label>
        <input
          id="login-username"
          name="username"
          autoComplete="username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
        />

        <label htmlFor="login-password">密码</label>
        <input
          id="login-password"
          name="password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />

        <button type="submit" disabled={!canSubmit}>
          {pending ? '正在登录…' : '登录'}
        </button>
      </form>
      <p>
        还没有账号？<a href="/register">去注册</a>
      </p>
    </section>
  )
}

/** Only same-origin absolute paths are accepted as a post-login target. */
function readTarget(state: unknown): string | null {
  if (typeof state !== 'object' || state === null || !('from' in state)) {
    return null
  }
  const from = (state as { from?: unknown }).from
  return typeof from === 'string' && from.startsWith('/') ? from : null
}
