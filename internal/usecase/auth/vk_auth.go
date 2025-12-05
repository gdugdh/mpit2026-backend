package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdugdh24/mpit2026-backend/internal/domain"
	"github.com/gdugdh24/mpit2026-backend/internal/repository"
	"github.com/gdugdh24/mpit2026-backend/pkg/vkapi"
	"github.com/golang-jwt/jwt/v5"
)

type VKAuthUseCase struct {
	userRepo    repository.UserRepository
	profileRepo repository.ProfileRepository
	sessionRepo repository.SessionRepository
	vkSecret    string
	jwtSecret   string
	vkAPIClient *vkapi.Client
}

func NewVKAuthUseCase(
	userRepo repository.UserRepository,
	profileRepo repository.ProfileRepository,
	sessionRepo repository.SessionRepository,
	vkSecret string,
	jwtSecret string,
) *VKAuthUseCase {
	return &VKAuthUseCase{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		sessionRepo: sessionRepo,
		vkSecret:    vkSecret,
		jwtSecret:   jwtSecret,
		vkAPIClient: vkapi.NewClient(),
	}
}

// VKLaunchParams represents VK Mini App launch parameters
type VKLaunchParams struct {
	VKID                    int    `json:"vk_user_id"`
	AppID                   int    `json:"vk_app_id"`
	IsAppUser               int    `json:"vk_is_app_user"`
	AreNotificationsEnabled int    `json:"vk_are_notifications_enabled"`
	Language                string `json:"vk_language"`
	Platform                string `json:"vk_platform"`
	AccessTokenSettings     string `json:"vk_access_token_settings"`
	Sign                    string `json:"sign"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *domain.User `json:"user"`
	IsNewUser bool         `json:"is_new_user"`
}

// AuthenticateVK authenticates user via VK Mini App launch params
func (uc *VKAuthUseCase) AuthenticateVK(ctx context.Context, params map[string]string, accessToken, deviceInfo, ipAddress string) (*AuthResponse, error) {
	// Verify VK signature
	// ВРЕМЕННО ОТКЛЮЧЕНО для отладки
	fmt.Println("⚠️  WARNING: VK signature verification is DISABLED for debugging")
	// if err := uc.verifyVKSignature(params); err != nil {
	// 	return nil, domain.ErrInvalidVKSignature
	// }

	vkID := 0
	fmt.Sscanf(params["vk_user_id"], "%d", &vkID)
	if vkID == 0 {
		fmt.Println("❌ ERROR: vk_user_id is 0")
		return nil, domain.ErrInvalidInput
	}

	fmt.Printf("📝 Fetching VK user info for ID: %d with access_token: %s...\n", vkID, accessToken[:20]+"...")

	// Fetch user info from VK API
	vkUserInfo, err := uc.vkAPIClient.GetUserInfo(accessToken, vkID)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to fetch VK user info: %v\n", err)
		return nil, fmt.Errorf("failed to fetch VK user info: %w", err)
	}

	fmt.Printf("✅ VK user info received: %s %s\n", vkUserInfo.FirstName, vkUserInfo.LastName)

	// Try to get existing user
	user, err := uc.userRepo.GetByVKID(ctx, vkID)
	isNewUser := false

	if err == domain.ErrUserNotFound {
		// Create new user with VK data
		user, err = uc.createUserFromVKInfo(ctx, vkUserInfo, accessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		isNewUser = true
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	} else {
		// Update existing user's access token
		tokenExpiry := time.Now().Add(24 * time.Hour) // VK tokens usually expire in 24 hours
		user.VKAccessToken = &accessToken
		user.VKTokenExpiresAt = &tokenExpiry
		if err := uc.userRepo.Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user token: %w", err)
		}
	}

	// Update online status
	if err := uc.userRepo.UpdateOnlineStatus(ctx, user.ID, true); err != nil {
		return nil, fmt.Errorf("failed to update online status: %w", err)
	}

	// Create session
	token, expiresAt, err := uc.createSession(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
		IsNewUser: isNewUser,
	}, nil
}

// AuthenticateVKTest authenticates user for testing without signature verification
func (uc *VKAuthUseCase) AuthenticateVKTest(ctx context.Context, params map[string]string, deviceInfo, ipAddress string) (*AuthResponse, error) {
	// Skip signature verification for test endpoint

	vkID := 0
	fmt.Sscanf(params["vk_user_id"], "%d", &vkID)
	if vkID == 0 {
		return nil, domain.ErrInvalidInput
	}

	// Try to get existing user
	user, err := uc.userRepo.GetByVKID(ctx, vkID)
	isNewUser := false

	if err == domain.ErrUserNotFound {
		// Create new user
		user, err = uc.createUserFromVK(ctx, vkID, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
		isNewUser = true
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update online status
	if err := uc.userRepo.UpdateOnlineStatus(ctx, user.ID, true); err != nil {
		return nil, fmt.Errorf("failed to update online status: %w", err)
	}

	// Create session
	token, expiresAt, err := uc.createSession(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
		IsNewUser: isNewUser,
	}, nil
}

// verifyVKSignature verifies VK Mini App launch params signature
func (uc *VKAuthUseCase) verifyVKSignature(params map[string]string) error {
	sign := params["sign"]
	if sign == "" {
		fmt.Println("DEBUG: No sign parameter found")
		return domain.ErrInvalidVKSignature
	}

	// Create query string from params (excluding sign and vk_access_token_settings)
	// vk_access_token_settings is added by VK after signature generation
	var keys []string
	for k := range params {
		if k != "sign" && k != "vk_access_token_settings" && strings.HasPrefix(k, "vk_") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var queryString strings.Builder
	for i, k := range keys {
		if i > 0 {
			queryString.WriteString("&")
		}
		queryString.WriteString(k)
		queryString.WriteString("=")
		// VK doesn't use URL encoding for signature verification
		queryString.WriteString(params[k])
	}

	fmt.Printf("DEBUG: Query string: %s\n", queryString.String())
	fmt.Printf("DEBUG: VK Secret length: %d\n", len(uc.vkSecret))

	// Calculate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(uc.vkSecret))
	h.Write([]byte(queryString.String()))
	calculatedSign := base64.URLEncoding.EncodeToString(h.Sum(nil))
	calculatedSign = strings.TrimRight(calculatedSign, "=")
	calculatedSign = strings.ReplaceAll(calculatedSign, "+", "-")
	calculatedSign = strings.ReplaceAll(calculatedSign, "/", "_")

	fmt.Printf("DEBUG: Received sign: %s\n", sign)
	fmt.Printf("DEBUG: Calculated sign: %s\n", calculatedSign)

	if sign != calculatedSign {
		return domain.ErrInvalidVKSignature
	}

	return nil
}

// createUserFromVK creates a new user from VK params
func (uc *VKAuthUseCase) createUserFromVK(ctx context.Context, vkID int, params map[string]string) (*domain.User, error) {
	// Parse gender from params
	gender := domain.GenderMale
	if g, ok := params["gender"]; ok {
		if g == "female" {
			gender = domain.GenderFemale
		}
	}

	// Parse birth_date from params
	birthDate := time.Now().AddDate(-20, 0, 0) // Default
	if bd, ok := params["birth_date"]; ok {
		if parsed, err := time.Parse("2006-01-02", bd); err == nil {
			birthDate = parsed
		}
	}

	user := &domain.User{
		VKID:       vkID,
		Gender:     gender,
		BirthDate:  birthDate,
		IsVerified: false,
		IsOnline:   true,
    
// createUserFromVKInfo creates a new user from VK API data
func (uc *VKAuthUseCase) createUserFromVKInfo(ctx context.Context, vkInfo *vkapi.VKUserInfo, accessToken string) (*domain.User, error) {
	// Parse gender
	gender := domain.GenderMale
	if vkInfo.Sex == 1 {
		gender = domain.GenderFemale
	}

	// Parse birthdate (VK format: DD.MM.YYYY or DD.MM)
	birthDate := time.Now().AddDate(-20, 0, 0) // default 20 years old
	if vkInfo.BirthDate != "" {
		// Try to parse full date
		if t, err := time.Parse("2.1.2006", vkInfo.BirthDate); err == nil {
			birthDate = t
		} else if t, err := time.Parse("02.01.2006", vkInfo.BirthDate); err == nil {
			birthDate = t
		}
	}

	tokenExpiry := time.Now().Add(24 * time.Hour)

	user := &domain.User{
		VKID:             vkInfo.ID,
		VKAccessToken:    &accessToken,
		VKTokenExpiresAt: &tokenExpiry,
		Gender:           gender,
		BirthDate:        birthDate,
		IsVerified:       false,
		IsOnline:         true,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Auto-create profile for new user
	displayName := params["first_name"]
	if lastName, ok := params["last_name"]; ok && lastName != "" {
		displayName = displayName + " " + lastName
	}

	profile := &domain.Profile{
		UserID:      user.ID,
		DisplayName: displayName,
	}

	if err := uc.profileRepo.Create(ctx, profile); err != nil {
		// Don't fail user creation if profile creation fails
		fmt.Printf("Warning: failed to create profile for user %d: %v\n", user.ID, err)
	}

	return user, nil
}

// createSession creates a new session and returns JWT token
func (uc *VKAuthUseCase) createSession(ctx context.Context, userID int, deviceInfo, ipAddress string) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * 7 * time.Hour) // 7 days

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	// Create session in DB
	session := &domain.Session{
		UserID:     userID,
		Token:      uc.hashToken(tokenString),
		DeviceInfo: &deviceInfo,
		IPAddress:  &ipAddress,
		ExpiresAt:  expiresAt,
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// VerifyToken verifies JWT token and returns user ID
func (uc *VKAuthUseCase) VerifyToken(ctx context.Context, tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return []byte(uc.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return 0, domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, domain.ErrInvalidToken
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, domain.ErrInvalidToken
	}

	// Verify session exists
	hashedToken := uc.hashToken(tokenString)
	session, err := uc.sessionRepo.GetByToken(ctx, hashedToken)
	if err != nil {
		return 0, domain.ErrSessionNotFound
	}

	if session.IsExpired() {
		return 0, domain.ErrSessionExpired
	}

	return int(userID), nil
}

// Logout deletes user session
func (uc *VKAuthUseCase) Logout(ctx context.Context, tokenString string) error {
	hashedToken := uc.hashToken(tokenString)
	return uc.sessionRepo.DeleteByToken(ctx, hashedToken)
}

// hashToken creates SHA256 hash of token for storage
func (uc *VKAuthUseCase) hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}
