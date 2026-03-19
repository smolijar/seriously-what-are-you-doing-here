package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
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
	client       *slack.Client
	workspaceURL string
	users        []slack.User
	targetUser   slack.User
	searchHandle string
	searchUserID string
	mentionToken string
	threadCache  map[string][]sdtypes.Message
}

type slackSearchHit struct {
	ChannelID   string
	ChannelName string
	Timestamp   string
	Text        string
	UserID      string
	Username    string
	Permalink   string
	Matches     []SlackMatchRecord
}

type resolvedSlackHit struct {
	Hit      slackSearchHit
	Message  sdtypes.Message
	ThreadTS string
}

type slackQuery struct {
	Name      string
	Query     string
	MatchType string
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
	session, err := slackdump.New(ctx, provider, slackdump.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	if err != nil {
		return nil, fmt.Errorf("create slack session: %w", err)
	}
	client, err := session.Client()
	if err != nil {
		return nil, fmt.Errorf("slack client: %w", err)
	}
	users, err := session.GetUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch slack users: %w", err)
	}
	targetUser, err := resolveSlackUser(users, cfg.SlackUserHandle, cfg.SlackUserID)
	if err != nil {
		return nil, err
	}

	workspaceURL := strings.TrimSuffix(session.Info().URL, "/")
	if workspaceURL == "" {
		workspaceURL = "https://slack.com"
	}

	return &SlackCollector{
		session:      session,
		client:       client,
		workspaceURL: workspaceURL,
		users:        users,
		targetUser:   targetUser,
		searchHandle: cfg.SlackUserHandle,
		searchUserID: targetUser.ID,
		mentionToken: fmt.Sprintf("<@%s>", targetUser.ID),
		threadCache:  map[string][]sdtypes.Message{},
	}, nil
}

func (c *SlackCollector) SmokeTest(ctx context.Context) (SlackSmokeResult, error) {
	channels, err := c.session.GetChannelsEx(ctx, slackdump.GetChannelsParameters{ChannelTypes: slackdump.AllChanTypes})
	if err != nil {
		return SlackSmokeResult{}, fmt.Errorf("fetch slack channels: %w", err)
	}
	result := SlackSmokeResult{
		WorkspaceURL: c.workspaceURL,
		UserID:       c.targetUser.ID,
		ChannelCount: len(channels),
		UserCount:    len(c.users),
	}
	if len(channels) == 0 {
		return result, fmt.Errorf("no Slack conversations visible to authenticated user")
	}
	if _, err := c.session.Client(); err != nil {
		return result, fmt.Errorf("slack client unavailable: %w", err)
	}
	return result, nil
}

