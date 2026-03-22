import { gql } from 'graphql-request'

export const GET_ENDPOINTS = gql`
  query GetEndpoints {
    endpoints
  }
`

export const GET_METRICS = gql`
  query GetMetrics($endpoint: String!) {
    metrics(endpoint: $endpoint)
  }
`

export const GET_SERIES_DATA = gql`
  query GetSeriesData($endpoint: String, $metric: String, $timeRange: TimeRangeInput!) {
    series(
      endpoint: $endpoint
      metric: $metric
      timeRange: $timeRange
      limit: 10
    ) {
      meta {
        id
        endpoint
        metric
        labels {
          entries {
            key
            value
          }
        }
      }
      points {
        time
        value
      }
    }
  }
`