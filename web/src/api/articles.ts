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

/** The strict author-controlled projection used by the editor. */
export interface AuthorArticle {
  title: string
  body: string
  moderationStatus: string
  editable: boolean
}

export interface ArticleWriteInput {
  title: string
  body: string
}

export interface ArticlesApi {
  list(params?: ListArticlesParams): Promise<ArticlePage>
  listMine(params?: ListArticlesParams): Promise<ArticlePage>
  get(id: string): Promise<ArticleDetail>
  getAuthor(id: string): Promise<{ article: AuthorArticle; etag: string | null }>
  create(input: ArticleWriteInput): Promise<{ id: string; etag: string | null }>
  update(
    id: string,
    input: ArticleWriteInput,
    etag: string,
  ): Promise<{ article: AuthorArticle; etag: string | null }>
  remove(id: string): Promise<void>
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

interface AuthorPayload {
  title: string
  body: string
  moderation_status: string
  editable: boolean
}

function readListQuery(params: ListArticlesParams): string {
  const query = new URLSearchParams()
  if (typeof params.limit === 'number') {
    query.set('limit', String(params.limit))
  }
  if (params.cursor) {
    query.set('cursor', params.cursor)
  }
  return query.toString() === '' ? '' : `?${query.toString()}`
}

function mapPage(payload: ListPayload): ArticlePage {
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
}

function mapAuthor(payload: AuthorPayload): AuthorArticle {
  return {
    title: payload.title,
    body: payload.body,
    moderationStatus: payload.moderation_status,
    editable: payload.editable,
  }
}

/** Read-only Article calls; the shared client already handles auth and CSRF. */
export function createArticlesApi(client: ApiClient): ArticlesApi {
  return {
    async list(params: ListArticlesParams = {}): Promise<ArticlePage> {
      const payload = await client.request<ListPayload>(
        `/articles${readListQuery(params)}`,
      )
      return mapPage(payload)
    },

    async listMine(params: ListArticlesParams = {}): Promise<ArticlePage> {
      const payload = await client.request<ListPayload>(
        `/users/me/articles${readListQuery(params)}`,
      )
      return mapPage(payload)
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

    async getAuthor(id: string) {
      const result = await client.requestWithEtag<AuthorPayload>(
        `/users/me/articles/${encodeURIComponent(id)}`,
      )
      return { article: mapAuthor(result.data), etag: result.etag }
    },

    async create(input: ArticleWriteInput) {
      const result = await client.requestWithEtag<{ id: string }>(
        '/articles',
        { method: 'POST', body: input },
      )
      return { id: result.data.id, etag: result.etag }
    },

    async update(id: string, input: ArticleWriteInput, etag: string) {
      const result = await client.requestWithEtag<AuthorPayload>(
        `/articles/${encodeURIComponent(id)}`,
        {
          method: 'PUT',
          body: input,
          headers: { 'If-Match': etag },
        },
      )
      return { article: mapAuthor(result.data), etag: result.etag }
    },

    async remove(id: string) {
      await client.request<void>(`/articles/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      })
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
