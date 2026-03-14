package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rusq/slack"
	"github.com/rusq/slackdump/v4"
	"github.com/rusq/slackdump/v4/auth"
	sdtypes "github.com/rusq/slackdump/v4/types"
)

type SlackCollector struct {
	session      *slackdump.Session
	workspaceURL string
	channels     []slack.Channel
	users        []slack.User
	targetUser   slack.User
	mentionToken string
	channelMap   map[string]slack.Channel
	threadCache  map[string][]sdtypes.Message
}

type SlackConversationRecord struct {
	Month        string               `json:"month"`
	ChannelID    string               `json:"channel_id"`
	ChannelName  string               `json:"channel_name"`
	ChannelType  string               `json:"channel_type"`
	ThreadTS     string               `json:"thread_ts,omitempty"`
	Matches      []SlackMatchRecord   `json:"matches"`
	Messages     []SlackMessageRecord `json:"messages"`
	SourceWindow TimeWindow           `json:"source_window"`
}

type SlackMatchRecord struct {
	Timestamp string `json:"timestamp"`
	MatchedAs string `json:"matched_as"`
}

type SlackMessageRecord struct {
	Timestamp string `json:"timestamp"`
	UserID    string `json:"user_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Text      string `json:"text,omitempty"`
	Subtype   string `json:"subtype,omitempty"`
	ThreadTS  string `json:"thread_ts,omitempty"`
}

type TimeWindow struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func newSlackCollector(ctx context.Context, cfg Config) (*SlackCollector, error) {
	provider, err := auth.NewValueAuth(cfg.SlackToken, cfg.SlackCookie)
	if err != nil {
		return nil, fmt.Errorf("slack auth: %w", err)
	}
	session, err := slackdump.New(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("create slack session: %w", err)
	}
	users, err := session.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch slack users: %w", err)
	}
	targetUser, err := resolveSlackUser(users, cfg.SlackUserHandle)
	if err != nil {
		return nil, err
	}
	channels, err := session.GetChannelsEx(ctx, slackdump.GetChannelsParameters{ChannelTypes: slackdump.AllChanTypes})
	if err != nil {
		return nil, fmt.Errorf("fetch slack channels: %w", err)
	}
	channelMap := make(map[string]slack.Channel, len(channels))
	for _, channel := range channels {
		channelMap[channel.ID] = channel
	}

	workspaceURL := strings.TrimSuffix(session.Info().URL, "/")
	if workspaceURL == "" {
		workspaceURL = "https://slack.com"
	}

	return &SlackCollector{
		session:      session,
		workspaceURL: workspaceURL,
		channels:     channels,
		users:        users,
		targetUser:   targetUser,
		mentionToken: fmt.Sprintf("<@%s>", targetUser.ID),
		channelMap:   channelMap,
		threadCache:  map[string][]sdtypes.Message{},
	}, nil
}

func (c *SlackCollector) SmokeTest(ctx context.Context) (SlackSmokeResult, error) {
	result := SlackSmokeResult{
		WorkspaceURL: c.workspaceURL,
		UserID:       c.targetUser.ID,
		ChannelCount: len(c.channels),
		UserCount:    len(c.users),
	}
	if len(c.channels) == 0 {
		return result, fmt.Errorf("no Slack conversations visible to authenticated user")
	}
	if _, err := c.session.Client(); err != nil {
		return result, fmt.Errorf("slack client unavailable: %w", err)
	}
	return result, nil
}

func (c *SlackCollector) CollectMonth(ctx context.Context, cfg Config, month MonthRange) ([]SlackConversationRecord, error) {
	monthStart, monthEnd := clipMonth(month, cfg)
	window := TimeWindow{From: monthStart.Format(time.RFC3339), To: monthEnd.Format(time.RFC3339)}
	var records []SlackConversationRecord

	for _, channel := range c.channels {
		conversation, err := c.session.Dump(ctx, channel.ID, monthStart, monthEnd)
		if err != nil {
			return nil, fmt.Errorf("dump Slack channel %s: %w", channel.ID, err)
		}

		threadMatches := map[string][]SlackMatchRecord{}
		for _, message := range conversation.Messages {
			matchType, matched := c.matchMessage(message)
			if !matched {
				continue
			}
			matchRecord := SlackMatchRecord{Timestamp: message.Timestamp, MatchedAs: matchType}

			threadTS := normalizedThreadTS(message)
			if threadTS != "" {
				key := channel.ID + ":" + threadTS
				threadMatches[key] = append(threadMatches[key], matchRecord)
				continue
			}

			records = append(records, SlackConversationRecord{
				Month:        month.Label,
				ChannelID:    channel.ID,
				ChannelName:  channelName(channel),
				ChannelType:  channelType(channel),
				Matches:      []SlackMatchRecord{matchRecord},
				Messages:     c.convertMessages([]sdtypes.Message{message}),
				SourceWindow: window,
			})
		}

		threadKeys := make([]string, 0, len(threadMatches))
		for key := range threadMatches {
			threadKeys = append(threadKeys, key)
		}
		sort.Strings(threadKeys)
		for _, key := range threadKeys {
			threadTS := strings.TrimPrefix(key, channel.ID+":")
			threadMessages, err := c.getThreadMessages(ctx, channel.ID, threadTS)
			if err != nil {
				return nil, fmt.Errorf("dump Slack thread %s: %w", key, err)
			}
			records = append(records, SlackConversationRecord{
				Month:        month.Label,
				ChannelID:    channel.ID,
				ChannelName:  channelName(channel),
				ChannelType:  channelType(channel),
				ThreadTS:     threadTS,
				Matches:      uniqueMatches(threadMatches[key]),
				Messages:     c.convertMessages(threadMessages),
				SourceWindow: window,
			})
		}
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].ChannelID == records[j].ChannelID {
			return firstMatchTimestamp(records[i].Matches) < firstMatchTimestamp(records[j].Matches)
		}
		return records[i].ChannelID < records[j].ChannelID
	})

	return records, nil
}

func (c *SlackCollector) matchMessage(message sdtypes.Message) (string, bool) {
	if message.User == c.targetUser.ID {
		return "author", true
	}
	if strings.Contains(message.Text, c.mentionToken) {
		return "mention", true
	}
	return "", false
}

func (c *SlackCollector) getThreadMessages(ctx context.Context, channelID, threadTS string) ([]sdtypes.Message, error) {
	cacheKey := channelID + ":" + threadTS
	if cached, ok := c.threadCache[cacheKey]; ok {
		return cached, nil
	}
	conversation, err := c.session.Dump(ctx, cacheKey, time.Time{}, time.Time{})
	if err != nil {
		return nil, err
	}
	messages := append([]sdtypes.Message(nil), conversation.Messages...)
	c.threadCache[cacheKey] = messages
	return messages, nil
}

func (c *SlackCollector) convertMessages(messages []sdtypes.Message) []SlackMessageRecord {
	records := make([]SlackMessageRecord, 0, len(messages))
	for _, message := range messages {
		records = append(records, SlackMessageRecord{
			Timestamp: message.Timestamp,
			UserID:    message.User,
			Username:  c.userName(message.User),
			Text:      message.Text,
			Subtype:   message.SubType,
			ThreadTS:  message.ThreadTimestamp,
		})
	}
	return records
}

func (c *SlackCollector) userName(userID string) string {
	for _, user := range c.users {
		if user.ID != userID {
			continue
		}
		if user.Profile.DisplayName != "" {
			return user.Profile.DisplayName
		}
		if user.Profile.RealName != "" {
			return user.Profile.RealName
		}
		return user.Name
	}
	return ""
}

func resolveSlackUser(users []slack.User, handle string) (slack.User, error) {
	needle := strings.ToLower(normalizeHandle(handle))
	for _, user := range users {
		candidates := []string{user.Name, user.Profile.DisplayName, user.Profile.RealName, user.Profile.DisplayNameNormalized, user.Profile.RealNameNormalized}
		for _, candidate := range candidates {
			if strings.ToLower(normalizeHandle(candidate)) == needle {
				return user, nil
			}
		}
	}
	return slack.User{}, fmt.Errorf("could not find Slack user matching handle %q", handle)
}

func normalizedThreadTS(message sdtypes.Message) string {
	if message.ThreadTimestamp == "" {
		return ""
	}
	return message.ThreadTimestamp
}

func uniqueMatches(matches []SlackMatchRecord) []SlackMatchRecord {
	seen := map[string]bool{}
	result := make([]SlackMatchRecord, 0, len(matches))
	for _, match := range matches {
		key := match.Timestamp + "|" + match.MatchedAs
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, match)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Timestamp == result[j].Timestamp {
			return result[i].MatchedAs < result[j].MatchedAs
		}
		return result[i].Timestamp < result[j].Timestamp
	})
	return result
}

func firstMatchTimestamp(matches []SlackMatchRecord) string {
	if len(matches) == 0 {
		return ""
	}
	return matches[0].Timestamp
}

func channelType(channel slack.Channel) string {
	switch {
	case channel.IsIM:
		return "im"
	case channel.IsMpIM:
		return "mpim"
	case channel.IsPrivate:
		return "private_channel"
	case channel.IsChannel:
		return "public_channel"
	default:
		return "unknown"
	}
}

func channelName(channel slack.Channel) string {
	if channel.Name != "" {
		return channel.Name
	}
	if channel.User != "" {
		return channel.User
	}
	return channel.ID
}

type SlackSmokeResult struct {
	WorkspaceURL string
	UserID       string
	ChannelCount int
	UserCount    int
}
