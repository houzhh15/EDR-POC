package opensearch

// SortOrder 排序方向
type SortOrder string

const (
	// Asc 升序
	Asc SortOrder = "asc"
	// Desc 降序
	Desc SortOrder = "desc"
)

// QueryBuilder 查询构建器接口
type QueryBuilder interface {
	// Bool 创建布尔查询
	Bool() BoolQueryBuilder

	// Term 精确匹配
	Term(field string, value interface{}) QueryBuilder

	// Terms 多值精确匹配
	Terms(field string, values ...interface{}) QueryBuilder

	// Range 范围查询
	Range(field string) RangeQueryBuilder

	// Match 全文匹配
	Match(field string, text string) QueryBuilder

	// MatchPhrase 短语匹配
	MatchPhrase(field string, text string) QueryBuilder

	// Wildcard 通配符查询
	Wildcard(field string, pattern string) QueryBuilder

	// Prefix 前缀查询
	Prefix(field string, prefix string) QueryBuilder

	// Exists 字段存在查询
	Exists(field string) QueryBuilder

	// Sort 添加排序
	Sort(field string, order SortOrder) QueryBuilder

	// SortWithOptions 添加带选项的排序
	SortWithOptions(field string, options map[string]interface{}) QueryBuilder

	// From 设置分页起始位置
	From(offset int) QueryBuilder

	// Size 设置返回数量
	Size(limit int) QueryBuilder

	// Source 设置返回字段
	Source(includes ...string) QueryBuilder

	// SourceExcludes 设置排除字段
	SourceExcludes(excludes ...string) QueryBuilder

	// Aggregation 添加聚合
	Aggregation(name string, agg Aggregation) QueryBuilder

	// Highlight 添加高亮
	Highlight(fields ...string) QueryBuilder

	// TrackTotalHits 追踪总命中数
	TrackTotalHits(track bool) QueryBuilder

	// Build 构建 DSL
	Build() map[string]interface{}
}

// BoolQueryBuilder 布尔查询构建器
type BoolQueryBuilder interface {
	// Must 必须匹配
	Must(queries ...interface{}) BoolQueryBuilder

	// Should 应该匹配
	Should(queries ...interface{}) BoolQueryBuilder

	// MustNot 必须不匹配
	MustNot(queries ...interface{}) BoolQueryBuilder

	// Filter 过滤器（不计算分数）
	Filter(queries ...interface{}) BoolQueryBuilder

	// MinimumShouldMatch 设置最小匹配数
	MinimumShouldMatch(n int) BoolQueryBuilder

	// Build 构建 bool 查询
	Build() map[string]interface{}

	// Done 返回父构建器
	Done() QueryBuilder
}

// RangeQueryBuilder 范围查询构建器
type RangeQueryBuilder interface {
	// Gt 大于
	Gt(value interface{}) RangeQueryBuilder

	// Gte 大于等于
	Gte(value interface{}) RangeQueryBuilder

	// Lt 小于
	Lt(value interface{}) RangeQueryBuilder

	// Lte 小于等于
	Lte(value interface{}) RangeQueryBuilder

	// Format 日期格式
	Format(format string) RangeQueryBuilder

	// TimeZone 时区
	TimeZone(tz string) RangeQueryBuilder

	// Build 构建 range 查询
	Build() map[string]interface{}

	// Done 返回父构建器
	Done() QueryBuilder
}

// Aggregation 聚合接口
type Aggregation interface {
	Build() map[string]interface{}
}

// queryBuilder QueryBuilder 实现
type queryBuilder struct {
	query          map[string]interface{}
	sorts          []interface{}
	from           *int
	size           *int
	source         interface{}
	aggregations   map[string]Aggregation
	highlight      map[string]interface{}
	trackTotalHits *bool
}

// NewQuery 创建新的查询构建器
func NewQuery() QueryBuilder {
	return &queryBuilder{
		aggregations: make(map[string]Aggregation),
	}
}

