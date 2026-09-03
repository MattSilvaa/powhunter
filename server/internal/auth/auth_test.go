package auth_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/auth"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQueries is an in-memory stand-in for the generated query set. It models
// the one behaviour the tests care about: consuming a login token is
// conditional, so it can only succeed once.
type fakeQueries struct {
	user        dbgen.User
	loginTokens map[string]dbgen.LoginToken
	sessions    map[string]dbgen.Session
	verified    []uuid.UUID
	touched     int
}

func newFakeQueries() *fakeQueries {
	return &fakeQueries{
		user: dbgen.User{
			Uuid:  uuid.New(),
			Email: "rider@example.com",
			Phone: sql.NullString{String: "+15555550123", Valid: true},
		},
		loginTokens: map[string]dbgen.LoginToken{},
		sessions:    map[string]dbgen.Session{},
	}
}

func (f *fakeQueries) GetOrCreateUserByEmail(_ context.Context, _ string) (dbgen.User, error) {
	return f.user, nil
}

func (f *fakeQueries) CreateLoginToken(
	_ context.Context,
	arg dbgen.CreateLoginTokenParams,
) (dbgen.LoginToken, error) {
	token := dbgen.LoginToken{
		UserUuid:  arg.UserUuid,
		TokenHash: arg.TokenHash,
		ExpiresAt: arg.ExpiresAt,
	}
	f.loginTokens[arg.TokenHash] = token

	return token, nil
}

func (f *fakeQueries) ConsumeLoginToken(_ context.Context, tokenHash string) (uuid.UUID, error) {
	token, ok := f.loginTokens[tokenHash]
	if !ok || token.ConsumedAt.Valid || time.Now().After(token.ExpiresAt) {
		return uuid.Nil, sql.ErrNoRows
	}

	token.ConsumedAt = sql.NullTime{Time: time.Now(), Valid: true}
	f.loginTokens[tokenHash] = token

	return token.UserUuid, nil
}

func (f *fakeQueries) DeleteExpiredLoginTokens(context.Context) error { return nil }

func (f *fakeQueries) CreateSession(_ context.Context, arg dbgen.CreateSessionParams) (dbgen.Session, error) {
	session := dbgen.Session{
		Uuid:      uuid.New(),
		UserUuid:  arg.UserUuid,
		TokenHash: arg.TokenHash,
		ExpiresAt: arg.ExpiresAt,
	}
	f.sessions[arg.TokenHash] = session

	return session, nil
}

func (f *fakeQueries) GetSessionByTokenHash(
	_ context.Context,
	tokenHash string,
) (dbgen.GetSessionByTokenHashRow, error) {
	session, ok := f.sessions[tokenHash]
	if !ok || time.Now().After(session.ExpiresAt) {
		return dbgen.GetSessionByTokenHashRow{}, sql.ErrNoRows
	}

	return dbgen.GetSessionByTokenHashRow{
		SessionUuid: session.Uuid,
		ExpiresAt:   session.ExpiresAt,
		UserUuid:    f.user.Uuid,
		Email:       f.user.Email,
		Phone:       f.user.Phone,
	}, nil
}

func (f *fakeQueries) TouchSession(_ context.Context, _ dbgen.TouchSessionParams) error {
	f.touched++
	return nil
}

func (f *fakeQueries) DeleteSession(_ context.Context, tokenHash string) error {
	delete(f.sessions, tokenHash)
	return nil
}

func (f *fakeQueries) DeleteExpiredSessions(context.Context) error { return nil }

func (f *fakeQueries) MarkEmailVerified(_ context.Context, userUUID uuid.UUID) error {
	f.verified = append(f.verified, userUUID)
	return nil
}

// The token handed to the user must never be what the database stores,
// otherwise a leak of the tables is a leak of working credentials.
func TestStoredTokenIsOnlyAHash(t *testing.T) {
	queries := newFakeQueries()
	service := auth.NewService(queries, true)

	token, err := service.IssueLoginToken(context.Background(), "rider@example.com")

	require.NoError(t, err)
	require.NotEmpty(t, token)
	assert.NotContains(t, queries.loginTokens, token)
	assert.Contains(t, queries.loginTokens, auth.HashToken(token))
}

func TestALoginLinkWorksExactlyOnce(t *testing.T) {
	queries := newFakeQueries()
	service := auth.NewService(queries, true)

	token, err := service.IssueLoginToken(context.Background(), "rider@example.com")
	require.NoError(t, err)

	sessionToken, expiresAt, err := service.Redeem(context.Background(), token)

	require.NoError(t, err)
	assert.NotEmpty(t, sessionToken)
	assert.True(t, expiresAt.After(time.Now()))
	assert.Equal(t, []uuid.UUID{queries.user.Uuid}, queries.verified)

	_, _, err = service.Redeem(context.Background(), token)

	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestRedeemRejectsAnUnknownToken(t *testing.T) {
	service := auth.NewService(newFakeQueries(), true)

	_, _, err := service.Redeem(context.Background(), "not-a-real-token")

	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestAuthenticateResolvesTheSessionUser(t *testing.T) {
	queries := newFakeQueries()
	service := auth.NewService(queries, true)

	token, err := service.IssueLoginToken(context.Background(), "rider@example.com")
	require.NoError(t, err)

	sessionToken, _, err := service.Redeem(context.Background(), token)
	require.NoError(t, err)

	user, err := service.Authenticate(context.Background(), sessionToken)

	require.NoError(t, err)
	assert.Equal(t, queries.user.Uuid, user.UUID)
	assert.Equal(t, "rider@example.com", user.Email)
}

func TestAuthenticateRejectsAbsentAndUnknownTokens(t *testing.T) {
	service := auth.NewService(newFakeQueries(), true)

	_, err := service.Authenticate(context.Background(), "")
	require.ErrorIs(t, err, auth.ErrNoSession)

	_, err = service.Authenticate(context.Background(), "some-other-token")
	require.ErrorIs(t, err, auth.ErrNoSession)
}

// Signing out must revoke the session server-side, not just drop the cookie.
func TestLogoutRevokesTheSession(t *testing.T) {
	queries := newFakeQueries()
	service := auth.NewService(queries, true)

	token, err := service.IssueLoginToken(context.Background(), "rider@example.com")
	require.NoError(t, err)

	sessionToken, _, err := service.Redeem(context.Background(), token)
	require.NoError(t, err)

	require.NoError(t, service.Logout(context.Background(), sessionToken))

	_, err = service.Authenticate(context.Background(), sessionToken)
	require.ErrorIs(t, err, auth.ErrNoSession)
}
