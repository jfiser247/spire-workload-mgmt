-- SPIFFE/SPIRE Workload Entry Management System Database Schema

CREATE DATABASE IF NOT EXISTS spire_mgmt;
USE spire_mgmt;

-- Sites table
CREATE TABLE sites (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    region VARCHAR(50) NOT NULL,
    spire_server_address VARCHAR(255) NOT NULL,
    last_sync_at TIMESTAMP NULL,
    status ENUM('active', 'inactive') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Workload entries table
CREATE TABLE workload_entries (
    id VARCHAR(36) PRIMARY KEY,
    spiffe_id VARCHAR(512) NOT NULL UNIQUE,
    parent_id VARCHAR(512) NOT NULL,
    selectors JSON NOT NULL,
    ttl INT DEFAULT 3600,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    INDEX idx_spiffe_id (spiffe_id),
    INDEX idx_parent_id (parent_id)
) ENGINE=InnoDB;

-- Site workload entry assignments
CREATE TABLE site_workload_entries (
    site_id VARCHAR(36),
    workload_entry_id VARCHAR(36),
    sync_status ENUM('pending', 'synced', 'failed') DEFAULT 'pending',
    last_sync_at TIMESTAMP NULL,
    sync_error TEXT,
    PRIMARY KEY (site_id, workload_entry_id),
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
    FOREIGN KEY (workload_entry_id) REFERENCES workload_entries(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Audit log
CREATE TABLE audit_log (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    actor VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    details JSON,
    INDEX idx_timestamp (timestamp),
    INDEX idx_actor (actor)
) ENGINE=InnoDB;

-- Insert sample sites
INSERT INTO sites (id, name, region, spire_server_address) VALUES
    ('site-1', 'US East', 'us-east-1', 'spire-server-1:8081'),
    ('site-2', 'EU West', 'eu-west-1', 'spire-server-2:8081');
