package projects

import (
	"fmt"
	"strings"
	"time"
)

// ============================================
// Types
// ============================================

type ProjectEntry struct {
	ID          string
	Name        string
	Slug        string
	URL         string
	Description string
	Service     string
	Forks       int
	Watchers    int
	Forked      bool
}

type ProjectBoardStats struct {
	CommitCount      int
	ContributorCount int
}

type ProjectGameActivitySummary struct {
	HasGame          bool
	Board            *ProjectBoardStats
	CommitsThisMonth int
	CommitsThisWeek  int
	FileTouchCount   int
	UniqueDirs       int
}

type ProjectAnalysisTile struct {
	MetricKey string
	Level     int16
	Score     int16
	I18nKey   string
}

type ProjectAnalysisBoard struct {
	Tiles                  []ProjectAnalysisTile
	EarnedPts              int
	MaxPts                 int
	LastAnalyzedAgo        string
	AnalysisRunCount       int
	MetricAIEnabled        bool
	RescanL1Slug           string
	RescanL1Disabled       bool
	RescanL1DisabledReason string
}

func (b ProjectAnalysisBoard) ProgressPct() int {
	if b.MaxPts <= 0 {
		return 0
	}
	return (b.EarnedPts * 100) / b.MaxPts
}

type CommitEntry struct {
	ID         string
	Hash       string
	Message    string
	Author     string
	URL        string
	Timestamp  time.Time
	Languages  []string
	IsVerified bool
	IsFlagged  bool
	FlagReason string
}

func (c CommitEntry) ShortHash() string {
	if len(c.Hash) >= 7 {
		return c.Hash[:7]
	}
	return c.Hash
}

func (c CommitEntry) ShortMessage() string {
	msg := strings.SplitN(c.Message, "\n", 2)[0]
	if len(msg) > 120 {
		return msg[:117] + "..."
	}
	return msg
}

// ============================================
// Project List
// ============================================

type ProjectListData struct {
	Entries    []ProjectEntry
	Search     string
	Service    string
	NextCursor string
	HasMore    bool
}

// listURL builds the HTMX URL preserving current search/service params.
func (d ProjectListData) listURL() string {
	params := []string{}
	if d.Search != "" {
		params = append(params, fmt.Sprintf("search=%s", d.Search))
	}
	if d.Service != "" {
		params = append(params, fmt.Sprintf("service=%s", d.Service))
	}
	if d.NextCursor != "" {
		params = append(params, fmt.Sprintf("cursor=%s", d.NextCursor))
	}
	if len(params) == 0 {
		return "/projects"
	}
	return "/projects?" + strings.Join(params, "&")
}
