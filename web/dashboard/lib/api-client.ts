const API_BASE_URL = '/api'

// 开发环境使用模拟token
const DEV_TOKEN = 'dev_tenant:dev_user:admin'

async function fetchAPI<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${DEV_TOKEN}`,
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: { message: 'Unknown error' } }))
    throw new Error(error.error?.message || `HTTP ${response.status}`)
  }

  return response.json()
}

// Types
export interface EndpointsResponse {
  data: string[]
}

export interface MetricsResponse {
  data: string[]
}

export interface SeriesResponse {
  data: SeriesData[]
}

export interface SeriesData {
  id: string
  endpoint: string
  metric: string
  labels: Record<string, string>
  labels_hash: string
  created_at: string
  points: DataPoint[]
  aggregated_points?: AggregatedPoint[]
  statistics?: SeriesStatistics
}

export interface DataPoint {
  time: string
  value: number
}

export interface AggregatedPoint {
  time: string
  value: number
  count: number
}

export interface SeriesStatistics {
  min: number
  max: number
  avg: number
  sum: number
  count: number
}

export interface SeriesQueryRequest {
  endpoints?: string[]
  metrics?: string[]
  labels?: string
  start: string
  end: string
  aggregation?: {
    interval: string
    function: 'AVG' | 'MIN' | 'MAX' | 'SUM' | 'COUNT'
  }
}

// API functions
export async function getEndpoints(): Promise<string[]> {
  const response = await fetchAPI<EndpointsResponse>('/v1/endpoints')
  return response.data || []
}

export async function getMetrics(endpoint: string): Promise<string[]> {
  const response = await fetchAPI<MetricsResponse>(
    `/v1/metrics?endpoint=${encodeURIComponent(endpoint)}`
  )
  return response.data || []
}

export async function getSeries(params: {
  endpoint?: string
  metric?: string
  labels?: string
  start: string
  end: string
  limit?: number
}): Promise<SeriesData[]> {
  const queryParams = new URLSearchParams()
  queryParams.set('start', params.start)
  queryParams.set('end', params.end)
  if (params.endpoint) queryParams.set('endpoint', params.endpoint)
  if (params.metric) queryParams.set('metric', params.metric)
  if (params.labels) queryParams.set('labels', params.labels)
  if (params.limit) queryParams.set('limit', params.limit.toString())

  const response = await fetchAPI<SeriesResponse>(`/v1/series?${queryParams.toString()}`)
  return response.data || []
}

export async function querySeries(request: SeriesQueryRequest): Promise<SeriesData[]> {
  const response = await fetchAPI<SeriesResponse>('/v1/series/query', {
    method: 'POST',
    body: JSON.stringify(request),
  })
  return response.data || []
}