// Bool 创建布尔查询
func (qb *queryBuilder) Bool() BoolQueryBuilder {
	return &boolQueryBuilder{
		parent: qb,
	}
}

// Term 精确匹配
func (qb *queryBuilder) Term(field string, value interface{}) QueryBuilder {
	qb.query = map[string]interface{}{
		"term": map[string]interface{}{
			field: value,
		},
	}
	return qb
}

// Terms 多值精确匹配
func (qb *queryBuilder) Terms(field string, values ...interface{}) QueryBuilder {
	qb.query = map[string]interface{}{
		"terms": map[string]interface{}{
			field: values,
		},
	}
	return qb
}

// Range 范围查询
func (qb *queryBuilder) Range(field string) RangeQueryBuilder {
	return &rangeQueryBuilder{
		parent: qb,
		field:  field,
		params: make(map[string]interface{}),
	}
}

// Match 全文匹配
func (qb *queryBuilder) Match(field string, text string) QueryBuilder {
	qb.query = map[string]interface{}{
		"match": map[string]interface{}{
			field: text,
		},
	}
	return qb
}

// MatchPhrase 短语匹配
func (qb *queryBuilder) MatchPhrase(field string, text string) QueryBuilder {
	qb.query = map[string]interface{}{
		"match_phrase": map[string]interface{}{
			field: text,
		},
	}
	return qb
}

// Wildcard 通配符查询
func (qb *queryBuilder) Wildcard(field string, pattern string) QueryBuilder {
	qb.query = map[string]interface{}{
		"wildcard": map[string]interface{}{
			field: pattern,
		},
	}
	return qb
}

// Prefix 前缀查询
func (qb *queryBuilder) Prefix(field string, prefix string) QueryBuilder {
	qb.query = map[string]interface{}{
		"prefix": map[string]interface{}{
			field: prefix,
		},
	}
	return qb
}

// Exists 字段存在查询
func (qb *queryBuilder) Exists(field string) QueryBuilder {
	qb.query = map[string]interface{}{
		"exists": map[string]interface{}{
			"field": field,
		},
	}
	return qb
}

// Sort 添加排序
func (qb *queryBuilder) Sort(field string, order SortOrder) QueryBuilder {
	qb.sorts = append(qb.sorts, map[string]interface{}{
		field: map[string]interface{}{
			"order": string(order),
		},
	})
	return qb
}

// SortWithOptions 添加带选项的排序
func (qb *queryBuilder) SortWithOptions(field string, options map[string]interface{}) QueryBuilder {
	qb.sorts = append(qb.sorts, map[string]interface{}{
		field: options,
	})
	return qb
}

// From 设置分页起始位置
func (qb *queryBuilder) From(offset int) QueryBuilder {
	qb.from = &offset
	return qb
}

// Size 设置返回数量
func (qb *queryBuilder) Size(limit int) QueryBuilder {
	qb.size = &limit
	return qb
}

// Source 设置返回字段
func (qb *queryBuilder) Source(includes ...string) QueryBuilder {
	if qb.source == nil {
		qb.source = map[string]interface{}{}
	}
	if m, ok := qb.source.(map[string]interface{}); ok {
		m["includes"] = includes
	}
	return qb
}

// SourceExcludes 设置排除字段
func (qb *queryBuilder) SourceExcludes(excludes ...string) QueryBuilder {
	if qb.source == nil {
		qb.source = map[string]interface{}{}
	}
	if m, ok := qb.source.(map[string]interface{}); ok {
		m["excludes"] = excludes
	}
	return qb
}

// Aggregation 添加聚合
func (qb *queryBuilder) Aggregation(name string, agg Aggregation) QueryBuilder {
	qb.aggregations[name] = agg
	return qb
}

// Highlight 添加高亮
func (qb *queryBuilder) Highlight(fields ...string) QueryBuilder {
	highlightFields := make(map[string]interface{})
	for _, field := range fields {
		highlightFields[field] = map[string]interface{}{}
	}
	qb.highlight = map[string]interface{}{
		"fields": highlightFields,
	}
	return qb
}

