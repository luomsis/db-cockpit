#!/usr/bin/env python3
"""
Mock Data Generator for db-cockpit
Generates realistic time-series, alert, and slow query data with configurable time ranges.

Usage:
    python3 generate_mock_data.py                    # Default: 7 days
    python3 generate_mock_data.py --days 15          # 15 days
    python3 generate_mock_data.py --days 30          # 30 days (one month)
    python3 generate_mock_data.py --start-date 2026-03-01 --end-date 2026-03-31  # Custom range
"""

import os
import sys
import random
import hashlib
import argparse
from datetime import datetime, timedelta
from typing import Dict, List, Tuple, Any

# Database connection settings
DB_HOST = "localhost"
DB_PORT = "5432"
DB_USER = "postgres"
DB_PASSWORD = "postgres"
DB_NAME = "postgres"

# Sampling interval
SAMPLING_INTERVAL = 60  # seconds (1 minute)

# Output directory
OUTPUT_DIR = os.path.dirname(os.path.abspath(__file__))


# Metric characteristics by db_type
METRIC_CONFIGS = {
    "mysql": {
        "cpu_usage_percent": {"base": 45, "min": 15, "max": 75, "variance": 15},
        "memory_usage_percent": {"base": 70, "min": 60, "max": 85, "variance": 8},
        "disk_usage_percent": {"base": 65, "min": 55, "max": 75, "variance": 5},
        "connections_active": {"base": 80, "min": 20, "max": 150, "variance": 30},
        "connections_idle": {"base": 25, "min": 5, "max": 50, "variance": 10},
        "queries_per_second": {"base": 400, "min": 100, "max": 800, "variance": 150},
        "replication_lag_seconds": {"base": 0.5, "min": 0, "max": 5, "variance": 1},
        "slow_queries_count": {"base": 5, "min": 0, "max": 20, "variance": 5},
        "replication_status": {"base": 1, "min": 0, "max": 1, "variance": 0},
    },
    "postgresql": {
        "cpu_usage_percent": {"base": 35, "min": 10, "max": 65, "variance": 12},
        "memory_usage_percent": {"base": 65, "min": 50, "max": 80, "variance": 10},
        "disk_usage_percent": {"base": 55, "min": 45, "max": 70, "variance": 8},
        "connections_active": {"base": 50, "min": 15, "max": 100, "variance": 20},
        "connections_idle": {"base": 15, "min": 5, "max": 30, "variance": 8},
        "queries_per_second": {"base": 200, "min": 50, "max": 400, "variance": 80},
        "slow_queries_count": {"base": 3, "min": 0, "max": 15, "variance": 4},
    },
    "oracle": {
        "cpu_usage_percent": {"base": 50, "min": 20, "max": 80, "variance": 18},
        "memory_usage_percent": {"base": 80, "min": 70, "max": 90, "variance": 6},
        "disk_usage_percent": {"base": 70, "min": 60, "max": 85, "variance": 8},
        "connections_active": {"base": 40, "min": 10, "max": 80, "variance": 15},
        "connections_idle": {"base": 10, "min": 2, "max": 20, "variance": 5},
        "queries_per_second": {"base": 100, "min": 30, "max": 200, "variance": 40},
        "slow_queries_count": {"base": 4, "min": 0, "max": 10, "variance": 3},
    },
    "redis": {
        "cpu_usage_percent": {"base": 20, "min": 5, "max": 40, "variance": 10},
        "memory_usage_percent": {"base": 88, "min": 80, "max": 95, "variance": 4},
        "connections_active": {"base": 2000, "min": 100, "max": 5000, "variance": 800},
        "connections_idle": {"base": 50, "min": 10, "max": 100, "variance": 20},
        "queries_per_second": {"base": 25000, "min": 5000, "max": 50000, "variance": 8000},
    },
}

# Environment modifiers
ENV_MODIFIERS = {
    "prod": {"business_hours": 1.3, "night": 0.6, "weekend": 0.7, "alert_freq": 3},
    "test": {"business_hours": 1.1, "night": 0.5, "weekend": 0.5, "alert_freq": 1.5},
    "dev": {"business_hours": 0.8, "night": 0.3, "weekend": 0.3, "alert_freq": 0.5},
}

