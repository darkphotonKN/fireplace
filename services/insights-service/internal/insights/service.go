package insights

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commonconstants "github.com/darkphotonKN/fireplace/common/constants"
	commonhelpers "github.com/darkphotonKN/fireplace/common/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// dailySuggestionCount is how many daily suggestions GenerateDailySuggestions
// requests from the LLM. Ported from the gateway behaviour.
const dailySuggestionCount = 3

// ContentGenerator is the LLM seam: anything that turns a prompt into text.
// Implemented by ai.Generator, which pairs a fixed system prompt with a client.
type ContentGenerator interface {
	Generate(prompt string) (string, error)
}

// VideoFinder resolves generated search terms into concrete videos. Implemented
// by the ported YouTube finder; the composition root supplies the concrete.
type VideoFinder interface {
	FindVideos(ctx context.Context, searchTerms []string) ([]Video, error)
}

// PlanGateway is the slice of plan-service this package depends on. Tests can
// substitute a fake.
type PlanGateway interface {
	GetPlanContext(ctx context.Context, planID, userID uuid.UUID) (*PlanContext, error)
}

// Repository is the persistence seam for insights. The consumer owns the
// abstraction (DIP); the concrete *repository is injected at SetupServices.
type Repository interface {
	Create(ctx context.Context, param CreateInsightParam) error
	CreateTx(ctx context.Context, tx *sqlx.Tx, param CreateInsightParam) error
}

type Service struct {
	plans PlanGateway
	// Two generators, one per system prompt — mirroring the gateway, which built
	// a separate insights service around each. checklistGen backs the suggestion
	// RPCs; searchTermGen backs the video search-term step.
	checklistGen  ContentGenerator
	searchTermGen ContentGenerator
	videos        VideoFinder
	inboxService  InboxService
	cache         Cache
	repo          Repository
	db            *sqlx.DB
}

func NewService(plans PlanGateway, checklistGen, searchTermGen ContentGenerator, videos VideoFinder, cache Cache, repo Repository, db *sqlx.DB) *Service {
	return &Service{
		plans:         plans,
		checklistGen:  checklistGen,
		searchTermGen: searchTermGen,
		videos:        videos,
		cache:         cache,
		repo:          repo,
		db:            db,
	}
}

type InboxService interface {
	CreateTx(ctx context.Context, tx *sqlx.Tx, eventID uuid.UUID) error
}

const (
	inProgressMarker   = "in-progress"
	inboxWriteComplete = "inbox write complete"
)

var (
	ErrEventAlreadyProcessed = errors.New("event was already processed")
	ErrUnexpectedError       = errors.New("unexpected error")
)

// TODO: temp, need to solve the source of truth of this
type InsightType string

var (
	InsightTypeSuggestion InsightType = "suggestion"
)

func (s *Service) Create(ctx context.Context, param CreateInsightFromPlanParam) error {
	// dedup with cache for best-effort, effeciency
	key := fmt.Sprintf("dedup:insights:%s", param.EventID)
	// shorter ttl before DB write, expires and prevents holding the lock for too
	// long if crash happens and subsequent requests are actually allowed to write
	// best effort so we don't care about error at this point, omit
	acquired, _ := s.cache.SetNX(ctx, key, inProgressMarker, time.Second*5).Result()

	if !acquired {
		return ErrEventAlreadyProcessed
	}

	// safe and not duplicate
	if err := commonhelpers.ExecTx(ctx, s.db, func(tx *sqlx.Tx) error {
		// attempt to create dedup table first, rollback on conflict, true authority here
		err := s.inboxService.CreateTx(ctx, tx, param.EventID)

		// rollback if duplicate attempted
		if err != nil {
			// duplicate caught, reject but return sentinel error so boundary can Ack
			// (message handled, event was ALREADY processed, no need to retry)
			if errors.Is(err, commonconstants.ErrDuplicateResource) {
				// use domain specific error to prevent nacks, let boundary ack
				return fmt.Errorf("service attempted to write insights inbox for eventID %s: %w", param.EventID, ErrEventAlreadyProcessed)
			}

			if errors.Is(err, commonconstants.ErrTransient) {
				return fmt.Errorf("service attempted to write insights inbox for eventID %s: %w", param.EventID, err)
			}

			// other types of errors, identify so boundary can log
			return fmt.Errorf("service attempted to write insights inbox for eventID %s: %w: %w", param.EventID, ErrUnexpectedError, err)
		}

		err = s.repo.CreateTx(ctx, tx, CreateInsightParam{
			PlanID:      param.PlanID,
			UserID:      param.UserID,
			InsightType: string(InsightTypeSuggestion),
			Content:     "", // TODO: need to acquire and fill
		})

		// rollback on error, wrapping context
		if err != nil {
			// retry on transient, propogate with context
			if errors.Is(err, commonconstants.ErrTransient) {
				return fmt.Errorf("service attempted to write generated_insights for planID %s, eventID %s: %w", param.PlanID, param.EventID, err)
			}

			// all other errors mean request error, execption, or duplicate, just reject

			return fmt.Errorf("service attempted to write generated_insights for planID %s, eventID %s: %w", param.PlanID, param.EventID, ErrUnexpectedError)
		}

		return nil
	}); err != nil {
		// delete cache early
		s.cache.Del(ctx, key)

		// propogate error
		return err
	}

	// succes, both inbox and business effect

	// update redis key with longer ttl and complete marker
	s.cache.Set(ctx, key, inboxWriteComplete, time.Hour*24)

	return nil
}