// TrackTotalHits 追踪总命中数
func (qb *queryBuilder) TrackTotalHits(track bool) QueryBuilder {
	qb.trackTotalHits = &track
	return qb
}

// Build 构建 DSL
func (qb *queryBuilder) Build() map[string]interface{} {
	result := make(map[string]interface{})

	if qb.query != nil {
		result["query"] = qb.query
	}

	if len(qb.sorts) > 0 {
		result["sort"] = qb.sorts
	}

	if qb.from != nil {
		result["from"] = *qb.from
	}

	if qb.size != nil {
		result["size"] = *qb.size
	}

	if qb.source != nil {
		result["_source"] = qb.source
	}

	if len(qb.aggregations) > 0 {
		aggs := make(map[string]interface{})
		for name, agg := range qb.aggregations {
			aggs[name] = agg.Build()
		}
		result["aggs"] = aggs
	}

	if qb.highlight != nil {
		result["highlight"] = qb.highlight
	}

	if qb.trackTotalHits != nil {
		result["track_total_hits"] = *qb.trackTotalHits
	}

	return result
}

// boolQueryBuilder BoolQueryBuilder 实现
type boolQueryBuilder struct {
	parent             *queryBuilder
	must               []interface{}
	should             []interface{}
	mustNot            []interface{}
	filter             []interface{}
	minimumShouldMatch *int
}

// Must 必须匹配
func (bqb *boolQueryBuilder) Must(queries ...interface{}) BoolQueryBuilder {
	for _, q := range queries {
		bqb.must = append(bqb.must, normalizeQuery(q))
	}
	return bqb
}

// Should 应该匹配
func (bqb *boolQueryBuilder) Should(queries ...interface{}) BoolQueryBuilder {
	for _, q := range queries {
		bqb.should = append(bqb.should, normalizeQuery(q))
	}
	return bqb
}

// MustNot 必须不匹配
func (bqb *boolQueryBuilder) MustNot(queries ...interface{}) BoolQueryBuilder {
	for _, q := range queries {
		bqb.mustNot = append(bqb.mustNot, normalizeQuery(q))
	}
	return bqb
}

// Filter 过滤器
func (bqb *boolQueryBuilder) Filter(queries ...interface{}) BoolQueryBuilder {
	for _, q := range queries {
		bqb.filter = append(bqb.filter, normalizeQuery(q))
	}
	return bqb
}

// MinimumShouldMatch 设置最小匹配数
func (bqb *boolQueryBuilder) MinimumShouldMatch(n int) BoolQueryBuilder {
	bqb.minimumShouldMatch = &n
	return bqb
}

// Build 构建 bool 查询
func (bqb *boolQueryBuilder) Build() map[string]interface{} {
	bool := make(map[string]interface{})

	if len(bqb.must) > 0 {
		bool["must"] = bqb.must
	}
	if len(bqb.should) > 0 {
		bool["should"] = bqb.should
	}
	if len(bqb.mustNot) > 0 {
		bool["must_not"] = bqb.mustNot
	}
	if len(bqb.filter) > 0 {
		bool["filter"] = bqb.filter
	}
	if bqb.minimumShouldMatch != nil {
		bool["minimum_should_match"] = *bqb.minimumShouldMatch
	}

	return map[string]interface{}{
		"bool": bool,
	}
}

// Done 返回父构建器
func (bqb *boolQueryBuilder) Done() QueryBuilder {
	bqb.parent.query = bqb.Build()
	return bqb.parent
}

// rangeQueryBuilder RangeQueryBuilder 实现
type rangeQueryBuilder struct {
	parent *queryBuilder
	field  string
	params map[string]interface{}
}

// Gt 大于
func (rqb *rangeQueryBuilder) Gt(value interface{}) RangeQueryBuilder {
	rqb.params["gt"] = value
	return rqb
}

// Gte 大于等于
func (rqb *rangeQueryBuilder) Gte(value interface{}) RangeQueryBuilder {
	rqb.params["gte"] = value
	return rqb
}

