CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50),
    mac_address VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    user_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT fk_devices_user FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_devices_mac_address ON devices(mac_address);
CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE INDEX idx_devices_deleted_at ON devices(deleted_at);
