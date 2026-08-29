ALTER TABLE welcome_config
    ADD COLUMN IF NOT EXISTS join_image_url TEXT,
    ADD COLUMN IF NOT EXISTS join_image_zoom INTEGER NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS join_image_pos_x INTEGER NOT NULL DEFAULT 50,
    ADD COLUMN IF NOT EXISTS join_image_pos_y INTEGER NOT NULL DEFAULT 50;

UPDATE welcome_config
SET join_image_zoom = LEAST(200, GREATEST(50, COALESCE(join_image_zoom, 100))),
    join_image_pos_x = LEAST(100, GREATEST(0, COALESCE(join_image_pos_x, 50))),
    join_image_pos_y = LEAST(100, GREATEST(0, COALESCE(join_image_pos_y, 50)));
