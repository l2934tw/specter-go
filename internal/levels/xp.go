// Package levels implements the XP and leveling system: progression math plus
// a message-driven engine that awards XP with cooldowns and exemptions.
package levels

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"

	"github.com/0xSalik/specter/internal/db"
	"github.com/0xSalik/specter/internal/db/queries"
	"github.com/0xSalik/specter/internal/discordutil"
	"github.com/0xSalik/specter/internal/embed"
)

func LevelForXP(xp int64) int {
	if xp <= 0 {
		return 0
	}
	return int(math.Floor(0.1 * math.Sqrt(float64(xp))))
}

func CalculateXPForLevel(level int) int64 {
	if level <= 0 {
		return 0
	}
	return int64(100 * level * level)
}

func AwardXP(r *rand.Rand, min, max int) int64 {
	if max < min {
		max = min
	}
	if max == min {
		return int64(min)
	}
	return int64(min + r.Intn(max-min+1))
}

func OnCooldown(last *time.Time, now time.Time, cooldownSecs int) bool {
	if last == nil {
		return false
	}
	return now.Sub(*last) < time.Duration(cooldownSecs)*time.Second
}

func IsExempt(userRoles []string, channelID string, noXPRoles, noXPChannels []string) bool {
	for _, c := range noXPChannels {
		if c == channelID {
			return true
		}
	}
	roleSet := make(map[string]struct{}, len(userRoles))
	for _, r := range userRoles {
		roleSet[r] = struct{}{}
	}
	for _, r := range noXPRoles {
		if _, ok := roleSet[r]; ok {
			return true
		}
	}
	return false
}

type Engine struct {
	store *queries.Store
	rng   *rand.Rand
}

func NewEngine(store *queries.Store) *Engine {
	return &Engine{store: store, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (e *Engine) HandleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot || m.GuildID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := e.store.GetLevelConfig(ctx, m.GuildID)
	if err != nil {
		log.Error().Err(err).Str("guild", m.GuildID).Msg("levels: load config")
		return
	}
	if !cfg.Enabled {
		return
	}

	var roles []string
	if m.Member != nil {
		roles = m.Member.Roles
	}
	if IsExempt(roles, m.ChannelID, cfg.NoXPRoles, cfg.NoXPChannels) {
		return
	}

	now := time.Now()
	if existing, err := e.store.GetLevel(ctx, m.GuildID, m.Author.ID); err == nil {
		if OnCooldown(existing.LastXPAt, now, cfg.XPCooldownSecs) {
			return
		}
	} else if !db.IsNotFound(err) {
		log.Error().Err(err).Msg("levels: get level")
		return
	}

	gained := AwardXP(e.rng, cfg.XPMin, cfg.XPMax)

	prev, err := e.store.GetLevel(ctx, m.GuildID, m.Author.ID)
	var prevXP int64
	var prevLevel int
	if err == nil {
		prevXP = prev.XP
		prevLevel = prev.Level
	}
	newXP := prevXP + gained
	newLevel := LevelForXP(newXP)

	entry, err := e.store.AddXP(ctx, m.GuildID, m.Author.ID, gained, newLevel, now)
	if err != nil {
		log.Error().Err(err).Msg("levels: add xp")
		return
	}

	if entry.Level > prevLevel {
		e.announce(ctx, s, m, cfg, entry.Level, entry.XP)
		e.applyLevelRewards(ctx, s, m.GuildID, m.Author.ID, roles, entry.Level, cfg.StackRewards)
	}
}

func (e *Engine) applyLevelRewards(ctx context.Context, s *discordgo.Session, guildID, userID string, currentRoles []string, level int, stack bool) {
	rewards, err := e.store.ListLevelRewards(ctx, guildID)
	if err != nil || len(rewards) == 0 {
		return
	}

	held := make(map[string]struct{}, len(currentRoles))
	for _, r := range currentRoles {
		held[r] = struct{}{}
	}

	var earned []queries.LevelReward
	for _, rw := range rewards {
		if rw.Level <= level {
			earned = append(earned, rw)
		}
	}
	if len(earned) == 0 {
		return
	}

	if stack {
		for _, rw := range earned {
			if _, ok := held[rw.RoleID]; !ok {
				if err := s.GuildMemberRoleAdd(guildID, userID, rw.RoleID); err != nil {
					log.Warn().Err(err).Str("guild", guildID).Str("role", rw.RoleID).Msg("levels: add reward role")
				}
			}
		}
		return
	}

	highest := earned[len(earned)-1].RoleID
	if _, ok := held[highest]; !ok {
		if err := s.GuildMemberRoleAdd(guildID, userID, highest); err != nil {
			log.Warn().Err(err).Str("guild", guildID).Str("role", highest).Msg("levels: add reward role")
		}
	}
	for _, rw := range rewards {
		if rw.RoleID == highest {
			continue
		}
		if _, ok := held[rw.RoleID]; ok {
			if err := s.GuildMemberRoleRemove(guildID, userID, rw.RoleID); err != nil {
				log.Warn().Err(err).Str("guild", guildID).Str("role", rw.RoleID).Msg("levels: remove old reward role")
			}
		}
	}
}

func (e *Engine) announce(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, cfg *queries.LevelConfig, level int, xp int64) {
	channelID := m.ChannelID
	if cfg.AnnounceChannelID != nil && *cfg.AnnounceChannelID != "" {
		channelID = *cfg.AnnounceChannelID
	}

	msg := "Congratulations <@" + m.Author.ID + ">, you reached level " + itoa(level) + "."
	if cfg.AnnounceMsg != nil && *cfg.AnnounceMsg != "" {
		msg = renderAnnounce(*cfg.AnnounceMsg, m.Author.ID, level)
	}

	// Prefer the image announcement, while preserving a normal embed fallback.
	rank, rankErr := e.store.GetRank(ctx, m.GuildID, m.Author.ID)
	card, cardErr := RenderLevelUpCard(ctx, LevelUpCardData{
		Username: m.Author.Username,
		AvatarURL: discordutil.AvatarURL(m.Author),
		Level: level,
		Rank: rank,
		XP: xp,
	})
	if cardErr == nil && rankErr == nil {
		em := embed.New(s, m.GuildID).Title("Level Up").Description(msg).Image("attachment://levelup.png").Build()
		_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embed: em,
			Files: []*discordgo.File{{Name: "levelup.png", ContentType: "image/png", Reader: bytesReader(card)}},
		})
		if err == nil {
			return
		}
		log.Warn().Err(err).Str("guild", m.GuildID).Msg("levels: send card")
	}

	em := embed.New(s, m.GuildID).Title("Level Up").Description(msg).Build()
	if _, err := s.ChannelMessageSendEmbed(channelID, em); err != nil {
		log.Warn().Err(err).Str("guild", m.GuildID).Msg("levels: announce")
	}
}
