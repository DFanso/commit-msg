package cmd

import (
	"context"
	"sync"
	"testing"

	"github.com/dfanso/commit-msg/pkg/types"
)

// FakeProvider implements the llm.Provider interface to simulate API responses
type FakeProvider struct{}

func (f FakeProvider) Name() types.LLMProvider { return "fake" }

func (f FakeProvider) Generate(ctx context.Context, changes string, opts *types.GenerationOptions) (string, error) {
	return "mock commit message", nil
}

func TestGenerateMessageRateLimiter(t *testing.T) {
	ctx := context.Background()
	var waitGroup sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Test sending a number of messages in a short period to check the rate limiter
	numCalls := 100
	waitGroup.Add(numCalls)
	for i := 0; i < numCalls; i++ {
		go func() {
			defer waitGroup.Done()
			_, err := generateMessage(ctx, FakeProvider{}, "", nil)
			if err != nil {
				t.Logf("rate limiter error: %v", err)
				return
			}
			mu.Lock()
			successCount++
			mu.Unlock()
		}()
	}
	waitGroup.Wait()

	t.Logf("Successful calls: %d out of %d", successCount, numCalls)
	if successCount != numCalls {
		t.Errorf("expected %d successful calls but got %d", numCalls, successCount)
	}
}
