package queries

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// WelcomeConfig contains all configurable join/leave presentation settings.
type WelcomeConfig struct {
	GuildID          string
	JoinEnabled      bool
	JoinChannelID    *string
	JoinMessage      *string
	JoinImageEnabled bool
	JoinImagePath    *string // Legacy local-upload field retained for compatibility.
	JoinImageURL     *string
	JoinImageZoom    int
	JoinImagePosX    int
	JoinImagePosY    int
	JoinDMEnabled    bool
	JoinDMMessage    *string
	LeaveEnabled     bool
	LeaveChannelID   *string
	LeaveMessage     *string
	UseEmbed         bool
}

func (s *Store) GetWelcomeConfig(ctx context.Context, guildID string) (*WelcomeConfig, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT guild_id, join_enabled, join_channel_id, join_message,
		       join_image_enabled, join_image_path, join_image_url,
		       join_image_zoom, join_image_pos_x, join_image_pos_y,
		       join_dm_enabled, join_dm_message,
		       leave_enabled, leave_channel_id, leave_message, use_embed
		FROM welcome_config WHERE guild_id = $1`, guildID)

	var c WelcomeConfig
	err := row.Scan(&c.GuildID, &c.JoinEnabled, &c.JoinChannelID, &c.JoinMessage,
		&c.JoinImageEnabled, &c.JoinImagePath, &c.JoinImageURL,
		&c.JoinImageZoom, &c.JoinImagePosX, &c.JoinImagePosY,
		&c.JoinDMEnabled, &c.JoinDMMessage,
		&c.LeaveEnabled, &c.LeaveChannelID, &c.LeaveMessage, &c.UseEmbed)
	if errors.Is(err, pgx.ErrNoRows) {
		return &WelcomeConfig{GuildID: guildID, UseEmbed: true, JoinImageEnabled: true, JoinImageZoom: 100, JoinImagePosX: 50, JoinImagePosY: 50}, nil
	}
	if err != nil {
		return nil, err
	}
	if c.JoinImageZoom < 50 || c.JoinImageZoom > 200 { c.JoinImageZoom = 100 }
	if c.JoinImagePosX < 0 || c.JoinImagePosX > 100 { c.JoinImagePosX = 50 }
	if c.JoinImagePosY < 0 || c.JoinImagePosY > 100 { c.JoinImagePosY = 50 }
	return &c, nil
}

func (s *Store) UpsertWelcomeConfig(ctx context.Context, c *WelcomeConfig) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO welcome_config (
			guild_id, join_enabled, join_channel_id, join_message,
			join_image_enabled, join_image_path, join_image_url,
			join_image_zoom, join_image_pos_x, join_image_pos_y,
			join_dm_enabled, join_dm_message,
			leave_enabled, leave_channel_id, leave_message, use_embed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (guild_id) DO UPDATE SET
			join_enabled = EXCLUDED.join_enabled,
			join_channel_id = EXCLUDED.join_channel_id,
			join_message = EXCLUDED.join_message,
			join_image_enabled = EXCLUDED.join_image_enabled,
			join_image_path = EXCLUDED.join_image_path,
			join_image_url = EXCLUDED.join_image_url,
			join_image_zoom = EXCLUDED.join_image_zoom,
			join_image_pos_x = EXCLUDED.join_image_pos_x,
			join_image_pos_y = EXCLUDED.join_image_pos_y,
			join_dm_enabled = EXCLUDED.join_dm_enabled,
			join_dm_message = EXCLUDED.join_dm_message,
			leave_enabled = EXCLUDED.leave_enabled,
			leave_channel_id = EXCLUDED.leave_channel_id,
			leave_message = EXCLUDED.leave_message,
			use_embed = EXCLUDED.use_embed`,
		c.GuildID, c.JoinEnabled, c.JoinChannelID, c.JoinMessage,
		c.JoinImageEnabled, c.JoinImagePath, c.JoinImageURL,
		c.JoinImageZoom, c.JoinImagePosX, c.JoinImagePosY,
		c.JoinDMEnabled, c.JoinDMMessage,
		c.LeaveEnabled, c.LeaveChannelID, c.LeaveMessage, c.UseEmbed)
	return err
}
