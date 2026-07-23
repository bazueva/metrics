CREATE TYPE metrics_type AS ENUM ('gauge', 'counter');

create table metrics
    (
        id SERIAL,
        metric_id varchar(50) NOT NULL UNIQUE,
        type metrics_type NOT NULL,
        delta integer,
        value double precision,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE INDEX idx_metrics_metric_id ON metrics(metric_id);