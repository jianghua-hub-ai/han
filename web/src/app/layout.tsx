import { Link, Outlet } from 'react-router-dom'

import { LogoutButton } from '../components/LogoutButton'

/** Application shell shared by every authenticated route. */
export function AppLayout() {
  return (
    <>
      <header>
        <Link to="/">Han</Link>
        <nav aria-label="主导航">
          <Link to="/">文章</Link>
          <Link to="/me/articles">我的文章</Link>
        </nav>
        <LogoutButton />
      </header>
      <main>
        <Outlet />
      </main>
    </>
  )
}
