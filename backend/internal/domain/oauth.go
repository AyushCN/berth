package domain

// ExternalUser represents a user profile from an OAuth provider.
type ExternalUser struct {
	ID        string
	Username  string
	Email     string
	AvatarURL string
}

// OAuthProvider defines the interface for OAuth 2.1 clients.
type OAuthProvider interface {
	AuthorizeURL(state, codeChallenge, codeChallengeMethod string) string
	ExchangeToken(code, codeVerifier string) (string, error)
	GetUser(token string) (*ExternalUser, error)
	GetPrimaryEmail(token string) (string, error)
}
