ALTER TABLE welcome_config
    DROP COLUMN IF EXISTS join_image_pos_y,
    DROP COLUMN IF EXISTS join_image_pos_x,
    DROP COLUMN IF EXISTS join_image_zoom,
    DROP COLUMN IF EXISTS join_image_url;
