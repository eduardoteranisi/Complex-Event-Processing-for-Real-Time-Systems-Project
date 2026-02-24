CREATE TABLE IF NOT EXISTS incident_log (
    critical_event_id UUID PRIMARY KEY,
    critical_event_type VARCHAR(50),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    event_timestamp BIGINT,
    cluster_size INTEGER
);
