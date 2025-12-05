package handler

import (
	"fmt"
	"net/http"

	"github.com/gdugdh24/mpit2026-backend/internal/usecase/auth"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUseCase *auth.VKAuthUseCase
}

func NewAuthHandler(authUseCase *auth.VKAuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

// VKAuthRequest represents VK authentication request
type VKAuthRequest struct {
	VKParams    map[string]string `json:"vk_params" binding:"required"`
	AccessToken string            `json:"access_token" binding:"required"`
}

// AuthResponse is the response structure
type AuthResponse struct {
	Token     string      `json:"token"`
	ExpiresAt int64       `json:"expires_at"`
	User      interface{} `json:"user"`
	IsNewUser bool        `json:"is_new_user"`
}

// VKAuth handles VK Mini App authentication
// @Summary VK authentication
// @Description Authenticate user via VK Mini App launch params
// @Tags auth
// @Accept json
// @Produce json
// @Param request body VKAuthRequest true "VK launch params"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/vk [post]
func (h *AuthHandler) VKAuth(c *gin.Context) {
	var req VKAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	// Debug logging
	c.Writer.Header().Set("X-Debug-VK-Params", fmt.Sprintf("%+v", req.VKParams))

	deviceInfo := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	result, err := h.authUseCase.AuthenticateVK(c.Request.Context(), req.VKParams, req.AccessToken, deviceInfo, ipAddress)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "authentication failed"

		switch err.Error() {
		case "invalid VK signature":
			statusCode = http.StatusUnauthorized
			message = "invalid VK signature"
		case "invalid input":
			statusCode = http.StatusBadRequest
			message = "invalid VK parameters"
		}

		c.JSON(statusCode, ErrorResponse{
			Error: message,
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.Unix(),
		User:      result.User,
		IsNewUser: result.IsNewUser,
	})
}

// TestAuthRequest represents test authentication request
type TestAuthRequest struct {
	VKID      int    `json:"vk_id" binding:"required"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Gender    string `json:"gender" binding:"required,oneof=male female"`
	BirthDate string `json:"birth_date" binding:"required"` // Format: YYYY-MM-DD
}

// TestAuth creates a test user without VK signature validation (for development/testing only)
// @Summary Test authentication
// @Description Create test user without VK validation (dev only)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body TestAuthRequest true "Test user data"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/test [post]
func (h *AuthHandler) TestAuth(c *gin.Context) {
	var req TestAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	// Create mock VK params
	vkParams := map[string]string{
		"vk_user_id":  fmt.Sprintf("%d", req.VKID),
		"vk_app_id":   "test",
		"vk_platform": "test",
		"vk_ts":       "0",
		"sign":        "test_bypass",
		"first_name":  req.FirstName,
		"last_name":   req.LastName,
		"gender":      req.Gender,
		"birth_date":  req.BirthDate,
	}

	deviceInfo := "test-client"
	ipAddress := c.ClientIP()

	result, err := h.authUseCase.AuthenticateVKTest(c.Request.Context(), vkParams, deviceInfo, ipAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: fmt.Sprintf("test auth failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.Unix(),
		User:      result.User,
		IsNewUser: result.IsNewUser,
	})
}

// Logout handles user logout
// @Summary Logout
// @Description Logout user and invalidate session
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "missing authorization token",
		})
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if err := h.authUseCase.Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "logout failed",
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "logged out successfully",
	})
}

// Me returns current user info
// @Summary Get current user
// @Description Get authenticated user information
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.User
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
	})
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents success response
type SuccessResponse struct {
	Message string `json:"message"`
}
