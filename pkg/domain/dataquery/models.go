package dataquery

import (
	"time"
)

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// SeriesMeta represents metadata for a time series
type SeriesMeta struct {
	ID         int64
	Endpoint   string
	Metric     string
	Labels     map[string]string
	LabelsHash string
	CreatedAt  time.Time
}

// DataPoint represents a single data point in a time series
type DataPoint struct {
	Time  time.Time
	Value float64
}

// AggregatedPoint represents an aggregated data point over a time bucket
type AggregatedPoint struct {
	Time  time.Time
	Value float64
	Count int
}

// SeriesStatistics represents statistical summary of a series
type SeriesStatistics struct {
	Min   float64
	Max   float64
	Avg   float64
	Sum   float64
	Count int
}

// SeriesData represents a complete series with metadata and data points
type SeriesData struct {
	Meta             SeriesMeta
	Points           []DataPoint
	AggregatedPoints []AggregatedPoint
	Statistics       *SeriesStatistics
}

// AggFunction represents aggregation function types
type AggFunction string

const (
	AggAvg   AggFunction = "AVG"
	AggMin   AggFunction = "MIN"
	AggMax   AggFunction = "MAX"
	AggSum   AggFunction = "SUM"
	AggCount AggFunction = "COUNT"
)

// SeriesQuery represents a query for series
type SeriesQuery struct {
	Endpoint    string
	Metric      string
	LabelFilter string
	TimeRange   TimeRange
	Limit       int
}

// MultiSeriesQuery represents a query for multiple series
type MultiSeriesQuery struct {
	Endpoints   []string
	Metrics     []string
	LabelFilter string
	TimeRange   TimeRange
	Aggregation *Aggregation
}

// Aggregation represents aggregation parameters
type Aggregation struct {
	Interval string      // "1m", "5m", "1h", "1d"
	Function AggFunction // AVG, MIN, MAX, SUM, COUNT
}

// SeriesQueryRequest is the repository request for querying series
type SeriesQueryRequest struct {
	Endpoint    string
	Metric      string
	LabelFilter string // raw expression
	TimeRange   TimeRange
	Limit       int
}

// PointsQueryRequest is the repository request for querying data points
type PointsQueryRequest struct {
	SeriesIDs []int64
	TimeRange TimeRange
}

// AggregationRequest is the repository request for aggregated points
type AggregationRequest struct {
	SeriesIDs []int64
	TimeRange TimeRange
	Interval  string
	Function  string
}

// StatsRequest is the repository request for series statistics
type StatsRequest struct {
	SeriesIDs []int64
	TimeRange TimeRange
}
