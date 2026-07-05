package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type userStore interface {
	Create(ctx context.Context, email string, passwordHash *string, role models.UserRole, status models.UserStatus) (*models.User, error)
	List(ctx context.Context) ([]*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateStatus(ctx context.Context, id string, status models.UserStatus) (*models.User, error)
	TouchLastLogin(ctx context.Context, id string) error
	CountActiveAdmins(ctx context.Context) (int, error)
	UpsertBootstrapAdmin(ctx context.Context, email, passwordHash string) (*models.User, error)
	DeleteInvitedByID(ctx context.Context, id string) error
}

type refreshTokenStore interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (*store.RefreshTokenSession, error)
	Rotate(ctx context.Context, currentTokenHash, nextTokenHash string, expiresAt time.Time) (*models.RefreshToken, error)
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}

type invitationTokenStore interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*models.InvitationToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*store.InvitationTokenSession, error)
	ActivateByTokenHash(ctx context.Context, tokenHash, passwordHash string) (*models.User, error)
	DeleteByUserID(ctx context.Context, userID string) error
}

type inviteMailer interface {
	SendAdminInvite(ctx context.Context, toEmail, inviterEmail, activationURL string, expiresAt time.Time) error
}

type Config struct {
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	InviteTokenTTL        time.Duration
	RefreshCookieName     string
	RefreshCookieSecure   bool
	RefreshCookiePath     string
	RefreshCookieDomain   string
	RefreshCookieSameSite string
	AppBaseURL            string
}

type Session struct {
	User                  *models.User `json:"user"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"-"`
	RefreshTokenExpiresAt time.Time    `json:"-"`
}

type InvitationPreview struct {
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Service struct {
	users         userStore
	refreshTokens refreshTokenStore
	invitations   invitationTokenStore
	accessTokens  *AccessTokenManager
	inviteMailer  inviteMailer
	config        Config
}

func NewService(
	users userStore,
	refreshTokens refreshTokenStore,
	invitations invitationTokenStore,
	cfg Config,
	jwtSecret string,
	inviteMailer inviteMailer,
) *Service {
	return &Service{
		users:         users,
		refreshTokens: refreshTokens,
		invitations:   invitations,
		accessTokens:  NewAccessTokenManager(jwtSecret),
		inviteMailer:  inviteMailer,
		config:        cfg,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (*Session, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if user.Status == models.UserStatusInvited {
		return nil, ErrInvitationPending
	}
	if user.Status != models.UserStatusActive {
		return nil, ErrInactiveUser
	}
	if err := ComparePassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := s.users.TouchLastLogin(ctx, user.ID); err != nil {
		return nil, err
	}
	user.LastLoginAt = timePtr(time.Now().UTC())

	return s.issueSession(ctx, user, "")
}

func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (*Session, error) {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil, ErrInvalidRefreshToken
	}

	tokenHash := HashToken(rawRefreshToken)
	session, err := s.refreshTokens.GetSessionByTokenHash(ctx, tokenHash)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}
	if session.User.Status != models.UserStatusActive {
		_ = s.refreshTokens.RevokeAllByUserID(ctx, session.User.ID)
		return nil, ErrInactiveUser
	}
	if session.User.Role != models.UserRoleAdmin {
		return nil, ErrForbiddenUser
	}

	nextSession, err := s.issueSession(ctx, session.User, tokenHash)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, err
	}

	return nextSession, nil
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil
	}

	if err := s.refreshTokens.RevokeByTokenHash(ctx, HashToken(rawRefreshToken)); err != nil {
		return err
	}

	return nil
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, token string) (*models.User, error) {
	claims, err := s.accessTokens.Parse(token)
	if err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, claims.Subject)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrInvalidAccessToken
	}
	if err != nil {
		return nil, err
	}
	if user.Status != models.UserStatusActive {
		return nil, ErrInactiveUser
	}
	if user.Role != models.UserRoleAdmin {
		return nil, ErrForbiddenUser
	}

	return user, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]*models.User, error) {
	return s.users.List(ctx)
}

func (s *Service) InviteAdmin(ctx context.Context, invitedByEmail, email string) (*models.User, error) {
	if s.invitations == nil || s.inviteMailer == nil || strings.TrimSpace(s.config.AppBaseURL) == "" {
		return nil, ErrMailerUnavailable
	}

	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}

	existingUser, err := s.users.GetByEmail(ctx, email)
	switch {
	case err == nil && existingUser.Status == models.UserStatusInvited:
		return nil, ErrInviteAlreadyPending
	case err == nil:
		return nil, store.ErrConflict
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}

	user, err := s.users.Create(ctx, email, nil, models.UserRoleAdmin, models.UserStatusInvited)
	if err != nil {
		return nil, err
	}

	rawToken, tokenHash, err := GenerateOpaqueToken()
	if err != nil {
		_ = s.users.DeleteInvitedByID(ctx, user.ID)
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(s.config.InviteTokenTTL)
	if _, err := s.invitations.Create(ctx, user.ID, tokenHash, expiresAt); err != nil {
		_ = s.users.DeleteInvitedByID(ctx, user.ID)
		if errors.Is(err, store.ErrConflict) {
			return nil, ErrInviteAlreadyPending
		}
		return nil, err
	}

	activationURL, err := s.buildInviteActivationURL(rawToken)
	if err != nil {
		_ = s.invitations.DeleteByUserID(ctx, user.ID)
		_ = s.users.DeleteInvitedByID(ctx, user.ID)
		return nil, err
	}

	if err := s.inviteMailer.SendAdminInvite(ctx, user.Email, invitedByEmail, activationURL, expiresAt); err != nil {
		_ = s.invitations.DeleteByUserID(ctx, user.ID)
		_ = s.users.DeleteInvitedByID(ctx, user.ID)
		return nil, fmt.Errorf("send invite email: %w", err)
	}

	return user, nil
}

