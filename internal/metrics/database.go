package metrics

// DatabaseMetrics contains all database-related metrics
type DatabaseMetrics struct {
	DBConnections   *Gauge
	DBQueriesTotal  *Counter
	DBQueryDuration *Histogram
	DBErrors        *Counter
}

// NewDatabaseMetrics creates a new DatabaseMetrics instance
func NewDatabaseMetrics() *DatabaseMetrics {
	return &DatabaseMetrics{
		DBConnections:   NewGauge("db_connections_active", "Number of active database connections"),
		DBQueriesTotal:  NewCounter("db_queries_total", "Total number of database queries"),
		DBQueryDuration: NewHistogram("db_query_duration_seconds", "Database query latencies in seconds"),
		DBErrors:        NewCounter("db_errors_total", "Total number of database errors"),
	}
}

// Register registers all database metrics
func (d *DatabaseMetrics) Register(registry MetricRegistry) {
	registry.Register(d.DBConnections)
	registry.Register(d.DBQueriesTotal)
	registry.Register(d.DBQueryDuration)
	registry.Register(d.DBErrors)
}

// RecordQuery records a database query
func (d *DatabaseMetrics) RecordQuery(duration float64, err error) {
	d.DBQueriesTotal.Inc()
	d.DBQueryDuration.Observe(duration)
	if err != nil {
		d.DBErrors.Inc()
	}
}

// SetActiveConnections sets the number of active connections
func (d *DatabaseMetrics) SetActiveConnections(count float64) {
	d.DBConnections.Set(count)
}