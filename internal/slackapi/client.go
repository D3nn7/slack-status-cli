// Package slackapi is a thin, testable wrapper around the Slack Web API.
package slackapi

import (
	"context"
	"time"

	"github.com/D3nn7/slack-status-cli/internal/domain"
	"github.com/slack-go/slack"
)

// User is the subset of Slack user information the UI needs.
type User struct {
	ID          string
	Name        string
	DisplayName string
	RealName    string
}

// Label returns the most human-friendly available name.
func (u User) Label() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.RealName != "" {
		return u.RealName
	}
	return u.Name
}

// Client is the interface the application depends on, allowing fakes in tests.
type Client interface {
	GetStatus(ctx context.Context) (domain.Status, error)
	SetStatus(ctx context.Context, status domain.Status) error
	AuthTest(ctx context.Context) (User, error)
}

// APIClient implements Client using slack-go.
type APIClient struct {
	api *slack.Client
}

// New creates an APIClient for the given user token.
func New(token string) *APIClient {
	return &APIClient{api: slack.New(token)}
}

// GetStatus fetches the authenticated user's current custom status.
func (c *APIClient) GetStatus(ctx context.Context) (domain.Status, error) {
	profile, err := c.api.GetUserProfileContext(ctx, &slack.GetUserProfileParameters{})
	if err != nil {
		return domain.Status{}, err
	}
	status := domain.Status{
		Text:  profile.StatusText,
		Emoji: profile.StatusEmoji,
	}
	if profile.StatusExpiration > 0 {
		status.Expiration = time.Unix(int64(profile.StatusExpiration), 0)
	}
	return status, nil
}

// SetStatus applies the given custom status.
func (c *APIClient) SetStatus(ctx context.Context, status domain.Status) error {
	var expiration int64
	if !status.Expiration.IsZero() {
		expiration = status.Expiration.Unix()
	}
	return c.api.SetUserCustomStatusContext(ctx, status.Text, status.Emoji, expiration)
}

// AuthTest validates the token and returns basic user information.
func (c *APIClient) AuthTest(ctx context.Context) (User, error) {
	resp, err := c.api.AuthTestContext(ctx)
	if err != nil {
		return User{}, err
	}
	user := User{ID: resp.UserID, Name: resp.User}
	if profile, err := c.api.GetUserProfileContext(ctx, &slack.GetUserProfileParameters{UserID: resp.UserID}); err == nil {
		user.DisplayName = profile.DisplayName
		user.RealName = profile.RealName
	}
	return user, nil
}
