package oauth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type githubRoundTripper func(*http.Request) (*http.Response, error)

func (fn githubRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func withGitHubHTTPClient(t *testing.T, roundTrip githubRoundTripper) {
	t.Helper()
	original := githubHTTPClient
	githubHTTPClient = &http.Client{Transport: roundTrip, Timeout: time.Second}
	t.Cleanup(func() { githubHTTPClient = original })
}

func TestGitHubProviderGetUserInfoCarriesServerVerifiedCreatedAt(t *testing.T) {
	withGitHubHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "https://api.github.com/user", req.URL.String())
		assert.Equal(t, "Bearer oauth-secret", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"id": 583231,
				"login": "octocat",
				"name": "The Octocat",
				"email": null,
				"created_at": "2011-01-25T18:44:36Z"
			}`)),
			Header: make(http.Header),
		}, nil
	})

	user, err := (&GitHubProvider{}).GetUserInfo(context.Background(), &OAuthToken{AccessToken: "oauth-secret"})
	require.NoError(t, err)
	assert.Equal(t, "583231", user.ProviderUserID)
	assert.Equal(t, "octocat", user.Username)
	assert.Equal(t, time.Date(2011, time.January, 25, 18, 44, 36, 0, time.UTC).Unix(), user.CreatedAt)
	assert.Empty(t, user.Email)
}

func TestGitHubProviderGetUserInfoLeavesMissingCreatedAtForRegistrationGate(t *testing.T) {
	withGitHubHTTPClient(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":583231,"login":"octocat"}`)),
			Header:     make(http.Header),
		}, nil
	})

	user, err := (&GitHubProvider{}).GetUserInfo(context.Background(), &OAuthToken{AccessToken: "oauth-secret"})
	require.NoError(t, err, "existing bound accounts must still be able to log in")
	assert.Zero(t, user.CreatedAt)
}

func TestResolveGitHubUserUsesFixedGitHubAPIAndRejectsInvalidUsername(t *testing.T) {
	requestCount := 0
	withGitHubHTTPClient(t, func(req *http.Request) (*http.Response, error) {
		requestCount++
		assert.Equal(t, "https://api.github.com/users/octocat", req.URL.String())
		assert.Empty(t, req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":583231,"login":"octocat","created_at":"2011-01-25T18:44:36Z"}`)),
			Header:     make(http.Header),
		}, nil
	})

	identity, err := ResolveGitHubUser(context.Background(), "octocat")
	require.NoError(t, err)
	assert.Equal(t, int64(583231), identity.ID)
	assert.Equal(t, "octocat", identity.Login)
	assert.Positive(t, identity.CreatedAt)

	_, err = ResolveGitHubUser(context.Background(), "../user")
	require.Error(t, err)
	assert.Equal(t, 1, requestCount)
}