// Lt 小于
func (rqb *rangeQueryBuilder) Lt(value interface{}) RangeQueryBuilder {
	rqb.params["lt"] = value
	return rqb
}

// Lte 小于等于
func (rqb *rangeQueryBuilder) Lte(value interface{}) RangeQueryBuilder {
	rqb.params["lte"] = value
	return rqb
}

// Format 日期格式
func (rqb *rangeQueryBuilder) Format(format string) RangeQueryBuilder {
	rqb.params["format"] = format
	return rqb
}

// TimeZone 时区
func (rqb *rangeQueryBuilder) TimeZone(tz string) RangeQueryBuilder {
	rqb.params["time_zone"] = tz
	return rqb
}

// Build 构建 range 查询
func (rqb *rangeQueryBuilder) Build() map[string]interface{} {
	return map[string]interface{}{
		"range": map[string]interface{}{
			rqb.field: rqb.params,
		},
	}
}

// Done 返回父构建器
func (rqb *rangeQueryBuilder) Done() QueryBuilder {
	rqb.parent.query = rqb.Build()
	return rqb.parent
}

// normalizeQuery 标准化查询对象
func normalizeQuery(q interface{}) interface{} {
	switch v := q.(type) {
	case QueryBuilder:
		return v.Build()["query"]
	case BoolQueryBuilder:
		return v.Build()
	case RangeQueryBuilder:
		return v.Build()
	case map[string]interface{}:
		return v
	default:
		return v
	}
}

// ============ 便捷查询函数 ============

// TermQuery 创建 term 查询
func TermQuery(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"term": map[string]interface{}{
			field: value,
		},
	}
}

// TermsQuery 创建 terms 查询
func TermsQuery(field string, values ...interface{}) map[string]interface{} {
	return map[string]interface{}{
		"terms": map[string]interface{}{
			field: values,
		},
	}
}

// MatchQuery 创建 match 查询
func MatchQuery(field string, text string) map[string]interface{} {
	return map[string]interface{}{
		"match": map[string]interface{}{
			field: text,
		},
	}
}

// RangeGte 创建 range >= 查询
func RangeGte(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"range": map[string]interface{}{
			field: map[string]interface{}{
				"gte": value,
			},
		},
	}
}

// RangeLte 创建 range <= 查询
func RangeLte(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"range": map[string]interface{}{
			field: map[string]interface{}{
				"lte": value,
			},
		},
	}
}

// RangeBetween 创建范围查询
func RangeBetween(field string, gte, lte interface{}) map[string]interface{} {
	return map[string]interface{}{
		"range": map[string]interface{}{
			field: map[string]interface{}{
				"gte": gte,
				"lte": lte,
			},
		},
	}
}

// ExistsQuery 创建 exists 查询
func ExistsQuery(field string) map[string]interface{} {
	return map[string]interface{}{
		"exists": map[string]interface{}{
			"field": field,
		},
	}
}

// WildcardQuery 创建 wildcard 查询
func WildcardQuery(field string, pattern string) map[string]interface{} {
	return map[string]interface{}{
		"wildcard": map[string]interface{}{
			field: pattern,
		},
	}
}

// ============ 聚合类型 ============

// TermsAgg terms 聚合
type TermsAgg struct {
	Field string
	Size  int
	Order map[string]string
	Aggs  map[string]Aggregation
}

// Build 构建 terms 聚合
func (a *TermsAgg) Build() map[string]interface{} {
	terms := map[string]interface{}{
		"field": a.Field,
	}
	if a.Size > 0 {
		terms["size"] = a.Size
	}
	if a.Order != nil {
		terms["order"] = a.Order
	}

	result := map[string]interface{}{
		"terms": terms,
	}

	if len(a.Aggs) > 0 {
		aggs := make(map[string]interface{})
		for name, agg := range a.Aggs {
			aggs[name] = agg.Build()
		}
		result["aggs"] = aggs
	}

	return result
}

