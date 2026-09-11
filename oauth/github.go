package oauth

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func init() {
	Register("github", &GitHubProvider{})
}

// GitHubProvider implements OAuth for GitHub
type GitHubProvider struct{}

var githubHTTPClient = &http.Client{Timeout: 20 * time.Second}

var githubUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)

type gitHubOAuthResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type gitHubUser struct {
	Id        int64  `json:"id"`    // GitHub numeric ID (permanent, never changes)
	Login     string `json:"login"` // GitHub username (can be changed by user)
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type GitHubIdentity struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	CreatedAt int64  `json:"created_at"`
}

func (p *GitHubProvider) GetName() string {
	return "GitHub"
}

func (p *GitHubProvider) IsEnabled() bool {
	return common.GitHubOAuthEnabled
}

func (p *GitHubProvider) ExchangeToken(ctx context.Context, code string, c *gin.Context) (*OAuthToken, error) {
	if code == "" {
		return nil, NewOAuthError(i18n.MsgOAuthInvalidCode, nil)
	}

	logger.LogDebug(ctx, "[OAuth-GitHub] ExchangeToken: code=%s...", code[:min(len(code), 10)])

	values := map[string]string{
		"client_id":     common.GitHubClientId,
		"client_secret": common.GitHubClientSecret,
		"code":          code,
	}
	jsonData, err := common.Marshal(values)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := githubHTTPClient.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] ExchangeToken error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "GitHub"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-GitHub] ExchangeToken response status: %d", res.StatusCode)

	var oAuthResponse gitHubOAuthResponse
	err = common.DecodeJson(res.Body, &oAuthResponse)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] ExchangeToken decode error: %s", err.Error()))
		return nil, err
	}

	if oAuthResponse.AccessToken == "" {
		logger.LogError(ctx, "[OAuth-GitHub] ExchangeToken failed: empty access token")
		return nil, NewOAuthError(i18n.MsgOAuthTokenFailed, map[string]any{"Provider": "GitHub"})
	}

	logger.LogDebug(ctx, "[OAuth-GitHub] ExchangeToken success: scope=%s", oAuthResponse.Scope)

	return &OAuthToken{
		AccessToken: oAuthResponse.AccessToken,
		TokenType:   oAuthResponse.TokenType,
		Scope:       oAuthResponse.Scope,
	}, nil
}

func (p *GitHubProvider) GetUserInfo(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	logger.LogDebug(ctx, "[OAuth-GitHub] GetUserInfo: fetching user info")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	res, err := githubHTTPClient.Do(req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] GetUserInfo error: %s", err.Error()))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthConnectFailed, map[string]any{"Provider": "GitHub"}, err.Error())
	}
	defer res.Body.Close()

	logger.LogDebug(ctx, "[OAuth-GitHub] GetUserInfo response status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] GetUserInfo failed: status=%d", res.StatusCode))
		return nil, NewOAuthErrorWithRaw(i18n.MsgOAuthGetUserErr, map[string]any{"Provider": "GitHub"}, fmt.Sprintf("status %d", res.StatusCode))
	}

	var githubUser gitHubUser
	err = common.DecodeJson(res.Body, &githubUser)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("[OAuth-GitHub] GetUserInfo decode error: %s", err.Error()))
		return nil, err
	}

	if githubUser.Id == 0 || githubUser.Login == "" {
		logger.LogError(ctx, "[OAuth-GitHub] GetUserInfo failed: empty id or login field")
		return nil, NewOAuthError(i18n.MsgOAuthUserInfoEmpty, map[string]any{"Provider": "GitHub"})
	}

	createdAt := int64(0)
	if parsed, parseErr := time.Parse(time.RFC3339, githubUser.CreatedAt); parseErr == nil {
		createdAt = parsed.Unix()
	}
	logger.LogDebug(ctx, "[OAuth-GitHub] GetUserInfo success: id=%d, login=%s", githubUser.Id, githubUser.Login)

	return &OAuthUser{
		ProviderUserID: strconv.FormatInt(githubUser.Id, 10), // Use numeric ID as primary identifier
		Username:       githubUser.Login,
		DisplayName:    githubUser.Name,
		Email:          githubUser.Email,
		CreatedAt:      createdAt,
		Extra: map[string]any{
			"legacy_id": githubUser.Login, // Store login for migration from old accounts
		},
	}, nil
}

func ResolveGitHubUser(ctx context.Context, username string) (*GitHubIdentity, error) {
	username = strings.TrimSpace(username)
	if !githubUsernamePattern.MatchString(username) {
		return nil, fmt.Errorf("invalid GitHub username")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/users/"+username, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "new-api")

	res, err := githubHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", res.StatusCode)
	}

	var githubUser gitHubUser
	if err := common.DecodeJson(res.Body, &githubUser); err != nil {
		return nil, fmt.Errorf("invalid GitHub API response")
	}
	if githubUser.Id <= 0 || githubUser.Login == "" || githubUser.CreatedAt == "" {
		return nil, fmt.Errorf("GitHub identity is incomplete")
	}
	createdAt, err := time.Parse(time.RFC3339, githubUser.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("GitHub identity has invalid created_at")
	}
	return &GitHubIdentity{ID: githubUser.Id, Login: githubUser.Login, CreatedAt: createdAt.Unix()}, nil
}

func (p *GitHubProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsGitHubIdAlreadyTaken(providerUserID)
}

func (p *GitHubProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.GitHubId = providerUserID
	return user.FillUserByGitHubId()
}

func (p *GitHubProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.GitHubId = providerUserID
}

func (p *GitHubProvider) GetProviderPrefix() string {
	return "github_"
}
