package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gdugdh24/mpit2026-backend/internal/domain"
	"github.com/gdugdh24/mpit2026-backend/internal/usecase/profile"
	"github.com/gin-gonic/gin"
)

type ProfileHandler struct {
	profileUseCase *profile.ProfileUseCase
}

func NewProfileHandler(profileUseCase *profile.ProfileUseCase) *ProfileHandler {
	return &ProfileHandler{
		profileUseCase: profileUseCase,
	}
}

// GetMyProfile handles GET /profile/me
// @Summary Get my profile
// @Description Get current user's profile
// @Tags profile
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.Profile
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /profile/me [get]
func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	profile, err := h.profileUseCase.GetMyProfile(c.Request.Context(), userID.(int))
	if err != nil {
		if err == domain.ErrProfileNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "profile not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to get profile",
		})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateMyProfile handles PUT /profile/me
// @Summary Update my profile
// @Description Update current user's profile
// @Tags profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body profile.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} domain.Profile
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /profile/me [put]
func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	var req profile.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	updatedProfile, err := h.profileUseCase.UpdateProfile(c.Request.Context(), userID.(int), &req)
	if err != nil {
		if err == domain.ErrProfileNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "profile not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to update profile",
		})
		return
	}

	c.JSON(http.StatusOK, updatedProfile)
}

// CompleteOnboarding handles POST /profile/complete-onboarding
// @Summary Complete onboarding
// @Description Create profile and complete onboarding
// @Tags profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body profile.CreateProfileRequest true "Profile creation data"
// @Success 201 {object} domain.Profile
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /profile/complete-onboarding [post]
func (h *ProfileHandler) CompleteOnboarding(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	var req profile.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	newProfile, err := h.profileUseCase.CreateProfile(c.Request.Context(), userID.(int), &req)
	if err != nil {
		if err == domain.ErrProfileAlreadyExists {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error: "profile already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to create profile",
		})
		return
	}

	c.JSON(http.StatusCreated, newProfile)
}

// GetProfileByUserID handles GET /profile/:user_id
// @Summary Get user profile
// @Description Get another user's profile by user ID
// @Tags profile
// @Security BearerAuth
// @Produce json
// @Param user_id path int true "User ID"
// @Success 200 {object} profile.ProfileResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /profile/{user_id} [get]
func (h *ProfileHandler) GetProfileByUserID(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	targetUserIDStr := c.Param("user_id")
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid user_id",
		})
		return
	}

	currentUID := currentUserID.(int)
	profileResp, err := h.profileUseCase.GetProfileByUserID(c.Request.Context(), targetUserID, &currentUID)
	if err != nil {
		if err == domain.ErrProfileNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "profile not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to get profile",
		})
		return
	}

	c.JSON(http.StatusOK, profileResp)
}

// GenerateBio handles POST /profile/generate-bio
// @Summary Generate bio with AI
// @Description Generate 3 creative bios
// @Tags profile
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body profile.GenerateBioRequest true "Bio generation data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /profile/generate-bio [post]
func (h *ProfileHandler) GenerateBio(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	var req profile.GenerateBioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid request body",
		})
		return
	}

	bios, err := h.profileUseCase.GenerateBio(c.Request.Context(), &req)
	if err != nil {
		fmt.Printf("Error generating bio: %v\n", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "failed to generate bio",
		})
		return
	}

	c.JSON(http.StatusOK, bios)
}