# Alert threshold and templates
ALERT_THRESHOLDS = {
    "cpu_usage_percent": {"threshold": 80, "text": "CPU使用率超过{threshold}%阈值"},
    "memory_usage_percent": {"threshold": 85, "text": "内存使用率超过{threshold}%"},
    "disk_usage_percent": {"threshold": 80, "text": "磁盘空间不足，使用率达{value}%"},
    "connections_active": {"threshold": 150, "text": "活跃连接数异常，当前{value}个"},
    "replication_lag_seconds": {"threshold": 30, "text": "主从复制延迟超过{value}秒"},
    "slow_queries_count": {"threshold": 15, "text": "慢查询数量异常增加，当前{value}个"},
}

# Slow query SQL templates by db_type
SLOW_QUERY_TEMPLATES = {
    "mysql": [
        ("SELECT * FROM orders WHERE created_at > '{date}' AND status = 'pending'", "order_db"),
        ("UPDATE inventory SET quantity = quantity - 1 WHERE product_id IN (SELECT product_id FROM order_items WHERE order_id = {id})", "order_db"),
        ("SELECT o.*, u.*, p.* FROM orders o JOIN users u ON o.user_id = u.id JOIN products p ON o.product_id = p.id WHERE o.created_at BETWEEN '{start}' AND '{end}'", "order_db"),
        ("SELECT COUNT(*) FROM order_items WHERE order_id IN (SELECT id FROM orders WHERE user_id = {user_id})", "order_db"),
    ],
    "postgresql": {
        "mysql": [
            ("SELECT u.*, o.* FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.created_at > '{date}'", "user_db"),
            ("SELECT * FROM user_sessions WHERE user_id = {id} AND session_start > '{date}'", "user_db"),
            ("SELECT u.name, COUNT(o.id) FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.name HAVING COUNT(o.id) > 10", "user_db"),
            ("SELECT * FROM user_activity WHERE activity_date BETWEEN '{start}' AND '{end}' ORDER BY activity_date", "user_db"),
        ],
    },
    "oracle": [
        ("SELECT * FROM core_employees WHERE hire_date > TO_DATE('{date}', 'YYYY-MM-DD')", "core_db"),
        ("SELECT e.*, d.* FROM core_employees e JOIN departments d ON e.dept_id = d.id WHERE e.salary > {salary}", "core_db"),
        ("SELECT COUNT(*) FROM attendance WHERE employee_id IN (SELECT id FROM core_employees WHERE dept_id = {dept_id})", "core_db"),
        ("SELECT * FROM payroll WHERE pay_date BETWEEN TO_DATE('{start}', 'YYYY-MM-DD') AND TO_DATE('{end}', 'YYYY-MM-DD')", "core_db"),
    ],
    "redis": [
        ("KEYS pattern:user:*", "cache_db"),
        ("GET large:session:data:{id}", "cache_db"),
        ("HGETALL user:profile:{id}", "cache_db"),
        ("LRANGE recent:orders:{user_id} 0 100", "cache_db"),
    ],
}

SLOW_QUERY_TEMPLATES["postgresql"] = [
    ("SELECT u.*, o.* FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE u.created_at > '{date}'", "user_db"),
    ("SELECT * FROM user_sessions WHERE user_id = {id} AND session_start > '{date}'", "user_db"),
    ("SELECT u.name, COUNT(o.id) FROM users u JOIN orders o ON u.id = o.user_id GROUP BY u.name HAVING COUNT(o.id) > 10", "user_db"),
    ("SELECT * FROM user_activity WHERE activity_date BETWEEN '{start}' AND '{end}' ORDER BY activity_date", "user_db"),
]

# Execute time ranges by db_type
EXECUTE_TIME_RANGE = {
    "mysql": (3, 15),
    "postgresql": (2, 10),
    "oracle": (5, 20),
    "redis": (1, 5),
}


def get_time_modifier(dt: datetime, env: str) -> float:
    """Calculate modifier based on time of day and environment."""
    hour = dt.hour
    is_weekend = dt.weekday() >= 5  # Saturday, Sunday

    env_mod = ENV_MODIFIERS.get(env, ENV_MODIFIERS["prod"])

    if is_weekend:
        return env_mod["weekend"]
    elif 9 <= hour <= 18:  # Business hours
        return env_mod["business_hours"]
    elif 0 <= hour <= 6:  # Night
        return env_mod["night"]
    else:  # Evening/morning transition
        return 0.9