func (c *SlackCollector) StreamMonth(ctx context.Context, cfg Config, month MonthRange, progress *ProgressReporter, emit func(SlackConversationRecord) error) (int, error) {
	monthStart, monthEnd := clipMonth(month, cfg)
	window := TimeWindow{From: monthStart.Format(time.RFC3339), To: monthEnd.Format(time.RFC3339)}
	queries := c.buildQueries(monthStart, monthEnd)
	progress.AddPlannedWork(len(queries))

	hits, err := c.searchHits(ctx, queries, progress)
	if err != nil {
		return 0, err
	}
	if len(hits) == 0 {
		return 0, nil
	}

	progress.AddPlannedWork(len(hits))
	resolved, threadCount, err := c.resolveHits(ctx, hits, progress)
	if err != nil {
		return 0, err
	}
	progress.AddPlannedWork(threadCount)

	count := 0
	threadGroups := map[string][]resolvedSlackHit{}
	threadMeta := map[string]slackSearchHit{}
	for _, hit := range resolved {
		if hit.ThreadTS == "" {
			record := SlackConversationRecord{
				Month:        month.Label,
				ChannelID:    hit.Hit.ChannelID,
				ChannelName:  hit.Hit.ChannelName,
				ChannelType:  "unknown",
				Matches:      uniqueMatches(hit.Hit.Matches),
				Messages:     c.convertMessages([]sdtypes.Message{hit.Message}),
				SourceWindow: window,
			}
			progress.SlackRecord(record)
			if err := emit(record); err != nil {
				return count, err
			}
			count++
			continue
		}
		key := hit.Hit.ChannelID + ":" + hit.ThreadTS
		threadGroups[key] = append(threadGroups[key], hit)
		threadMeta[key] = hit.Hit
	}

	threadKeys := make([]string, 0, len(threadGroups))
	for key := range threadGroups {
		threadKeys = append(threadKeys, key)
	}
	sort.Strings(threadKeys)
	for _, key := range threadKeys {
		meta := threadMeta[key]
		threadTS := strings.TrimPrefix(key, meta.ChannelID+":")
		matches := collectThreadMatches(threadGroups[key])
		progress.StartThread(meta.ChannelName, threadTS, len(matches))
		threadMessages, err := c.getThreadMessages(ctx, meta.ChannelID, threadTS)
		if err != nil {
			return count, fmt.Errorf("dump Slack thread %s: %w", key, err)
		}
		record := SlackConversationRecord{
			Month:        month.Label,
			ChannelID:    meta.ChannelID,
			ChannelName:  meta.ChannelName,
			ChannelType:  "unknown",
			ThreadTS:     threadTS,
			Matches:      matches,
			Messages:     c.convertMessages(threadMessages),
			SourceWindow: window,
		}
		progress.SlackRecord(record)
		if err := emit(record); err != nil {
			return count, err
		}
		progress.ThreadDone()
		count++
	}

	return count, nil
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

func (c *SlackCollector) buildQueries(monthStart, monthEnd time.Time) []slackQuery {
	dateAfter := monthStart.Format("2006-01-02")
	dateBefore := monthEnd.Format("2006-01-02")
	queries := []slackQuery{
		{Name: "authored-id", Query: fmt.Sprintf("from:<@%s> after:%s before:%s", c.searchUserID, dateAfter, dateBefore), MatchType: "author"},
		{Name: "mentions-id", Query: fmt.Sprintf("<@%s> after:%s before:%s", c.searchUserID, dateAfter, dateBefore), MatchType: "mention"},
	}
	if c.searchHandle != "" {
		queries = append(queries, slackQuery{Name: "authored-handle", Query: fmt.Sprintf("from:@%s after:%s before:%s", c.searchHandle, dateAfter, dateBefore), MatchType: "author"})
	}
	return queries
}

func (c *SlackCollector) searchHits(ctx context.Context, queries []slackQuery, progress *ProgressReporter) ([]slackSearchHit, error) {
	all := map[string]*slackSearchHit{}
	for _, query := range queries {
		progress.SearchQueryStart(query.Name, query.Query)
		results, err := c.searchMessages(ctx, query.Query)
		if err != nil {
			return nil, fmt.Errorf("slack search %s: %w", query.Name, err)
		}
		matches := 0
		for _, result := range results {
			if !c.acceptSearchResult(result, query.MatchType) {
				continue
			}
			matches++
			key := result.Channel.ID + ":" + result.Timestamp
			hit, ok := all[key]
			if !ok {
				hit = &slackSearchHit{
					ChannelID:   result.Channel.ID,
					ChannelName: fallbackSearchChannelName(result.Channel),
					Timestamp:   result.Timestamp,
					Text:        result.Text,
					UserID:      result.User,
					Username:    result.Username,
					Permalink:   result.Permalink,
				}
				all[key] = hit
			}
			hit.Matches = append(hit.Matches, SlackMatchRecord{Timestamp: result.Timestamp, MatchedAs: query.MatchType})
		}
		progress.SearchQueryDone(query.Name, matches)
	}
	hits := make([]slackSearchHit, 0, len(all))
	for _, hit := range all {
		hit.Matches = uniqueMatches(hit.Matches)
		hits = append(hits, *hit)
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Timestamp == hits[j].Timestamp {
			return hits[i].ChannelID < hits[j].ChannelID
		}
		return hits[i].Timestamp < hits[j].Timestamp
	})
	logf("[%s] slack search yielded %d unique hits", progress.monthLabel, len(hits))
	return hits, nil
}

