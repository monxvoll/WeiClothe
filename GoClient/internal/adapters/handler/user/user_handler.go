package user

import (
	"net/http"
	"strings"
	"time"
	"weicloth/internal/core/domain"
	"weicloth/internal/core/services"

	"github.com/gin-gonic/gin"
)

type (
	HTTPHandler struct {
		userService *services.UserService
	}
)

func NewHTTPHandler(userService *services.UserService) *HTTPHandler {
	return &HTTPHandler{
		userService: userService,
	}
}

func (h *HTTPHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"message": "pong"})
}

// Register handles user registration
func (h *HTTPHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil { //Parse JSON and validate
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse date_birth
	dob, err := time.Parse("2006-01-02", req.DateBirth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_birth format"})
		return
	}

	input := domain.RegisterUserInput{ //Struct expected by the user service
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Nickname:  req.Nickname,
		Email:     req.Email,
		Password:  req.Password,
		DateBirth: dob,
		Gender:    req.Gender,
	}

	//Call use case
	if err := h.userService.RegisterUser(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

// Login handles user login
func (h *HTTPHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := domain.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}

	token, err := h.userService.LoginUser(c.Request.Context(), input)
	if err != nil {
		if strings.Contains(err.Error(), "401") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user logged in", "token": token})
}
