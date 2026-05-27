-- Indexes for recommendation / style filtering by garment metadata.

CREATE INDEX IF NOT EXISTS idx_clothes_color ON clothes(color);
CREATE INDEX IF NOT EXISTS idx_clothes_material ON clothes(material);
CREATE INDEX IF NOT EXISTS idx_clothes_occasion ON clothes(occasion);
CREATE INDEX IF NOT EXISTS idx_clothes_season ON clothes(season);
CREATE INDEX IF NOT EXISTS idx_clothes_pattern ON clothes(pattern);
