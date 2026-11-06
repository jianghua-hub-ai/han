import type { ApiClient } from './client'

/** Author projection shared by list and detail. */
export interface ArticleAuthor {
  kind: string
  id: string | null
  label?: string
}

export interface ArticleSummary {
  id: string
  title: string
  author: ArticleAuthor
  createdAt: string
  updatedAt: string
}

export interface ArticlePage {
  items: ArticleSummary[]
  nextCursor: string | null
}

export interface ArticleDetail {
  id: string
  title: string
  body: string
  author: ArticleAuthor
  publicationState: string
  violationState: string
  createdAt: string
  updatedAt: string
}

export interface ListArticlesParams {
  cursor?: string | null
  limit?: number
}

export interface ArticlesApi {
  list(params?: ListArticlesParams): Promise<ArticlePage>
  get(id: string): Promise<ArticleDetail>
}

interface SummaryPayload {
  id: string
  title: string
  author: ArticleAuthor
  created_at: string
  updated_at: string
}

interface ListPayload {
  items: SummaryPayload[] | null
  next_cursor: string | null
}

interface DetailPayload {
  id: string
  title: string
  body: string
  author: ArticleAuthor
  publication_state: string
  violation_state: string
  created_at: string
  updated_at: string
}

/** Read-only Article calls; the shared client already handles auth and CSRF. */
export function createArticlesApi(client: ApiClient): ArticlesApi {
  return {
    async list(params: ListArticlesParams = {}): Promise<ArticlePage> {
      const query = new URLSearchParams()
      if (typeof params.limit === 'number') {
        query.set('limit', String(params.limit))
      }
      if (params.cursor) {
        query.set('cursor', params.cursor)
      }
      const suffix = query.toString() === '' ? '' : `?${query.toString()}`
      const payload = await client.request<ListPayload>(`/articles${suffix}`)
      return {
        items: (payload.items ?? []).map((item) => ({
          id: item.id,
          title: item.title,
          author: item.author,
          createdAt: item.created_at,
          updatedAt: item.updated_at,
        })),
        nextCursor: payload.next_cursor ?? null,
      }
    },

    async get(id: string): Promise<ArticleDetail> {
      const payload = await client.request<DetailPayload>(
        `/articles/${encodeURIComponent(id)}`,
      )
      return {
        id: payload.id,
        title: payload.title,
        body: payload.body,
        author: payload.author,
        publicationState: payload.publication_state,
        violationState: payload.violation_state,
        createdAt: payload.created_at,
        updatedAt: payload.updated_at,
      }
    },
  }
}

/** Renders the author projection without inventing a username. */
export function authorLabel(author: ArticleAuthor): string {
  if (author.kind === 'anonymous') {
    return '匿名'
  }
  if (author.label && author.label !== '') {
    return author.label
  }
  return '社区成员'
}