// basePrompt is the shared instruction block for single-task suggestions.
// Ported from the api-gateway insights service.
const basePrompt = `
Please suggest ONE specific, actionable task that would be the most valuable next step to add to my checklist.

Your suggestion should:
- Be a single, concrete task (not multiple tasks)
- Start with a verb
- Be specific enough to complete in a single sitting
- Be directly relevant to the project focus
- Use technical terminology accurately if applicable
- Be 4-20 words in length

Format your response as a single task item with no additional commentary, explanation or punctuation at the end.`

// GenerateSuggestion returns a single, actionable next checklist item for the
// plan, derived from its focus + current checklist.
func (s *Service) GenerateSuggestion(ctx context.Context, planID, userID uuid.UUID) (string, error) {
	pc, err := s.plans.GetPlanContext(ctx, planID, userID)
	if err != nil {
		return "", fmt.Errorf("insights: generate suggestion: %w", err)
	}

	res, err := s.checklistGen.Generate(buildChecklistPrompt(pc, ""))
	if err != nil {
		return "", fmt.Errorf("insights: generate suggestion: %w", err)
	}
	return res, nil
}

// GenerateDailySuggestions returns dailySuggestionCount daily focus suggestions
// derived from the plan's longterm checklist items + focus, nudging each draw
// away from the previous so they don't collide.
func (s *Service) GenerateDailySuggestions(ctx context.Context, planID, userID uuid.UUID) ([]string, error) {
	pc, err := s.plans.GetPlanContext(ctx, planID, userID)
	if err != nil {
		return nil, fmt.Errorf("insights: generate daily suggestions: %w", err)
	}

	base := buildChecklistPrompt(pc, `Focus on tasks that are marked as "longterm" and break them down when you make your suggestions.`)

	suggestions := make([]string, 0, dailySuggestionCount)
	for i := 0; i < dailySuggestionCount; i++ {
		prompt := base
		if i > 0 {
			prompt = fmt.Sprintf("%s\nAlso, don't choose one closely related to this, as it has already been added: %s", base, suggestions[i-1])
		}
		res, err := s.checklistGen.Generate(prompt)
		if err != nil {
			return nil, fmt.Errorf("insights: generate daily suggestions: %w", err)
		}
		suggestions = append(suggestions, res)
	}
	return suggestions, nil
}

// SuggestVideos uses the plan focus + checklist to ask the LLM for search terms,
// which (once migrated) feed a video finder.
//
// TODO: the video-finder that turns these search terms into concrete results
// still lives in the api-gateway (internal/discovery). Until it is migrated this
// returns an empty list after the search-term generation step.
func (s *Service) SuggestVideos(ctx context.Context, planID, userID uuid.UUID) ([]Video, error) {
	pc, err := s.plans.GetPlanContext(ctx, planID, userID)
	if err != nil {
		return nil, fmt.Errorf("insights: suggest videos: %w", err)
	}

	raw, err := s.searchTermGen.Generate(buildVideoSearchPrompt(pc))
	if err != nil {
		return nil, fmt.Errorf("insights: suggest videos: generate search terms: %w", err)
	}

	// The prompt asks for one search term per line. Blank lines are dropped: a
	// trailing newline is common in LLM output and an empty term would otherwise
	// become an empty search query whose results are returned as recommendations.
	searchTerms := make([]string, 0, dailySuggestionCount)
	for _, term := range strings.Split(raw, "\n") {
		if t := strings.TrimSpace(term); t != "" {
			searchTerms = append(searchTerms, t)
		}
	}

	if len(searchTerms) == 0 {
		return []Video{}, nil
	}

	videos, err := s.videos.FindVideos(ctx, searchTerms)
	if err != nil {
		return nil, fmt.Errorf("insights: suggest videos: find videos: %w", err)
	}

	return videos, nil
}

// buildChecklistPrompt assembles the checklist-suggestion prompt from plan
// context plus any extra instruction.
func buildChecklistPrompt(pc *PlanContext, additional string) string {
	return fmt.Sprintf(`Based on this project focus: "%s"
%s
So far the checklist already has these items, so either add one that follows the current progress or don't suggest one that's already present.
This is the current existing checklist:
%s
%s`, pc.Focus, basePrompt, formatChecklist(pc), additional)
}

// buildVideoSearchPrompt assembles the prompt that asks for video search terms.
func buildVideoSearchPrompt(pc *PlanContext) string {
	return fmt.Sprintf(`The user's focus for this task: %s
Current checklist items for this task:
%s

Please use this information to now provide exactly 3 relevant search terms.`, pc.Focus, formatChecklist(pc))
}

// formatChecklist flattens checklist items into prompt lines.
func formatChecklist(pc *PlanContext) string {
	var b strings.Builder
	for _, item := range pc.ChecklistItems {
		fmt.Fprintf(&b, "A %s task: %s\n", item.Scope, item.Description)
	}
	return b.String()
}
