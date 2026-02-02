package prompt

import (
	"context"
	"github.com/nathfavour/vibeauracle/model"
)

// NoopRecommender is the default recommender; it never triggers network/model calls.
type NoopRecommender struct{}

func (n *NoopRecommender) Generate(ctx context.Context, prompt string) (string, model.Usage, error) {
	return "", model.Usage{}, nil
}

func (n *NoopRecommender) SetStreamCallbacks(onDelta func(string), onDone func(string)) {
}

func (n *NoopRecommender) Recommend(ctx context.Context, in RecommendInput) ([]Recommendation, error) {
	return nil, nil
}