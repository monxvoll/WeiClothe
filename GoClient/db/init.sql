CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    sub_keycloak VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    nickname VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    date_birth DATE NOT NULL,
    gender VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- TODO: Update diagram with the new table

CREATE TABLE clothes (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(sub_keycloak),
    image_url TEXT NOT NULL UNIQUE,
    image_width INT,
    image_height INT,
    garment_type VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    classification_id VARCHAR(255),
    category VARCHAR(255),
    subcategory VARCHAR(255),
    color VARCHAR(255),
    pattern VARCHAR(255),
    material VARCHAR(255),
    season VARCHAR(255),
    occasion VARCHAR(255),
    confidence NUMERIC(5,4),
    source VARCHAR(30) NOT NULL DEFAULT 'ai',
    model_name VARCHAR(255),
    model_version VARCHAR(100),
    status VARCHAR(30) NOT NULL DEFAULT 'queued',
    processing_error TEXT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT clothes_status_check CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
    CONSTRAINT clothes_source_check CHECK (source IN ('ai', 'manual', 'ai+manual')),
    CONSTRAINT clothes_confidence_check CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1))
);

CREATE TABLE styles (
    id SERIAL PRIMARY KEY,
    clothe_id INT NOT NULL REFERENCES clothes(id),
    style_id VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- YOLO can return multiple detections for one image.
-- Keep the canonical garment in clothes and detailed detections here.
CREATE TABLE clothe_detections (
    id SERIAL PRIMARY KEY,
    clothe_id INT NOT NULL REFERENCES clothes(id) ON DELETE CASCADE,
    class_name VARCHAR(255) NOT NULL,
    confidence NUMERIC(5,4) NOT NULL,
    bbox_x NUMERIC(10,4) NOT NULL,
    bbox_y NUMERIC(10,4) NOT NULL,
    bbox_w NUMERIC(10,4) NOT NULL,
    bbox_h NUMERIC(10,4) NOT NULL,
    mask_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT clothe_detections_confidence_check CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT clothe_detections_bbox_x_check CHECK (bbox_x >= 0),
    CONSTRAINT clothe_detections_bbox_y_check CHECK (bbox_y >= 0),
    CONSTRAINT clothe_detections_bbox_w_check CHECK (bbox_w > 0),
    CONSTRAINT clothe_detections_bbox_h_check CHECK (bbox_h > 0)
);

CREATE INDEX idx_clothes_user_id ON clothes(user_id);
CREATE INDEX idx_clothes_status ON clothes(status);
CREATE INDEX idx_clothes_category ON clothes(category);
CREATE INDEX idx_clothes_processed_at ON clothes(processed_at);
CREATE INDEX idx_clothe_detections_clothe_id ON clothe_detections(clothe_id);

-- TODO: Update style name in the diagram