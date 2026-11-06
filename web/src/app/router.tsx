import { useMemo } from 'react'
import {
  createBrowserRouter,
  Navigate,
  Outlet,
  RouterProvider,
  useLocation,
  type RouteObject,
} from 'react-router-dom'

import { ArticleDetailPage } from '../pages/article-detail'
import { ArticleEditorPage } from '../pages/article-editor'
import { ArticleListPage } from '../pages/article-list'
import { LoginPage } from '../pages/login'
import { MyArticlesPage } from '../pages/my-articles'
import { NotFoundPage } from '../pages/not-found'
import { RegisterPage } from '../pages/register'
import { AppLayout } from './layout'
import { useSession } from './session'

/**
 * Blocks protected routes until the session is known. While the startup probe
 * runs, the skeleton keeps the user from being bounced to /login by a flicker;
 * an anonymous visitor is redirected with the target path preserved.
 */
function RequireSession() {
  const { state } = useSession()
  const location = useLocation()

  if (state === 'unknown') {
    return <p role="status">正在恢复登录状态…</p>
  }
  if (state === 'anonymous') {
    return (
      <Navigate
        to="/login"
        replace
        state={{ from: location.pathname + location.search }}
      />
    )
  }
  return <Outlet />
}

/** The SPA route table; exported so tests can mount it in memory. */
export const routes: RouteObject[] = [
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPage /> },
  {
    element: <RequireSession />,
    children: [
      {
        element: <AppLayout />,
        children: [
          { index: true, element: <ArticleListPage /> },
          { path: 'articles/:id', element: <ArticleDetailPage /> },
          { path: 'me/articles', element: <MyArticlesPage /> },
          { path: 'me/articles/new', element: <ArticleEditorPage /> },
          { path: 'me/articles/:id/edit', element: <ArticleEditorPage /> },
        ],
      },
    ],
  },
  { path: '*', element: <NotFoundPage /> },
]

/** Production entry: one browser router for the whole app. */
export function AppRouter() {
  const router = useMemo(() => createBrowserRouter(routes), [])
  return <RouterProvider router={router} />
}
