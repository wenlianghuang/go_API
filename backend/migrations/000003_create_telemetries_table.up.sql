CREATE TABLE telemetries (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL,
    data_type VARCHAR(255),
    value DOUBLE PRECISION,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT fk_telemetries_device FOREIGN KEY (device_id)
        REFERENCES devices(id) ON DELETE CASCADE
);

CREATE INDEX idx_telemetries_device_id ON telemetries(device_id);
CREATE INDEX idx_telemetries_recorded_at ON telemetries(recorded_at);
CREATE INDEX idx_telemetries_deleted_at ON telemetries(deleted_at);
