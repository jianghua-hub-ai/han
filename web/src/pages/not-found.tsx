import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <section>
      <h1>页面不存在</h1>
      <p>
        <Link to="/">返回文章列表</Link>
      </p>
    </section>
  )
}
