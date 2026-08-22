package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/internal/infrastructure/github"
	"github.com/AyushCN/berth/pkg/crypto"
)

type AuthUsecase struct {
	userRepo    domain.UserRepository
	oauthClient *github.OAuthClient
	jwtSecret   string
}

func NewAuthUsecase(repo domain.UserRepository, client *github.OAuthClient, secret string) *AuthUsecase {
	return &AuthUsecase{userRepo: repo, oauthClient: client, jwtSecret: secret}
}

func (uc *AuthUsecase) GenerateState() string {
	return uuid.New().String()
}

func (uc *AuthUsecase) ProcessCallback(ctx context.Context, code, verifier string) (string, any, error) {
	token, err := uc.oauthClient.ExchangeToken(code, verifier)
	if err != nil {
		return "", nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	ghUser, err := uc.oauthClient.GetUser(token)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get user: %w", err)
	}

	email := ghUser.Email
	if email == "" {
		email, err = uc.oauthClient.GetPrimaryEmail(token)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get primary email: %w", err)
		}
	}

	encryptedToken, err := crypto.Encrypt(token)
	if err != nil {
		return "", nil, fmt.Errorf("failed to encrypt token: %w", err)
	}

	user, err := uc.userRepo.GetByGithubID(ctx, fmt.Sprint(ghUser.ID))
	if err != nil {
		// Create new user
		user = &domain.User{
			ID:                   uuid.New(),
			Email:                email,
			Username:             ghUser.Login,
			GithubID:             fmt.Sprint(ghUser.ID),
			GithubUsername:       ghUser.Login,
			GithubTokenEncrypted: encryptedToken,
			AvatarURL:            ghUser.AvatarURL,
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