def generate_metric_value(metric: str, db_type: str, dt: datetime, env: str) -> float:
    """Generate a realistic metric value based on time and environment."""
    # Get base config, fall back to mysql if not found
    configs = METRIC_CONFIGS.get(db_type, METRIC_CONFIGS["mysql"])
    config = configs.get(metric, {"base": 50, "min": 0, "max": 100, "variance": 10})

    # Check if metric applies to this db_type
    if db_type == "redis":
        if metric in ["disk_usage_percent", "replication_lag_seconds", "replication_status", "slow_queries_count"]:
            return None  # Redis doesn't have these metrics

    if db_type in ["postgresql", "redis"]:
        if metric in ["replication_lag_seconds", "replication_status"]:
            return None  # Only MySQL has replication metrics

    # Calculate base value with time modifier
    time_mod = get_time_modifier(dt, env)
    base = config["base"] * time_mod

    # Add random variance
    variance = config["variance"] * random.uniform(-1, 1)
    value = base + variance

    # Special handling for replication_status (0 or 1)
    if metric == "replication_status":
        # 95% normal (1), 5% abnormal (0)
        return 1 if random.random() > 0.05 else 0

    # Clamp to min/max
    value = max(config["min"], min(config["max"], value))

    # Add some spikes (1% chance of significant deviation)
    if random.random() < 0.01:
        spike_factor = random.uniform(1.5, 2.0) if random.random() > 0.5 else random.uniform(0.3, 0.5)
        value = config["base"] * spike_factor
        value = max(config["min"], min(config["max"], value))

    return round(value, 2)


def generate_series_points(
    series_data: List[Tuple[int, str, str, str]],
    instance_info: Dict[str, Tuple[str, str]],
    start_date: datetime,
    end_date: datetime
) -> List[str]:
    """Generate INSERT statements for series_points."""
    inserts = []
    total_points = 0

    # Calculate total minutes in the time range
    total_minutes = int((end_date - start_date).total_seconds() / SAMPLING_INTERVAL)

    for series_id, endpoint, metric, labels_json in series_data:
        # Get instance info
        db_type, env = instance_info.get(endpoint, ("mysql", "prod"))

        # Generate points for each minute
        current_time = start_date
        while current_time <= end_date:
            value = generate_metric_value(metric, db_type, current_time, env)

            if value is not None:  # Skip metrics not applicable to this db_type
                time_str = current_time.strftime("%Y-%m-%d %H:%M:%S")
                inserts.append(f"('{time_str}', {series_id}, {value})")
                total_points += 1

            current_time += timedelta(seconds=SAMPLING_INTERVAL)

    print(f"Generated {total_points} series_points records")
    return inserts


