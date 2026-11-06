import { useState } from 'react'

import { useSession } from '../app/session'

/** Ends the server session; the UI lands on anonymous even if the call fails. */
export function LogoutButton() {
  const { logout } = useSession()
  const [pending, setPending] = useState(false)

  async function handleClick() {
    setPending(true)
    try {
      await logout()
    } catch {
      // The session is already cleared locally; a failed revoke is reported by
      // the next protected request, not by this button.
    } finally {
      setPending(false)
    }
  }

  return (
    <button type="button" onClick={handleClick} disabled={pending}>
      {pending ? '正在退出…' : '退出登录'}
    </button>
  )
}
