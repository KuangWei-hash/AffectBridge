package service

import (
	"context"
	"errors"
	"strings"

	"github.com/KuangWei-hash/AffectBridge/internal/llm"
	"github.com/KuangWei-hash/AffectBridge/internal/model"
	"github.com/KuangWei-hash/AffectBridge/internal/repository"
)

// StoryPipeline owns the recent-dialogue window around StoryService. It keeps
// the current player message ephemeral until the caller has successfully
// generated the matching character reply and commits the completed exchange.
type StoryPipeline struct {
	stories *StoryService
	history repository.DialogueRepository
}

func NewStoryPipeline(stories *StoryService, history repository.DialogueRepository) *StoryPipeline {
	return &StoryPipeline{stories: stories, history: history}
}

// BuildForPlayerMessage builds a Story from at most nine previously committed
// utterances plus the current player message. It does not mutate history.
func (p *StoryPipeline) BuildForPlayerMessage(
	ctx context.Context,
	characterID string,
	characterName string,
	playerMessage string,
) (model.Story, error) {
	if strings.TrimSpace(characterID) == "" {
		return model.Story{}, errors.New("story pipeline: character ID is empty")
	}
	playerMessage = strings.TrimSpace(playerMessage)
	if playerMessage == "" {
		return model.Story{}, errors.New("story pipeline: player message is empty")
	}

	recent, err := p.history.Recent(characterID, llm.StoryDialogueWindow-1)
	if err != nil {
		return model.Story{}, err
	}
	recent = append(recent, model.DialogueMessage{
		Role:    model.DialogueRolePlayer,
		Content: playerMessage,
	})
	return p.stories.Build(ctx, characterName, recent)
}

// CommitExchange stores a completed player/character pair atomically from the
// pipeline caller's point of view. Call it only after reply generation succeeds.
func (p *StoryPipeline) CommitExchange(characterID, playerMessage, characterReply string) error {
	playerMessage = strings.TrimSpace(playerMessage)
	characterReply = strings.TrimSpace(characterReply)
	if playerMessage == "" || characterReply == "" {
		return errors.New("story pipeline: completed exchange contains an empty message")
	}
	return p.history.Append(
		characterID,
		model.DialogueMessage{Role: model.DialogueRolePlayer, Content: playerMessage},
		model.DialogueMessage{Role: model.DialogueRoleCharacter, Content: characterReply},
	)
}