func (s *Service) GetInvitation(ctx context.Context, rawToken string) (*InvitationPreview, error) {
	if strings.TrimSpace(rawToken) == "" || s.invitations == nil {
		return nil, ErrInvalidInvitationToken
	}

	invitation, err := s.invitations.GetByTokenHash(ctx, HashToken(rawToken))
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrInvalidInvitationToken
	}
	if err != nil {
		return nil, err
	}

	return &InvitationPreview{
		Email:     invitation.User.Email,
		ExpiresAt: invitation.Token.ExpiresAt,
	}, nil
}

func (s *Service) ActivateInvitation(ctx context.Context, rawToken, password string) (*Session, error) {
	if strings.TrimSpace(rawToken) == "" || s.invitations == nil {
		return nil, ErrInvalidInvitationToken
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.invitations.ActivateByTokenHash(ctx, HashToken(rawToken), passwordHash)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrInvalidInvitationToken
	}
	if err != nil {
		return nil, err
	}

	if err := s.users.TouchLastLogin(ctx, user.ID); err != nil {
		return nil, err
	}
	user.LastLoginAt = timePtr(time.Now().UTC())

	return s.issueSession(ctx, user, "")
}

func (s *Service) UpdateUserStatus(ctx context.Context, userID string, status models.UserStatus) (*models.User, error) {
	if status != models.UserStatusActive && status != models.UserStatusInactive {
		return nil, ErrForbiddenUser
	}

	currentUser, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if currentUser.Status == status {
		return currentUser, nil
	}

	if currentUser.Role == models.UserRoleAdmin && currentUser.Status == models.UserStatusActive && status == models.UserStatusInactive {
		activeAdmins, err := s.users.CountActiveAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if activeAdmins <= 1 {
			return nil, ErrLastActiveAdmin
		}
	}

	updatedUser, err := s.users.UpdateStatus(ctx, userID, status)
	if err != nil {
		return nil, err
	}
	if updatedUser.Status == models.UserStatusInactive {
		if err := s.refreshTokens.RevokeAllByUserID(ctx, updatedUser.ID); err != nil {
			return nil, err
		}
	}

	return updatedUser, nil
}

func (s *Service) EnsureBootstrapAdmin(ctx context.Context, email, password string) (*models.User, bool, error) {
	email = NormalizeEmail(email)
	password = strings.TrimSpace(password)

	if email == "" && password == "" {
		return nil, false, nil
	}
	if email == "" || password == "" {
		return nil, false, fmt.Errorf("bootstrap admin email and password must both be set")
	}
	if err := ValidateEmail(email); err != nil {
		return nil, false, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, false, err
	}

	activeAdmins, err := s.users.CountActiveAdmins(ctx)
	if err != nil {
		return nil, false, err
	}
	if activeAdmins > 0 {
		return nil, false, nil
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, false, fmt.Errorf("hash bootstrap password: %w", err)
	}

	user, err := s.users.UpsertBootstrapAdmin(ctx, email, hash)
	if err != nil {
		return nil, false, err
	}

	return user, true, nil
}

func (s *Service) issueSession(ctx context.Context, user *models.User, previousRefreshTokenHash string) (*Session, error) {
	accessToken, accessExpiresAt, err := s.accessTokens.Issue(user, s.config.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	rawRefreshToken, refreshTokenHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshExpiresAt := time.Now().UTC().Add(s.config.RefreshTokenTTL)

	if previousRefreshTokenHash == "" {
		if _, err := s.refreshTokens.Create(ctx, user.ID, refreshTokenHash, refreshExpiresAt); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.refreshTokens.Rotate(ctx, previousRefreshTokenHash, refreshTokenHash, refreshExpiresAt); err != nil {
			return nil, err
		}
	}

	return &Session{
		User:                  user,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshToken:          rawRefreshToken,
		RefreshTokenExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *Service) buildInviteActivationURL(rawToken string) (string, error) {
	baseURL := strings.TrimSpace(s.config.AppBaseURL)
	if baseURL == "" {
		return "", ErrMailerUnavailable
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse app base url: %w", err)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/activate"
	query := parsed.Query()
	query.Set("token", rawToken)
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func timePtr(v time.Time) *time.Time {
	return &v
}
