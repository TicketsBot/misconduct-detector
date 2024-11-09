package rules

import (
	"context"
	"errors"
	"fmt"
	"github.com/TicketsBot/misconduct-detector/internal/config"
	"github.com/rxdn/gdl/cache"
	"github.com/rxdn/gdl/objects/user"
	"github.com/rxdn/gdl/rest"
	"github.com/rxdn/gdl/rest/request"
)

type AppContext struct {
	config config.Config
	cache  cache.Cache
}

func NewAppContext(config config.Config, cache cache.Cache) *AppContext {
	return &AppContext{
		config: config,
		cache:  cache,
	}
}

func (a *AppContext) FetchUser(ctx context.Context, userId uint64) (user.User, bool, error) {
	cached, err := a.cache.GetUser(ctx, userId)
	if err == nil {
		return cached, true, nil
	} else if !errors.Is(err, cache.ErrNotFound) {
		return user.User{}, false, fmt.Errorf("failed to fetch user from cache: %w", err)
	}

	// Fetch from API
	fetched, err := rest.GetUser(ctx, a.config.Discord.Token, nil, userId)
	if err != nil {
		var restError request.RestError
		if errors.As(err, &restError) && restError.StatusCode == 404 {
			return user.User{}, false, nil
		}

		return user.User{}, false, fmt.Errorf("failed to fetch user from Discord API: %w", err)
	}

	// Cache
	if err := a.cache.StoreUser(ctx, fetched); err != nil {
		return user.User{}, false, fmt.Errorf("failed to cache user: %w", err)
	}

	return fetched, true, nil
}