func (c *SlackCollector) resolveHits(ctx context.Context, hits []slackSearchHit, progress *ProgressReporter) ([]resolvedSlackHit, int, error) {
	resolved := make([]resolvedSlackHit, 0, len(hits))
	threads := map[string]bool{}
	for i, hit := range hits {
		progress.ResolveHit(i+1, len(hits), hit.ChannelName, hit.Timestamp, hit.Text)
		message, threadTS := c.resolveSearchHit(ctx, hit)
		if threadTS != "" {
			threads[hit.ChannelID+":"+threadTS] = true
		}
		resolved = append(resolved, resolvedSlackHit{Hit: hit, Message: message, ThreadTS: threadTS})
		progress.ResolveHitDone()
	}
	return resolved, len(threads), nil
}

func (c *SlackCollector) resolveSearchHit(ctx context.Context, hit slackSearchHit) (sdtypes.Message, string) {
	message := sdtypes.Message{Message: slack.Message{
		Msg: slack.Msg{
			User:      hit.UserID,
			Username:  hit.Username,
			Text:      hit.Text,
			Timestamp: hit.Timestamp,
			Permalink: hit.Permalink,
		},
	}}

	threadTS := threadTSFromPermalink(hit.Permalink)
	if threadTS != "" {
		message.ThreadTimestamp = threadTS
		return message, threadTS
	}

	if hit.Permalink != "" {
		conversation, err := c.session.Dump(ctx, hit.Permalink, time.Time{}, time.Time{})
		if err == nil {
			for _, message := range conversation.Messages {
				if message.Timestamp == hit.Timestamp {
					return message, normalizedThreadTS(message)
				}
			}
			if len(conversation.Messages) > 0 {
				message := conversation.Messages[0]
				return message, normalizedThreadTS(message)
			}
		}
	}
	return message, ""
}

func (c *SlackCollector) fetchExactMessage(ctx context.Context, channelID, timestamp string) (sdtypes.Message, error) {
	ts, err := parseSlackTimestamp(timestamp)
	if err != nil {
		return sdtypes.Message{}, err
	}
	conversation, err := c.session.Dump(ctx, channelID, ts, ts)
	if err != nil {
		return sdtypes.Message{}, err
	}
	for _, message := range conversation.Messages {
		if message.Timestamp == timestamp {
			return message, nil
		}
	}
	return sdtypes.Message{}, fmt.Errorf("message %s not found in %s", timestamp, channelID)
}

func (c *SlackCollector) searchMessages(ctx context.Context, query string) ([]slack.SearchMessage, error) {
	params := slack.SearchParameters{Sort: "timestamp", SortDirection: "asc", Count: 100, Page: 1}
	all := make([]slack.SearchMessage, 0)
	for {
		results, err := c.client.SearchMessagesContext(ctx, query, params)
		if err != nil {
			if rateLimited, ok := err.(*slack.RateLimitedError); ok {
				logf("slack search rate limited retry=%s query=%q", rateLimited.RetryAfter, query)
				select {
				case <-ctx.Done():
					return nil, context.Cause(ctx)
				case <-time.After(rateLimited.RetryAfter):
				}
				continue
			}
			return nil, err
		}
		all = append(all, results.Matches...)
		if results.Pagination.Last == 0 || params.Page >= results.Pagination.Last {
			break
		}
		params.Page++
	}
	return all, nil
}

func (c *SlackCollector) acceptSearchResult(result slack.SearchMessage, matchType string) bool {
	switch matchType {
	case "author":
		return result.User == c.targetUser.ID
	case "mention":
		return strings.Contains(result.Text, c.mentionToken)
	default:
		return false
	}
}

func fallbackSearchChannelName(channel slack.CtxChannel) string {
	if channel.Name != "" {
		return channel.Name
	}
	return channel.ID
}

func threadTSFromPermalink(permalink string) string {
	if permalink == "" {
		return ""
	}
	parsed, err := url.Parse(permalink)
	if err != nil {
		return ""
	}
	v := parsed.Query().Get("thread_ts")
	if v != "" {
		return v
	}
	return ""
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

func resolveSlackUser(users []slack.User, handle, userID string) (slack.User, error) {
	if userID != "" {
		for _, user := range users {
			if user.ID == userID {
				return user, nil
			}
		}
		return slack.User{}, fmt.Errorf("could not find Slack user with id %q", userID)
	}

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

func collectThreadMatches(hits []resolvedSlackHit) []SlackMatchRecord {
	all := make([]SlackMatchRecord, 0, len(hits))
	for _, hit := range hits {
		all = append(all, hit.Hit.Matches...)
	}
	return uniqueMatches(all)
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
