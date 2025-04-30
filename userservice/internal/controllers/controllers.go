package controllers

import (
	"time"
	"userservice/database"
	"userservice/grpcclient"
	"userservice/internal/jwtpkg"
	"userservice/internal/kafka"
	"userservice/internal/models"
	"userservice/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var grpcClient *grpcclient.Client
var logger *zap.Logger
var producer *kafka.Producer
var blacklist *utils.Blacklist

func InitAuthController(grpcAddress string, log *zap.Logger, p *kafka.Producer, b *utils.Blacklist) {
	blacklist = b
	producer = p
	var err error
	grpcClient, err = grpcclient.NewClient(grpcAddress)
	if err != nil {
		log.Fatal("Failed to create gRPC client", zap.Error(err))
	}
	logger = log
}

func Register(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot hash password"})
	}
	user.Password = string(hashedPassword)

	// Insert user into database
	if err := database.CreateUser(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot register user"})
	}

	emailEvent := map[string]string{
		"userID":  user.ID,
		"emailID": user.Email,
	}
	if err := producer.SendMessageToEmailTopic(user.Email, emailEvent); err != nil {
		logger.Error("Failed to send email event ", zap.Error(err), zap.String("userId", user.ID))
		logEvent := map[string]interface{}{
			"message": "Failed to send email event",
			"userId":  user.ID,
			"error":   err.Error(),
			"time":    time.Now().Format(time.RFC3339),
		}
		if logErr := producer.SendMessageToLogsTopic(user.ID, logEvent); logErr != nil {
			logger.Error("Failed to send log event", zap.Error(logErr))
		}
	}

	registrationLog := map[string]interface{}{
		"message": "User registered successfully",
		"userId":  user.ID,
		"time":    time.Now().Format(time.RFC3339),
	}
	if logErr := producer.SendMessageToLogsTopic(user.ID, registrationLog); logErr != nil {
		logger.Error("Failed to send registration log", zap.Error(logErr))
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User registered successfully"})
}

func Login(c *fiber.Ctx) error {
	var input models.User
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var user models.User
	err := database.DB.QueryRow("SELECT id, email, password, role FROM users WHERE id = ?", input.ID).Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid email or password"})
	}
	if user.Role != input.Role || user.Email != input.Email || user.ID != input.ID {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}
	// Generate JWT token
	token, err := jwtpkg.GenerateJWT(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot generate token"})
	}

	loginSuccessLog := map[string]interface{}{
		"message": "User logged in successfully",
		"userId":  input.ID,
		"time":    time.Now().Format(time.RFC3339),
	}
	if logErr := producer.SendMessageToLogsTopic(input.ID, loginSuccessLog); logErr != nil {
		logger.Error("Failed to send login success log", zap.Error(logErr))
	}

	return c.JSON(fiber.Map{"token": token})
}

func GetProfile(c *fiber.Ctx) error {
	userId := c.Query("id")
	if userId == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User ID is required"})
	}

	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	token = token[7:]
	is, _ := blacklist.Get(token)
	if is {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	user, err := database.GetUserById(userId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user information"})
	}
	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

func Logout(c *fiber.Ctx) error {
	var req models.LogoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	token = token[7:] // Remove Bearer prefix
	is, _ := blacklist.Get(token)
	if is {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	claims := c.Locals("user").(*jwt.Token).Claims.(jwt.MapClaims)
	userId := claims["id"].(string)
	if userId != req.UserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token and userid mismatch"})
	}

	expiration := time.Until(time.Unix(int64(claims["exp"].(float64)), 0))
	err := blacklist.Set(token, expiration)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to blacklist token"})
	}
	res, err := grpcClient.Logout(userId)
	if err != nil {
		logger.Error("Failed to call Logout RPC", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to logout"})
	}

	if !res.GetSuccess() {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Logout RPC call unsuccessful"})
	}

	logEvent := map[string]interface{}{
		"message": "User logged out successfully",
		"userId":  userId,
		"time":    time.Now().Format(time.RFC3339),
	}
	if logErr := producer.SendMessageToLogsTopic(userId, logEvent); logErr != nil {
		logger.Error("Failed to send logout success log", zap.Error(logErr))
	}

	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

func GetUsersWithUserRole(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	token = token[7:]
	is, _ := blacklist.Get(token)
	if is {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	users, err := database.GetUsersWithRole("user")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch users"})
	}
	return c.JSON(users)
}

func GetUserByID(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	token = token[7:]
	is, _ := blacklist.Get(token)
	if is {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID := c.Params("userID")
	//	logger.Info("UserOnly", zap.String("userIdQuery", userID))
	claims := c.Locals("user").(*jwt.Token).Claims.(jwt.MapClaims)
	userId := claims["id"].(string)
	if userID != userId {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token and userID mismatch"})
	}
	user, err := database.GetUserById(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch user"})
	}

	return c.JSON(user)
}
