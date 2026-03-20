package data

import (
	"context"
	"fmt"

	"github.com/db-cockpit/pkg/common/config"
	"github.com/db-cockpit/pkg/data/neo4j"
	"github.com/db-cockpit/pkg/data/pgmq"
	"github.com/db-cockpit/pkg/data/pgvector"
	"github.com/db-cockpit/pkg/data/redis"
	"github.com/db-cockpit/pkg/data/timescaledb"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DataLayer manages all data layer connections
type DataLayer struct {
	TimescaleDB *timescaledb.TimescaleDBClient
	Neo4j       *neo4j.Neo4jClient
	Redis       *redis.RedisClient
	PgVector    *pgvector.PgVectorClient
	PgMQ        *pgmq.PGMQClient

	// Pool is the shared PostgreSQL connection pool
	// Used by both TimescaleDB operations and PGMQ
	Pool *pgxpool.Pool
}

// NewDataLayer creates a new DataLayer instance
func NewDataLayer(cfg *config.Config) (*DataLayer, error) {
	return &DataLayer{}, nil
}

// Connect establishes all connections
func (d *DataLayer) Connect(ctx context.Context) error {
	// Initialize all data layer clients
	return nil
}

// Close closes all connections
func (d *DataLayer) Close() error {
	if d.Pool != nil {
		d.Pool.Close()
	}
	return nil
}

// Health checks health of all data layer components
func (d *DataLayer) Health(ctx context.Context) map[string]string {
	status := make(map[string]string)

	// Check PostgreSQL pool
	if d.Pool != nil {
		if err := d.Pool.Ping(ctx); err != nil {
			status["postgresql"] = "unhealthy: " + err.Error()
		} else {
			status["postgresql"] = "healthy"
		}
	}

	// Check PGMQ (shares pool with PostgreSQL)
	if d.PgMQ != nil {
		if err := d.PgMQ.Ping(ctx); err != nil {
			status["pgmq"] = "unhealthy: " + err.Error()
		} else {
			status["pgmq"] = "healthy"
		}
	}

	// Check Redis
	if d.Redis != nil {
		if err := d.Redis.Ping(ctx); err != nil {
			status["redis"] = "unhealthy: " + err.Error()
		} else {
			status["redis"] = "healthy"
		}
	}

	// Check Neo4j
	if d.Neo4j != nil {
		if err := d.Neo4j.Ping(ctx); err != nil {
			status["neo4j"] = "unhealthy: " + err.Error()
		} else {
			status["neo4j"] = "healthy"
		}
	}

	// Check PgVector
	if d.PgVector != nil {
		if err := d.PgVector.Ping(ctx); err != nil {
			status["pgvector"] = "unhealthy: " + err.Error()
		} else {
			status["pgvector"] = "healthy"
		}
	}

	return status
}

// InitializeDataLayer initializes and connects all data layer components
func InitializeDataLayer(cfg *config.Config) (*DataLayer, error) {
	dataLayer := &DataLayer{}

	ctx := context.Background()

	// Initialize PostgreSQL connection pool (shared by TimescaleDB, PGMQ, PgVector)
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.TimescaleDB.Host,
		cfg.Database.TimescaleDB.Port,
		cfg.Database.TimescaleDB.User,
		cfg.Database.TimescaleDB.Password,
		cfg.Database.TimescaleDB.Database,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	dataLayer.Pool = pool

	// Initialize TimescaleDB (uses shared pool)
	dataLayer.TimescaleDB = timescaledb.NewTimescaleDBClientWithPool(pool)

	// Initialize PGMQ (uses shared pool)
	dataLayer.PgMQ, err = pgmq.NewPGMQClientWithPool(pool)
	if err != nil {
		return nil, fmt.Errorf("failed to create PGMQ client: %w", err)
	}

	// Initialize PgVector (uses shared pool)
	dataLayer.PgVector = pgvector.NewPgVectorClientWithPool(pool)

	// Initialize Redis
	redisClient, err := redis.NewRedisClient(&cfg.Cache.Redis)
	if err != nil {
		return nil, err
	}
	if err := redisClient.Connect(ctx); err != nil {
		return nil, err
	}
	dataLayer.Redis = redisClient

	// Initialize Neo4j
	neo4jClient, err := neo4j.NewNeo4jClient(&cfg.Database.Neo4j)
	if err != nil {
		return nil, err
	}
	if err := neo4jClient.Connect(ctx); err != nil {
		return nil, err
	}
	dataLayer.Neo4j = neo4jClient

	return dataLayer, nil
}