def generate_alerts(
    series_data: List[Tuple[int, str, str, str]],
    instance_info: Dict[str, Tuple[str, str]],
    start_date: datetime,
    end_date: datetime
) -> List[str]:
    """Generate INSERT statements for alerts."""
    inserts = []
    total_alerts = 0

    # Generate alerts based on metric thresholds
    for series_id, endpoint, metric, labels_json in series_data:
        db_type, env = instance_info.get(endpoint, ("mysql", "prod"))

        # Check if this metric has an alert threshold
        if metric not in ALERT_THRESHOLDS:
            continue

        # Check if metric applies to this db_type
        if db_type == "redis" and metric in ["disk_usage_percent", "replication_lag_seconds", "slow_queries_count"]:
            continue
        if db_type in ["postgresql", "redis"] and metric in ["replication_lag_seconds"]:
            continue

        threshold_config = ALERT_THRESHOLDS[metric]
        threshold = threshold_config["threshold"]
        env_mod = ENV_MODIFIERS.get(env, ENV_MODIFIERS["prod"])

        # Calculate expected alerts per day based on environment
        alerts_per_day = env_mod["alert_freq"]

        # Generate alerts for each day
        current_day = start_date
        while current_day <= end_date:
            day_end = current_day + timedelta(days=1) - timedelta(seconds=1)

            # Determine number of alerts for this day
            num_alerts = int(random.uniform(0.5, alerts_per_day + 0.5))

            for _ in range(num_alerts):
                # Generate alert timing
                alert_hour = random.randint(9, 18) if random.random() > 0.3 else random.randint(0, 23)
                start_time = current_day + timedelta(hours=alert_hour, minutes=random.randint(0, 59))

                # Alert duration: 5 min to 2 hours
                duration_minutes = random.randint(5, 120)
                end_time = start_time + timedelta(minutes=duration_minutes)

                # Generate alert value (above threshold)
                base_config = METRIC_CONFIGS.get(db_type, METRIC_CONFIGS["mysql"]).get(metric, {"max": 100})
                value = threshold + random.uniform(5, base_config["max"] - threshold)

                # Alert status: 40% firing, 60% resolved
                status = "firing" if random.random() < 0.4 else "resolved"

                # Generate event_id
                event_id = hashlib.md5(f"{endpoint}-{metric}-{start_time}".encode()).hexdigest()

                # Format alert text
                alert_text = threshold_config["text"].format(
                    threshold=threshold,
                    value=round(value, 2)
                )

                start_str = start_time.strftime("%Y-%m-%d %H:%M:%S")
                end_str = end_time.strftime("%Y-%m-%d %H:%M:%S")

                inserts.append(f"('{event_id}', '{endpoint}', '{alert_text}', '{start_str}', '{end_str}', '{metric}', '{status}')")
                total_alerts += 1

            current_day += timedelta(days=1)

    print(f"Generated {total_alerts} alert records")
    return inserts


def generate_slow_queries(
    instance_info: Dict[str, Tuple[str, str]],
    start_date: datetime,
    end_date: datetime
) -> List[str]:
    """Generate INSERT statements for slow queries."""
    inserts = []
    total_queries = 0

    for endpoint, (db_type, env) in instance_info.items():
        env_mod = ENV_MODIFIERS.get(env, ENV_MODIFIERS["prod"])

        # Daily query count based on environment
        queries_per_day = env_mod["alert_freq"] * 2  # Slow queries are more frequent

        templates = SLOW_QUERY_TEMPLATES.get(db_type, SLOW_QUERY_TEMPLATES["mysql"])
        time_range = EXECUTE_TIME_RANGE.get(db_type, (3, 10))

        # Generate queries for each day
        current_day = start_date
        while current_day <= end_date:
            num_queries = int(random.uniform(0.5, queries_per_day + 0.5))

            for _ in range(num_queries):
                # Generate query timing
                query_hour = random.randint(9, 18) if random.random() > 0.3 else random.randint(0, 23)
                execute_date = current_day + timedelta(hours=query_hour, minutes=random.randint(0, 59))

                # Select random SQL template
                sql_template, db_name = random.choice(templates)

                # Generate realistic SQL with placeholder values
                sql_text = sql_template.format(
                    date=(execute_date - timedelta(days=random.randint(1, 30))).strftime("%Y-%m-%d"),
                    id=random.randint(1000, 99999),
                    user_id=random.randint(100, 10000),
                    dept_id=random.randint(1, 50),
                    salary=random.randint(5000, 50000),
                    start=(execute_date - timedelta(days=random.randint(7, 30))).strftime("%Y-%m-%d"),
                    end=execute_date.strftime("%Y-%m-%d"),
                )

                # Execute time
                execute_time = round(random.uniform(time_range[0], time_range[1]), 2)

                # Hostname and port extraction from endpoint pattern
                # Pattern: <db_type>-<region>-<num>-<domain>-<suffix>-<num>
                parts = endpoint.split("-")
                hostname = f"{parts[0]}-{parts[1]}-{parts[2]}-{parts[3]}"
                port = random.choice([3306, 5432, 1521, 6379])  # Standard ports by db_type

                if db_type == "mysql":
                    port = 3306
                elif db_type == "postgresql":
                    port = 5432
                elif db_type == "oracle":
                    port = 1521
                elif db_type == "redis":
                    port = 6379

                username = random.choice(["app_user", "admin", "readonly", "service_user"])

                # Escape single quotes in SQL text (PostgreSQL uses '' for escaped quotes)
                sql_text_escaped = sql_text.replace("'", "''")

                date_str = execute_date.strftime("%Y-%m-%d %H:%M:%S")
                inserts.append(f"('{endpoint}', '{hostname}', {port}, '{db_name}', '{username}', '{sql_text_escaped}', {execute_time}, '{date_str}')")
                total_queries += 1

            current_day += timedelta(days=1)

    print(f"Generated {total_queries} slow_query records")
    return inserts


