package auth

import "errors"

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInactiveUser = errors.New("inactive user")
var ErrInvitationPending = errors.New("invitation pending")
var ErrInvalidRefreshToken = errors.New("invalid refresh token")
var ErrInvalidAccessToken = errors.New("invalid access token")
var ErrInvalidInvitationToken = errors.New("invalid invitation token")
var ErrInviteAlreadyPending = errors.New("invite already pending")
var ErrMailerUnavailable = errors.New("mailer unavailable")
var ErrForbiddenUser = errors.New("forbidden user")
var ErrInvalidEmail = errors.New("invalid email")
var ErrWeakPassword = errors.New("password does not meet requirements")
var ErrLastActiveAdmin = errors.New("cannot deactivate the last active admin")
