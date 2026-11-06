import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'

import { ApiError } from '../api/client'
import { useSession } from '../app/session'

/** Static copy agreed by the project owner for IR-02 (no notice endpoint). */
export const privacyNoticeVersion = 'v1.0.0'
export const privacyNoticeText =
  '我已阅读并同意《用户隐私声明》（v1.0.0）。我理解并同意：注册时提交的账号名、' +
  '姓名、联系方式将用于账号创建、身份核验与运维通知，仅限授权管理员按最小必要' +
  '原则访问，保存期限见声明第 4 条，我可随时申请查询、更正或删除本人的个人信息。'

const acknowledgementRequired = '请先阅读并同意《用户隐私声明》再提交。'

interface RegistrationPayload {
  username: string
  email: string
  phone: string
  password: string
  privacy_notice_version: string
  privacy_acknowledged: boolean
}

export function RegisterPage() {
  const { api } = useSession()

  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [acknowledged, setAcknowledged] = useState(false)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [registered, setRegistered] = useState(false)

  const complete =
    username !== '' && email !== '' && phone !== '' && password !== ''

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (pending) {
      return
    }
    if (!acknowledged) {
      setError(acknowledgementRequired)
      return
    }
    if (!complete) {
      setError('请完整填写用户名、邮箱、手机号和密码。')
      return
    }

    setPending(true)
    setError(null)
    const payload: RegistrationPayload = {
      username,
      email,
      phone,
      password,
      privacy_notice_version: privacyNoticeVersion,
      privacy_acknowledged: true,
    }
    try {
      await api.request('/users', { method: 'POST', body: payload })
      setPassword('')
      setRegistered(true)
    } catch (caught) {
      setError(registrationFailure(caught))
    } finally {
      setPending(false)
    }
  }

  if (registered) {
    return (
      <section>
        <h1>注册成功</h1>
        <p role="status">账号已创建，请使用新账号登录。</p>
        <Link to="/login">前往登录</Link>
      </section>
    )
  }

  return (
    <section>
      <h1>注册</h1>
      {error !== null ? <p role="alert">{error}</p> : null}
      <form aria-label="注册表单" onSubmit={handleSubmit}>
        <label htmlFor="register-username">用户名</label>
        <input
          id="register-username"
          name="username"
          autoComplete="username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
        />

        <label htmlFor="register-email">邮箱</label>
        <input
          id="register-email"
          name="email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />

        <label htmlFor="register-phone">手机号</label>
        <input
          id="register-phone"
          name="phone"
          type="tel"
          autoComplete="tel"
          value={phone}
          onChange={(event) => setPhone(event.target.value)}
        />

        <label htmlFor="register-password">密码</label>
        <input
          id="register-password"
          name="password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />

        <section aria-labelledby="privacy-notice-heading">
          <h2 id="privacy-notice-heading">
            用户隐私声明（{privacyNoticeVersion}）
          </h2>
          <p>{privacyNoticeText}</p>
          <label htmlFor="register-acknowledge">
            <input
              id="register-acknowledge"
              name="privacy_acknowledged"
              type="checkbox"
              checked={acknowledged}
              onChange={(event) => setAcknowledged(event.target.checked)}
            />
            我已阅读并同意《用户隐私声明》（{privacyNoticeVersion}）
          </label>
        </section>

        <button type="submit" disabled={pending}>
          {pending ? '正在提交…' : '注册'}
        </button>
      </form>
      <p>
        已有账号？<Link to="/login">去登录</Link>
      </p>
    </section>
  )
}

/** Maps stable backend codes to actionable copy without leaking account data. */
function registrationFailure(caught: unknown): string {
  if (!(caught instanceof ApiError)) {
    return '注册失败，请稍后重试。'
  }
  switch (caught.code) {
    case 'identity.identifier_conflict':
      return '用户名、邮箱或手机号已被使用，请更换后重试。'
    case 'identity.validation_failed':
      return '请检查填写内容：用户名、邮箱、手机号需完整且格式正确，密码需满足复杂度要求。'
    case 'privacy.notice_outdated':
      return '隐私声明已更新，请刷新页面后重新确认。'
    case 'privacy.acknowledgement_required':
      return acknowledgementRequired
    case 'platform.processing_gate_closed':
      return '注册暂未开放，请稍后再试。'
    default:
      return caught.status >= 500
        ? '注册失败，请稍后重试。'
        : caught.message
  }
}
