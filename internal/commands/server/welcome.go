package server

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/0xSalik/specter/internal/core"
	"github.com/0xSalik/specter/internal/db/queries"
)

func registerWelcome(r *core.Router) {
	r.Register(core.Command{
		Group: group, RequiredPerm: discordgo.PermissionManageServer, Handler: handleWelcome,
		Def: &discordgo.ApplicationCommand{
			Name: "welcome", Description: "Configure welcome and goodbye messages",
			Options: []*discordgo.ApplicationCommandOption{
				sub("show", "Show the current welcome/goodbye configuration"),
				sub("join", "Configure the channel welcome message",
					optBool("enabled", "Turn the join message on or off", true),
					optChannel("channel", "Channel to post welcomes in", false),
					optString("message", "Message text. Placeholders: {user} {username} {server} {membercount}", false),
					optBool("image", "Send a generated welcome card", false)),
				sub("leave", "Configure the goodbye message",
					optBool("enabled", "Turn the leave message on or off", true),
					optChannel("channel", "Channel to post goodbyes in", false),
					optString("message", "Message text. Placeholders: {user} {username} {server} {membercount}", false)),
				sub("dm", "Configure a welcome DM to new members",
					optBool("enabled", "Turn the welcome DM on or off", true),
					optString("message", "DM text. Same placeholders as join messages.", false)),
				sub("embed", "Send welcome/goodbye messages as embeds or plain text",
					optBool("enabled", "On = embeds, Off = plain text", true)),
				sub("test", "Preview the join message in this channel"),
			},
		},
	})
}

func handleWelcome(c *core.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := c.Store.GetWelcomeConfig(ctx, c.GuildID)
	if err != nil {
		_ = c.Errorf("Failed to load the welcome configuration.", err)
		return
	}

	switch c.SubCommand {
	case "show":
		_ = c.Reply(welcomeEmbed(c, cfg))
	case "join":
		cfg.JoinEnabled = c.BoolOpt("enabled", cfg.JoinEnabled)
		if ch := c.ChannelOpt("channel"); ch != nil {
			cfg.JoinChannelID = &ch.ID
		}
		if c.HasOpt("message") {
			m := c.StringOpt("message", "")
			cfg.JoinMessage = &m
		}
		if c.HasOpt("image") {
			cfg.JoinImageEnabled = c.BoolOpt("image", cfg.JoinImageEnabled)
		}
		saveWelcome(c, ctx, cfg)
	case "leave":
		cfg.LeaveEnabled = c.BoolOpt("enabled", cfg.LeaveEnabled)
		if ch := c.ChannelOpt("channel"); ch != nil {
			cfg.LeaveChannelID = &ch.ID
		}
		if c.HasOpt("message") {
			m := c.StringOpt("message", "")
			cfg.LeaveMessage = &m
		}
		saveWelcome(c, ctx, cfg)
	case "dm":
		cfg.JoinDMEnabled = c.BoolOpt("enabled", cfg.JoinDMEnabled)
		if c.HasOpt("message") {
			m := c.StringOpt("message", "")
			cfg.JoinDMMessage = &m
		}
		saveWelcome(c, ctx, cfg)
	case "embed":
		cfg.UseEmbed = c.BoolOpt("enabled", cfg.UseEmbed)
		saveWelcome(c, ctx, cfg)
	case "test":
		previewWelcome(c, cfg)
	default:
		_ = c.Errorf("Unknown welcome subcommand.", nil)
	}
}

func saveWelcome(c *core.Context, ctx context.Context, cfg *queries.WelcomeConfig) {
	if err := c.Store.UpsertWelcomeConfig(ctx, cfg); err != nil {
		_ = c.Errorf("Failed to save the welcome configuration.", err)
		return
	}
	_ = c.Reply(welcomeEmbed(c, cfg))
}

func previewWelcome(c *core.Context, cfg *queries.WelcomeConfig) {
	text := "Welcome to {server}, {user}! You are member #{membercount}."
	if cfg.JoinMessage != nil && *cfg.JoinMessage != "" {
		text = *cfg.JoinMessage
	}
	var user *discordgo.User
	if c.Interaction.Member != nil {
		user = c.Interaction.Member.User
	}
	if user == nil {
		user = c.Interaction.User
	}
	rendered := renderPreview(c, text, user)
	_ = c.Reply(c.Embed().Title("Welcome Preview").Description(rendered).Build())
}

func renderPreview(c *core.Context, tmpl string, u *discordgo.User) string {
	server := "the server"
	count := 0
	if g, err := c.Session.State.Guild(c.GuildID); err == nil && g != nil {
		server = g.Name
		count = g.MemberCount
	}
	id, name := "0", "member"
	if u != nil {
		id, name = u.ID, u.Username
	}
	return strings.NewReplacer(
		"{user}", "<@"+id+">", "{mention}", "<@"+id+">",
		"{username}", name, "{server}", server, "{guild}", server,
		"{membercount}", strconv.Itoa(count), "{memberCount}", strconv.Itoa(count), "{id}", id,
	).Replace(tmpl)
}

func welcomeEmbed(c *core.Context, cfg *queries.WelcomeConfig) *discordgo.MessageEmbed {
	b := c.Embed().Title("Welcome Configuration").
		Field("Join Message", onOff(cfg.JoinEnabled), true).
		Field("Join Channel", channelMention(cfg.JoinChannelID), true).
		Field("Welcome Card", onOff(cfg.JoinImageEnabled), true).
		Field("Welcome DM", onOff(cfg.JoinDMEnabled), true).
		Field("Leave Message", onOff(cfg.LeaveEnabled), true).
		Field("Leave Channel", channelMention(cfg.LeaveChannelID), true).
		Field("Format", embedFormat(cfg.UseEmbed), true)
	if cfg.JoinMessage != nil && *cfg.JoinMessage != "" {
		b.Field("Join Text", *cfg.JoinMessage, false)
	}
	if cfg.LeaveMessage != nil && *cfg.LeaveMessage != "" {
		b.Field("Leave Text", *cfg.LeaveMessage, false)
	}
	return b.Timestamp().Build()
}

func embedFormat(useEmbed bool) string {
	if useEmbed {
		return "Embed"
	}
	return "Plain text"
}