// DateHistogramAgg date_histogram 聚合
type DateHistogramAgg struct {
	Field            string
	CalendarInterval string
	FixedInterval    string
	Format           string
	TimeZone         string
	MinDocCount      int
	Aggs             map[string]Aggregation
}

// Build 构建 date_histogram 聚合
func (a *DateHistogramAgg) Build() map[string]interface{} {
	histogram := map[string]interface{}{
		"field": a.Field,
	}
	if a.CalendarInterval != "" {
		histogram["calendar_interval"] = a.CalendarInterval
	}
	if a.FixedInterval != "" {
		histogram["fixed_interval"] = a.FixedInterval
	}
	if a.Format != "" {
		histogram["format"] = a.Format
	}
	if a.TimeZone != "" {
		histogram["time_zone"] = a.TimeZone
	}
	if a.MinDocCount >= 0 {
		histogram["min_doc_count"] = a.MinDocCount
	}

	result := map[string]interface{}{
		"date_histogram": histogram,
	}

	if len(a.Aggs) > 0 {
		aggs := make(map[string]interface{})
		for name, agg := range a.Aggs {
			aggs[name] = agg.Build()
		}
		result["aggs"] = aggs
	}

	return result
}

// CardinalityAgg cardinality 聚合
type CardinalityAgg struct {
	Field              string
	PrecisionThreshold int
}

// Build 构建 cardinality 聚合
func (a *CardinalityAgg) Build() map[string]interface{} {
	cardinality := map[string]interface{}{
		"field": a.Field,
	}
	if a.PrecisionThreshold > 0 {
		cardinality["precision_threshold"] = a.PrecisionThreshold
	}
	return map[string]interface{}{
		"cardinality": cardinality,
	}
}

// SumAgg sum 聚合
type SumAgg struct {
	Field string
}

// Build 构建 sum 聚合
func (a *SumAgg) Build() map[string]interface{} {
	return map[string]interface{}{
		"sum": map[string]interface{}{
			"field": a.Field,
		},
	}
}

// AvgAgg avg 聚合
type AvgAgg struct {
	Field string
}

// Build 构建 avg 聚合
func (a *AvgAgg) Build() map[string]interface{} {
	return map[string]interface{}{
		"avg": map[string]interface{}{
			"field": a.Field,
		},
	}
}

// MaxAgg max 聚合
type MaxAgg struct {
	Field string
}

// Build 构建 max 聚合
func (a *MaxAgg) Build() map[string]interface{} {
	return map[string]interface{}{
		"max": map[string]interface{}{
			"field": a.Field,
		},
	}
}

// MinAgg min 聚合
type MinAgg struct {
	Field string
}

// Build 构建 min 聚合
func (a *MinAgg) Build() map[string]interface{} {
	return map[string]interface{}{
		"min": map[string]interface{}{
			"field": a.Field,
		},
	}
}

// FilterAgg filter 聚合
type FilterAgg struct {
	Filter map[string]interface{}
	Aggs   map[string]Aggregation
}

// Build 构建 filter 聚合
func (a *FilterAgg) Build() map[string]interface{} {
	result := map[string]interface{}{
		"filter": a.Filter,
	}

	if len(a.Aggs) > 0 {
		aggs := make(map[string]interface{})
		for name, agg := range a.Aggs {
			aggs[name] = agg.Build()
		}
		result["aggs"] = aggs
	}

	return result
}

// TopHitsAgg top_hits 聚合
type TopHitsAgg struct {
	Size   int
	Sort   []interface{}
	Source interface{}
}

// Build 构建 top_hits 聚合
func (a *TopHitsAgg) Build() map[string]interface{} {
	topHits := make(map[string]interface{})
	if a.Size > 0 {
		topHits["size"] = a.Size
	}
	if len(a.Sort) > 0 {
		topHits["sort"] = a.Sort
	}
	if a.Source != nil {
		topHits["_source"] = a.Source
	}
	return map[string]interface{}{
		"top_hits": topHits,
	}
}
