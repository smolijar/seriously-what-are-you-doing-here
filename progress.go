package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rusq/slack"
)

type ProgressReporter struct {
	monthLabel    string
	windowStart   time.Time
	windowEnd     time.Time
	startedAt     time.Time
	totalWork     int
	completedWork int
	slackRecords  int
	githubCommits int
}

func NewProgressReporter(month MonthRange, cfg Config) *ProgressReporter {
	windowStart, windowEnd := clipMonth(month, cfg)
	return &ProgressReporter{
		monthLabel:  month.Label,
		windowStart: windowStart,
		windowEnd:   windowEnd,
		startedAt:   time.Now(),
	}
}

func (p *ProgressReporter) StartMonth() {
	logf("[%s] starting month window %s -> %s", p.monthLabel, p.windowStart.Format(time.DateOnly), p.windowEnd.Format(time.DateOnly))
}

func (p *ProgressReporter) AddPlannedWork(n int) {
	if n > 0 {
		p.totalWork += n
	}
}

func (p *ProgressReporter) SearchQueryStart(name, query string) {
	logf("[%s] search %s query=%q progress=%s eta=%s", p.monthLabel, name, query, p.progressString(), p.etaString())
}

func (p *ProgressReporter) SearchQueryDone(name string, matches int) {
	p.doneWork()
	logf("[%s] search %s matches=%d progress=%s eta=%s", p.monthLabel, name, matches, p.progressString(), p.etaString())
}

func (p *ProgressReporter) ResolveHit(index, total int, channelName, ts, text string) {
	logf("[%s] hit %d/%d %s ts=%s progress=%s eta=%s preview=%q", p.monthLabel, index, total, channelName, ts, p.progressString(), p.etaString(), previewText(text, 88))
}

func (p *ProgressReporter) ResolveHitDone() {
	p.doneWork()
}

func (p *ProgressReporter) StartThread(channelName, threadTS string, matches int) {
	logf("[%s] thread %s matches=%d ts=%s progress=%s eta=%s", p.monthLabel, channelName, matches, threadTS, p.progressString(), p.etaString())
}

func (p *ProgressReporter) SlackRecord(record SlackConversationRecord) {
	p.slackRecords++
	logf("[%s] slack %s %s progress=%s eta=%s preview=%q", p.monthLabel, recordKind(record), record.ChannelName, p.progressString(), p.etaString(), previewText(recordPreview(record), 88))
}

func (p *ProgressReporter) ThreadDone() {
	p.doneWork()
}

func (p *ProgressReporter) StartGitHubRepo(index, total int, repo string) {
	logf("[%s] github repo %d/%d %s progress=%s eta=%s", p.monthLabel, index, total, repo, p.progressString(), p.etaString())
}

func (p *ProgressReporter) GitHubCommit(record GitHubCommitRecord) {
	p.githubCommits++
	logf("[%s] github %s %s progress=%s eta=%s preview=%q", p.monthLabel, record.Repo, shortSHA(record.SHA), p.progressString(), p.etaString(), previewText(firstLine(record.Message), 88))
	p.doneWork()
}

func (p *ProgressReporter) GitHubRepoDone(repo string, commits int) {
	logf("[%s] github repo %s commits=%d progress=%s eta=%s", p.monthLabel, repo, commits, p.progressString(), p.etaString())
}

func (p *ProgressReporter) Summary() string {
	return fmt.Sprintf("progress=%s eta=%s slack=%d github=%d", p.progressString(), p.etaString(), p.slackRecords, p.githubCommits)
}

func (p *ProgressReporter) progressString() string {
	if p.totalWork <= 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(p.completedWork)/float64(p.totalWork)*100)
}

func (p *ProgressReporter) etaString() string {
	if p.completedWork <= 0 || p.totalWork <= p.completedWork {
		if p.totalWork > 0 && p.totalWork == p.completedWork {
			return "<1s"
		}
		return "--"
	}
	elapsed := time.Since(p.startedAt)
	remainingUnits := p.totalWork - p.completedWork
	remaining := time.Duration(float64(elapsed) * float64(remainingUnits) / float64(p.completedWork))
	return remaining.Round(time.Second).String()
}

func (p *ProgressReporter) doneWork() {
	if p.completedWork < p.totalWork {
		p.completedWork++
	}
}

func recordPreview(record SlackConversationRecord) string {
	if len(record.Messages) == 0 {
		return ""
	}
	return record.Messages[0].Text
}

func recordKind(record SlackConversationRecord) string {
	if record.ThreadTS != "" {
		return "thread"
	}
	if len(record.Matches) > 0 {
		return record.Matches[0].MatchedAs
	}
	return "message"
}

func previewText(text string, max int) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if clean == "" {
		return ""
	}
	runes := []rune(clean)
	if len(runes) <= max {
		return clean
	}
	return string(runes[:max-1]) + "..."
}

func parseSlackTimestamp(value string) (time.Time, error) {
	sec, frac, ok := strings.Cut(value, ".")
	if !ok {
		return time.Time{}, fmt.Errorf("invalid Slack ts %q", value)
	}
	seconds, err := strconv.ParseInt(sec, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	frac = (frac + "000000")[:6]
	micros, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(seconds, micros*int64(time.Microsecond)).UTC(), nil
}

func firstLine(value string) string {
	line, _, _ := strings.Cut(value, "\n")
	return line
}

func shortSHA(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func channelLabel(channel slack.Channel) string {
	name := channelName(channel)
	if name == channel.ID {
		return channel.ID
	}
	return fmt.Sprintf("%s (%s)", name, channel.ID)
}

func logf(format string, args ...any) {
	stamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s\n", stamp, fmt.Sprintf(format, args...))
}
