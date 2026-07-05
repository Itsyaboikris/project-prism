package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeServiceUserStore struct{}

func (f *fakeServiceUserStore) Create(context.Context, string, *string, models.UserRole, models.UserStatus) (*models.User, error) {
	return nil, nil
}

func (f *fakeServiceUserStore) List(context.Context) ([]*models.User, error) {
	return nil, nil
}

func (f *fakeServiceUserStore) GetByID(context.Context, string) (*models.User, error) {
	return nil, nil
}

func (f *fakeServiceUserStore) GetByEmail(context.Context, string) (*models.User, error) {
	return nil, nil
}

func (f *fakeServiceUserStore) UpdateStatus(context.Context, string, models.UserStatus) (*models.User, error) {
	return nil, nil
}

func (f *fakeServiceUserStore) TouchLastLogin(context.Context, string) error {
	return nil
}

func (f *fakeServiceUserStore) CountActiveAdmins(context.Context) (int, error) {
	return 0, nil
}

func (f *fakeServiceUserStore) UpsertBootstrapAdmin(context.Context, string, string) (*models.User, error) {
	return nil, nil
}

func (f *fakeServiceUserStore) DeleteInvitedByID(context.Context, string) error {
	return nil
}

type fakeServiceRefreshTokenStore struct {
	getSessionByTokenHashFn func(context.Context, string) (*store.RefreshTokenSession, error)
	rotateFn                func(context.Context, string, string, time.Time) (*models.RefreshToken, error)
}

func (f *fakeServiceRefreshTokenStore) Create(context.Context, string, string, time.Time) (*models.RefreshToken, error) {
	return nil, nil
}

func (f *fakeServiceRefreshTokenStore) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*store.RefreshTokenSession, error) {
	return f.getSessionByTokenHashFn(ctx, tokenHash)
}

func (f *fakeServiceRefreshTokenStore) Rotate(ctx context.Context, currentTokenHash, nextTokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	return f.rotateFn(ctx, currentTokenHash, nextTokenHash, expiresAt)
}

func (f *fakeServiceRefreshTokenStore) RevokeByTokenHash(context.Context, string) error {
	return nil
}

func (f *fakeServiceRefreshTokenStore) RevokeAllByUserID(context.Context, string) error {
	return nil
}

type fakeServiceInvitationStore struct{}

func (f *fakeServiceInvitationStore) Create(context.Context, string, string, time.Time) (*models.InvitationToken, error) {
	return nil, nil
}

func (f *fakeServiceInvitationStore) GetByTokenHash(context.Context, string) (*store.InvitationTokenSession, error) {
	return nil, nil
}

func (f *fakeServiceInvitationStore) ActivateByTokenHash(context.Context, string, string) (*models.User, error) {
	return nil, nil
}

func (f *fakeServiceInvitationStore) DeleteByUserID(context.Context, string) error {
	return nil
}

func TestServiceRefreshTreatsRotationRaceAsInvalidRefreshToken(t *testing.T) {
	refreshTokens := &fakeServiceRefreshTokenStore{
		getSessionByTokenHashFn: func(context.Context, string) (*store.RefreshTokenSession, error) {
			return &store.RefreshTokenSession{
				User: &models.User{
					ID:     "user_123",
					Email:  "admin@example.com",
					Role:   models.UserRoleAdmin,
					Status: models.UserStatusActive,
				},
			}, nil
		},
		rotateFn: func(context.Context, string, string, time.Time) (*models.RefreshToken, error) {
			return nil, store.ErrNotFound
		},
	}

	service := NewService(&fakeServiceUserStore{}, refreshTokens, &fakeServiceInvitationStore{}, Config{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}, "test-secret", nil)

	_, err := service.Refresh(context.Background(), "raw-refresh-token")
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected %v, got %v", ErrInvalidRefreshToken, err)
	}
}
