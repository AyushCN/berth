package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/pkg/crypto"
)

type AuthUsecase struct {
	userRepo      domain.UserRepository
	oauthProvider domain.OAuthProvider
	jwtSecret     string
}

func NewAuthUsecase(repo domain.UserRepository, provider domain.OAuthProvider, secret string) *AuthUsecase {
	return &AuthUsecase{userRepo: repo, oauthProvider: provider, jwtSecret: secret}
}

func (uc *AuthUsecase) GenerateState() string {
	return uuid.New().String()
}

func (uc *AuthUsecase) GenerateAuthorizeURL() (authorizeURL, state, verifier string, err error) {
	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate pkce verifier: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	method := "S256"

	state = uc.GenerateState()
	authorizeURL = uc.oauthProvider.AuthorizeURL(state, challenge, method)

	return authorizeURL, state, verifier, nil
}

func (uc *AuthUsecase) ProcessCallback(ctx context.Context, code, verifier string) (string, any, error) {
	token, err := uc.oauthProvider.ExchangeToken(code, verifier)
	if err != nil {
		return "", nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	extUser, err := uc.oauthProvider.GetUser(token)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get user: %w", err)
	}

	email := extUser.Email
	if email == "" {
		email, err = uc.oauthProvider.GetPrimaryEmail(token)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get primary email: %w", err)
		}
	}

	encryptedToken, err := crypto.Encrypt(token)
	if err != nil {
		return "", nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	user, err := uc.userRepo.GetByGithubID(ctx, extUser.ID)
	if err != nil {
		// Create new user
		user = &domain.User{
			ID:                   uuid.New(),
			Email:                email,
			Username:             extUser.Username,
			GithubID:             extUser.ID,
			GithubUsername:       extUser.Username,
			GithubTokenEncrypted: encryptedToken,
			AvatarURL:            extUser.AvatarURL,
		}
		if err := uc.userRepo.Create(ctx, user); err != nil {
			return "", nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else {
		// Update user token
		user.GithubTokenEncrypted = encryptedToken
		if err := uc.userRepo.Update(ctx, user); err != nil {
			return "", nil, fmt.Errorf("failed to update user token: %w", err)
		}
	}

	claims := jwt.MapClaims{
		"userId": user.ID.String(),
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := jwtToken.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign jwt: %w", err)
	}

	return tokenString, user, nil
}

func (uc *AuthUsecase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return uc.userRepo.GetByID(ctx, id)
}
