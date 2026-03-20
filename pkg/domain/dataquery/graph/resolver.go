package graph

import (
	"strconv"

	"github.com/db-cockpit/pkg/domain/dataquery"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	service dataquery.DataQueryService
}

// NewResolver creates a new Resolver with the given service
func NewResolver(service dataquery.DataQueryService) *Resolver {
	return &Resolver{service: service}
}

// Helper functions for conversion

func toSeriesMeta(m dataquery.SeriesMeta) *SeriesMeta {
	return &SeriesMeta{
		ID:         strconv.FormatInt(m.ID, 10),
		Endpoint:   m.Endpoint,
		Metric:     m.Metric,
		Labels:     toLabels(m.Labels),
		LabelsHash: m.LabelsHash,
		CreatedAt:  m.CreatedAt,
	}
}

func toLabels(l map[string]string) *Labels {
	entries := make([]*LabelEntry, 0, len(l))
	keys := make([]string, 0, len(l))
	for k, v := range l {
		entries = append(entries, &LabelEntry{Key: k, Value: v})
		keys = append(keys, k)
	}
	return &Labels{
		Keys:    keys,
		Entries: entries,
	}
}

func toDataPoints(points []dataquery.DataPoint) []*DataPoint {
	result := make([]*DataPoint, len(points))
	for i, p := range points {
		result[i] = &DataPoint{
			Time:  p.Time,
			Value: p.Value,
		}
	}
	return result
}

func toAggregatedPoints(points []dataquery.AggregatedPoint) []*AggregatedPoint {
	result := make([]*AggregatedPoint, len(points))
	for i, p := range points {
		result[i] = &AggregatedPoint{
			Time:  p.Time,
			Value: p.Value,
			Count: p.Count,
		}
	}
	return result
}

func toSeriesStatistics(stats *dataquery.SeriesStatistics) *SeriesStatistics {
	if stats == nil {
		return nil
	}
	return &SeriesStatistics{
		Min:   stats.Min,
		Max:   stats.Max,
		Avg:   stats.Avg,
		Sum:   stats.Sum,
		Count: stats.Count,
	}
}

func toSeries(sd *dataquery.SeriesData) *Series {
	if sd == nil {
		return nil
	}
	return &Series{
		Meta:             toSeriesMeta(sd.Meta),
		Points:           toDataPoints(sd.Points),
		AggregatedPoints: toAggregatedPoints(sd.AggregatedPoints),
		Statistics:       toSeriesStatistics(sd.Statistics),
	}
}

func toSeriesList(series []*dataquery.SeriesData) []*Series {
	result := make([]*Series, len(series))
	for i, s := range series {
		result[i] = toSeries(s)
	}
	return result
}

func parseTimeRange(input TimeRangeInput) dataquery.TimeRange {
	return dataquery.TimeRange{
		Start: input.Start,
		End:   input.End,
	}
}

func parseAggFunction(af AggFunction) dataquery.AggFunction {
	switch af {
	case AggFunctionAvg:
		return dataquery.AggAvg
	case AggFunctionMin:
		return dataquery.AggMin
	case AggFunctionMax:
		return dataquery.AggMax
	case AggFunctionSum:
		return dataquery.AggSum
	case AggFunctionCount:
		return dataquery.AggCount
	default:
		return dataquery.AggAvg
	}
}

// parseID parses a string ID to int64
func parseID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}
