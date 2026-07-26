DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'metrics_type') THEN
CREATE TYPE metrics_type AS ENUM ('gauge', 'counter');
END IF;
END$$;

create table IF NOT EXISTS metrics
    (
        id SERIAL PRIMARY KEY,
        metric_id varchar(50) NOT NULL UNIQUE,
        type metrics_type NOT NULL,
        delta integer,
        value double precision,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX IF NOT EXISTS idx_metrics_metric_id ON metrics(metric_id);