package events

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"

	"github.com/0xSalik/specter/internal/discordutil"
	"github.com/0xSalik/specter/internal/embed"
	levelsvc "github.com/0xSalik/specter/internal/levels"
)

const (
	defaultJoinMessage  = "Welcome to {server}, {user}! You are member #{membercount}."
	defaultLeaveMessage = "{username} has left {server}. We now have {membercount} members."
)

func (h *Handlers) handleWelcomeJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if m.User == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := h.deps.Store.GetWelcomeConfig(ctx, m.GuildID)
	if err != nil {
		log.Error().Err(err).Str("guild", m.GuildID).Msg("welcome: load config")
		return
	}

	if cfg.JoinEnabled && cfg.JoinChannelID != nil && *cfg.JoinChannelID != "" {
		text := defaultJoinMessage
		if cfg.JoinMessage != nil && *cfg.JoinMessage != "" {
			text = *cfg.JoinMessage
		}
		text = renderWelcome(s, text, m.GuildID, m.User)

		var card []byte
		if cfg.JoinImageEnabled {
			serverName := "the server"
			memberCount := 0
			if g, err := s.State.Guild(m.GuildID); err == nil && g != nil {
				serverName = g.Name
				memberCount = g.MemberCount
			}
			card, err = levelsvc.RenderWelcomeCard(ctx, levelsvc.WelcomeCardData{
				Username:       m.User.Username,
				AvatarURL:      discordutil.AvatarURL(m.User),
				ServerName:     serverName,
				MemberCount:    memberCount,
				BackgroundPath: derefString(cfg.JoinImagePath),
			})
			if err != nil {
				log.Warn().Err(err).Str("guild", m.GuildID).Msg("welcome: render card")
			}
		}

		sendWelcome(s, m.GuildID, *cfg.JoinChannelID, text, cfg.UseEmbed, "Welcome", card)
	}

	if cfg.JoinDMEnabled && cfg.JoinDMMessage != nil && *cfg.JoinDMMessage != "" {
		if ch, err := s.UserChannelCreate(m.User.ID); err == nil {
			sendWelcome(s, m.GuildID, ch.ID, renderWelcome(s, *cfg.JoinDMMessage, m.GuildID, m.User), cfg.UseEmbed, "Welcome", nil)
		}
	}
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (h *Handlers) handleWelcomeLeave(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	if m.User == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := h.deps.Store.GetWelcomeConfig(ctx, m.GuildID)
	if err != nil {
		log.Error().Err(err).Str("guild", m.GuildID).Msg("goodbye: load config")
		return
	}
	if !cfg.LeaveEnabled || cfg.LeaveChannelID == nil || *cfg.LeaveChannelID == "" {
		return
	}
	text := defaultLeaveMessage
	if cfg.LeaveMessage != nil && *cfg.LeaveMessage != "" {
		text = *cfg.LeaveMessage
	}
	sendWelcome(s, m.GuildID, *cfg.LeaveChannelID, renderWelcome(s, text, m.GuildID, m.User), cfg.UseEmbed, "Goodbye", nil)
}

func sendWelcome(s *discordgo.Session, guildID, channelID, text string, useEmbed bool, title string, card []byte) {
	if strings.TrimSpace(text) == "" && len(card) == 0 {
		return
	}
	if len(card) > 0 {
		msg := &discordgo.MessageSend{Files: []*discordgo.File{{Name: "welcome.png", ContentType: "image/png", Reader: bytes.NewReader(card)}}}
		if useEmbed {
			msg.Embed = embed.New(s, guildID).Title(title).Description(text).Image("attachment://welcome.png").Build()
		} else {
			msg.Content = text
		}
		if _, err := s.ChannelMessageSendComplex(channelID, msg); err == nil {
			return
		} else {
			log.Warn().Err(err).Str("guild", guildID).Msg("welcome: send card")
		}
	}
	if useEmbed {
		e := embed.New(s, guildID).Title(title).Description(text).Build()
		_, _ = s.ChannelMessageSendEmbed(channelID, e)
		return
	}
	_, _ = s.ChannelMessageSend(channelID, text)
}

func renderWelcome(s *discordgo.Session, tmpl, guildID string, u *discordgo.User) string {
	server := "the server"
	count := 0
	if g, err := s.State.Guild(guildID); err == nil && g != nil {
		server = g.Name
		count = g.MemberCount
	}
	repl := strings.NewReplacer(
		"{user}", "<@"+u.ID+">",
		"{mention}", "<@"+u.ID+">",
		"{username}", u.Username,
		"{tag}", userTag(u),
		"{server}", server,
		"{guild}", server,
		"{membercount}", strconv.Itoa(count),
		"{memberCount}", strconv.Itoa(count),
		"{id}", u.ID,
	)
	return repl.Replace(tmpl)
}

func (h *Handlers) applyAutorole(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	if m.User == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cfg, err := h.deps.Store.GetAutoroleConfig(ctx, m.GuildID)
	if err != nil || !cfg.Enabled {
		return
	}
	roles := cfg.RoleIDs
	if m.User.Bot {
		roles = cfg.BotRoleIDs
	}
	for _, roleID := range roles {
		if roleID == "" {
			continue
		}
		if err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, roleID); err != nil {
			log.Warn().Err(err).Str("guild", m.GuildID).Str("role", roleID).Msg("autorole: add role")
		}
	}
}
