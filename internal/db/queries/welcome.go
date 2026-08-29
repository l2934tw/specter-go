package queries

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// WelcomeConfig mirrors a row of the welcome_config table.
type WelcomeConfig struct {
	GuildID          string
	JoinEnabled      bool
	JoinChannelID    *string
	JoinMessage      *string
	JoinImageEnabled bool
	JoinImagePath    *string
	JoinDMEnabled    bool
	JoinDMMessage    *string
	LeaveEnabled     bool
	LeaveChannelID   *string
	LeaveMessage     *string
	UseEmbed         bool
}

// GetWelcomeConfig fetches welcome configuration, returning defaults if absent.
func (s *Store) GetWelcomeConfig(ctx context.Context, guildID string) (*WelcomeConfig, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT guild_id, join_enabled, join_channel_id, join_message, join_image_enabled, join_image_path,
		       join_dm_enabled, join_dm_message,
		       leave_enabled, leave_channel_id, leave_message, use_embed
		FROM welcome_config WHERE guild_id = $1`, guildID)

	var c WelcomeConfig
	err := row.Scan(&c.GuildID, &c.JoinEnabled, &c.JoinChannelID, &c.JoinMessage, &c.JoinImageEnabled, &c.JoinImagePath,
		&c.JoinDMEnabled, &c.JoinDMMessage,
		&c.LeaveEnabled, &c.LeaveChannelID, &c.LeaveMessage, &c.UseEmbed)
	if errors.Is(err, pgx.ErrNoRows) {
		return &WelcomeConfig{GuildID: guildID, UseEmbed: true, JoinImageEnabled: true}, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpsertWelcomeConfig writes the full welcome configuration for a guild.
func (s *Store) UpsertWelcomeConfig(ctx context.Context, c *WelcomeConfig) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO welcome_config (guild_id, join_enabled, join_channel_id, join_message, join_image_enabled, join_image_path,
			join_dm_enabled, join_dm_message, leave_enabled, leave_channel_id, leave_message, use_embed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (guild_id) DO UPDATE SET
			join_enabled = EXCLUDED.join_enabled,
			join_channel_id = EXCLUDED.join_channel_id,
			join_message = EXCLUDED.join_message,
			join_image_enabled = EXCLUDED.join_image_enabled,
			join_image_path = EXCLUDED.join_image_path,
			join_dm_enabled = EXCLUDED.join_dm_enabled,
			join_dm_message = EXCLUDED.join_dm_message,
			leave_enabled = EXCLUDED.leave_enabled,
			leave_channel_id = EXCLUDED.leave_channel_id,
			leave_message = EXCLUDED.leave_message,
			use_embed = EXCLUDED.use_embed`,
		c.GuildID, c.JoinEnabled, c.JoinChannelID, c.JoinMessage, c.JoinImageEnabled, c.JoinImagePath,
		c.JoinDMEnabled, c.JoinDMMessage, c.LeaveEnabled, c.LeaveChannelID, c.LeaveMessage, c.UseEmbed)
	return err
}