def write_sql_file(filename: str, table_name: str, columns: str, inserts: List[str], batch_size: int = 1000):
    """Write INSERT statements to a SQL file in batches."""
    filepath = os.path.join(OUTPUT_DIR, filename)

    with open(filepath, 'w') as f:
        f.write(f"-- {table_name} INSERT statements\n")
        f.write(f"-- Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        f.write(f"-- Total records: {len(inserts)}\n\n")

        # Write in batches for better performance
        for i in range(0, len(inserts), batch_size):
            batch = inserts[i:i + batch_size]
            f.write(f"INSERT INTO {table_name} ({columns}) VALUES\n")
            f.write(",\n".join(batch))
            f.write(";\n\n")

    print(f"Written {filename} ({len(inserts)} records)")
    return filepath


def parse_args():
    """Parse command line arguments for time range configuration."""
    parser = argparse.ArgumentParser(
        description='Generate mock data for db-cockpit testing',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog='''
Examples:
  %(prog)s                      # Default: 7 days (today - 7 to today)
  %(prog)s --days 15            # 15 days
  %(prog)s --days 30            # 30 days (one month)
  %(prog)s --start-date 2026-03-01 --end-date 2026-03-31  # Custom range

Data volume estimates:
  --days 7:   ~1.6M series_points, ~800 alerts, ~200 slow_queries
  --days 15:  ~3.4M series_points, ~1.7K alerts, ~400 slow_queries
  --days 30:  ~6.8M series_points, ~3.4K alerts, ~800 slow_queries
'''
    )
    parser.add_argument('--days', type=int, default=7,
                        help='Number of days to generate data for (default: 7)')
    parser.add_argument('--start-date', type=str, default=None,
                        help='Start date in YYYY-MM-DD format (default: today - days)')
    parser.add_argument('--end-date', type=str, default=None,
                        help='End date in YYYY-MM-DD format (default: today)')
    return parser.parse_args()


def calculate_time_range(args):
    """Calculate START_DATE and END_DATE from arguments."""
    today = datetime.now().replace(hour=0, minute=0, second=0, microsecond=0)

    if args.start_date and args.end_date:
        # Custom date range
        start_date = datetime.strptime(args.start_date, '%Y-%m-%d')
        end_date = datetime.strptime(args.end_date, '%Y-%m-%d') + timedelta(hours=23, minutes=59, seconds=59)
    elif args.start_date:
        # Start date specified, end date is today
        start_date = datetime.strptime(args.start_date, '%Y-%m-%d')
        end_date = today + timedelta(hours=23, minutes=59, seconds=59)
    elif args.end_date:
        # End date specified, start date is end_date - days
        end_date = datetime.strptime(args.end_date, '%Y-%m-%d') + timedelta(hours=23, minutes=59, seconds=59)
        start_date = (end_date - timedelta(days=args.days)).replace(hour=0, minute=0, second=0)
    else:
        # Default: today - days to today
        end_date = today + timedelta(hours=23, minutes=59, seconds=59)
        start_date = today - timedelta(days=args.days)

    return start_date, end_date


def main():
    """Main function to generate mock data."""
    # Parse arguments and calculate time range
    args = parse_args()
    START_DATE, END_DATE = calculate_time_range(args)

    print("=== Mock Data Generator ===")
    print(f"Time range: {START_DATE.strftime('%Y-%m-%d')} to {END_DATE.strftime('%Y-%m-%d')} ({args.days} days)")
    print(f"Output directory: {OUTPUT_DIR}")
    print()

    # Try to connect to database to get existing data
    try:
        import psycopg2

        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            user=DB_USER,
            password=DB_PASSWORD,
            database=DB_NAME
        )
        cur = conn.cursor()

        # Get instance metadata
        cur.execute("SELECT instance_endpoint, db_type, environment FROM instance_meta")
        instances = cur.fetchall()
        instance_info = {row[0]: (row[1], row[2]) for row in instances}
        print(f"Found {len(instance_info)} instances from instance_meta")

        # Get series metadata
        cur.execute("SELECT id, endpoint, metric, labels::text FROM series_meta ORDER BY endpoint, metric")
        series_data = cur.fetchall()
        print(f"Found {len(series_data)} series from series_meta")

        cur.close()
        conn.close()

    except ImportError:
        print("psycopg2 not installed. Using hardcoded instance data.")
        # Fallback hardcoded data
        instance_info = {
            "mysql-cn-east-1-finance-order-01": ("mysql", "prod"),
            "mysql-cn-east-1-finance-order-02": ("mysql", "prod"),
            "mysql-cn-east-1-finance-order-dev-01": ("mysql", "dev"),
            "mysql-cn-east-1-finance-order-test-01": ("mysql", "test"),
            "oracle-cn-south-1-hr-core-01": ("oracle", "prod"),
            "pg-cn-north-2-ecom-user-01": ("postgresql", "prod"),
            "pg-cn-north-2-ecom-user-02": ("postgresql", "prod"),
            "pg-cn-north-2-ecom-user-test-01": ("postgresql", "test"),
            "redis-cn-north-2-ecom-cache-01": ("redis", "prod"),
            "redis-cn-north-2-ecom-cache-02": ("redis", "prod"),
        }

        # Hardcoded series IDs (approximate)
        series_data = []
        series_id = 87
        for endpoint in instance_info.keys():
            for metric in ["cpu_usage_percent", "memory_usage_percent", "disk_usage_percent",
                          "connections_active", "connections_idle", "queries_per_second"]:
                series_data.append((series_id, endpoint, metric, '{"unit": "percent"}'))
                series_id += 1

    print()

    # Generate cleanup SQL
    cleanup_file = os.path.join(OUTPUT_DIR, "cleanup_data.sql")
    with open(cleanup_file, 'w') as f:
        f.write("-- Cleanup existing data before inserting new mock data\n")
        f.write(f"-- Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
        f.write("-- Delete all series_points\n")
        f.write("DELETE FROM series_points;\n\n")
        f.write("-- Delete all alerts\n")
        f.write("DELETE FROM alert;\n\n")
        f.write("-- Delete all slow queries\n")
        f.write("DELETE FROM slow_query;\n\n")
        f.write("-- Reset sequences (optional, to start IDs from 1)\n")
        f.write("ALTER SEQUENCE alert_id_seq RESTART WITH 1;\n")
        f.write("ALTER SEQUENCE slow_query_id_seq RESTART WITH 1;\n")
    print("Written cleanup_data.sql")

    # Generate series_points
    print("\nGenerating series_points...")
    series_inserts = generate_series_points(series_data, instance_info, START_DATE, END_DATE)
    series_file = write_sql_file(
        "series_points_insert.sql",
        "series_points",
        '"time", series_id, value',
        series_inserts,
        batch_size=5000  # Larger batch for time-series data
    )

    # Generate alerts
    print("\nGenerating alerts...")
    alert_inserts = generate_alerts(series_data, instance_info, START_DATE, END_DATE)
    alert_file = write_sql_file(
        "alert_insert.sql",
        "public.alert",
        "event_id, endpoint, alert_text, start_time, end_time, metric, status",
        alert_inserts,
        batch_size=100
    )

    # Generate slow queries
    print("\nGenerating slow queries...")
    slow_query_inserts = generate_slow_queries(instance_info, START_DATE, END_DATE)
    slow_query_file = write_sql_file(
        "slow_query_insert.sql",
        "public.slow_query",
        "endpoint, hostname, port, database_name, username, sql_text, execute_time, execute_date",
        slow_query_inserts,
        batch_size=50
    )

    print()
    print("=== Generation Complete ===")
    print("\nTo insert the data, run:")
    print(f"  PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f {cleanup_file}")
    print(f"  PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f {series_file}")
    print(f"  PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f {alert_file}")
    print(f"  PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -f {slow_query_file}")

    print("\nOr use the combined script:")
    print(f"  cat {cleanup_file} {series_file} {alert_file} {slow_query_file} | PGPASSWORD=postgres psql -h localhost -U postgres -d postgres")


if __name__ == "__main__":
    main()