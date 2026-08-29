-- Image-based welcome and level-up announcements.
ALTER TABLE welcome_config
    ADD COLUMN IF NOT EXISTS join_image_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- Level-up announcements use the existing announce_channel_id and always render
-- an image card when an announcement is sent.
