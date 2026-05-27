-- User-defined style preferences for recommendation scoring.

CREATE TABLE IF NOT EXISTS user_style_preferences (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(sub_keycloak),
    preferred_colors TEXT[] NOT NULL DEFAULT '{}',
    preferred_occasions TEXT[] NOT NULL DEFAULT '{}',
    preferred_seasons TEXT[] NOT NULL DEFAULT '{}',
    avoid_colors TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id)
);

CREATE INDEX IF NOT EXISTS idx_user_style_preferences_user_id ON user_style_preferences(user_id);
