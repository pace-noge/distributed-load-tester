package http

import (
	"context"
	"net/http"

	"github.com/pace-noge/distributed-load-tester/internal/domain"
)

type contextKey string

const userContextKey contextKey = "user"

// setUserInContext adds a user to the request context
func setUserInContext(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// setUserProfileInContext adds a user profile to the request context
func setUserProfileInContext(ctx context.Context, user *domain.UserProfile) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// getUserFromContext retrieves a user from the request context
func getUserFromContext(r *http.Request) *domain.User {
	user, ok := r.Context().Value(userContextKey).(*domain.User)
	if !ok {
		return nil
	}
	return user
}

// getUserProfileFromContext retrieves a user profile from the request context
func getUserProfileFromContext(r *http.Request) *domain.UserProfile {
	user, ok := r.Context().Value(userContextKey).(*domain.UserProfile)
	if !ok {
		return nil
	}
	return user
}
