package controllers

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"

	"www.github.com/ic-ETITE-24/icetite-24-backend/internal/database"
	"www.github.com/ic-ETITE-24/icetite-24-backend/internal/models"
	"www.github.com/ic-ETITE-24/icetite-24-backend/internal/utils"
)

func CreateUser(c *fiber.Ctx) error {
	var createUser models.CreateUser

	if err := c.BodyParser(&createUser); err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"message": "Please send complete data"})
	}

	dob, _ := time.Parse("2006-01-02", createUser.DateOfBirth)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(createUser.Password), 10)

	user := models.User{
		FirstName:   createUser.FirstName,
		LastName:    createUser.LastName,
		Email:       createUser.Email,
		Password:    string(hashedPassword),
		Gender:      createUser.Gender,
		DateOfBirth: dob,
		Bio:         createUser.Bio,
		TeamId:      0,
		IsLeader:    false,
		IsApproved:  false,
		PhoneNumber: createUser.PhoneNumber,
		College:     createUser.College,
		Github:      createUser.Github,
	}

	if result := database.DB.Create(&user); result.Error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": result.Error.Error(),
		})
	}

	return c.Status(fiber.StatusOK).
		JSON(fiber.Map{"message": "Successfully created user", "user": user})
}

func ForgotPassword(c *fiber.Ctx) error {
	email := c.Params("email")

	var check models.User
	database.DB.Find(&check, "email = ?", email)
	if check.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(&fiber.Map{
			"Status": false,
			"Error":  "The email address given doesn't exist",
		})
	}

	payload := utils.TokenPayload{
		Email:   email,
		Role:    "",
		Version: 0,
	}

	resetToken, err := utils.CreateToken(
		time.Minute*2,
		payload,
		utils.REFRESH_TOKEN,
		viper.GetString("RESET_SECRET_KEY"),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"Status": false,
			"Error":  "Failed to create an JWT token",
		})
	}

	url := fmt.Sprintf("%s%s", viper.GetString("RESET_PASSWORD_URL"), resetToken)
	message := fmt.Sprintf("%s\n%s %s\n%s",
		"Click the link below to reset your password",
		url,
		"If this request was not sent by you please report to the concerned authorities",
		"This is an auto generated email.",
	)

	err = utils.SendMail("Password Reset", email, message)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"Status": false,
			"Error":  "Something went wrong while sending the email",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(&fiber.Map{
		"Status": true,
		"data":   resetToken,
	})
}

func ResetPassword(c *fiber.Ctx) error {
	token := c.Params("Token", "")

	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid token"})
	}

	type Password struct {
		Password     string `json:"password"`
		Confirm_pass string `json:"confirm_pass"`
	}

	Token, _ := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}

		return []byte(viper.GetString("RESET_SECRET_KEY")), nil
	})

	if decoded, ok := Token.Claims.(jwt.MapClaims); ok {
		if float64(time.Now().Unix()) > decoded["exp"].(float64) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"Error": "Token Expired",
			})
		}

		email := decoded["email"]
		var user models.User
		database.DB.Find(&user, "email = ?", email)
		if user.ID == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(&fiber.Map{
				"Error": "The email doesn't exist",
			})
		}

		req := new(Password)
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(&fiber.Map{
				"Error": "Error rose while parsing through the body",
			})
		}

		if req.Password != req.Confirm_pass {
			return c.Status(fiber.StatusBadRequest).JSON(&fiber.Map{
				"Error": "Password and confirm password are not the same",
			})
		}

		hashed_password, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
		user.Password = string(hashed_password)
		database.DB.Save(user)
		return c.Status(fiber.StatusAccepted).JSON(&fiber.Map{
			"Message": "The password has been updated",
		})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid Token"})
}

func SendOTP(c *fiber.Ctx) error {
	email := c.Params("email", "")

	if email == "" {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"message": "Please give a valid email"})
	}

	otp := rand.Intn(900000) + 100000

	var user models.User
	database.DB.Find(&user, "email = ?", email)

	if user.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User does not exist"})
	}

	if err := utils.SendMail("OTP", fmt.Sprintf("Your OTP is: %d", otp), email); err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"message": "Some error occurred", "error": err.Error()})
	}

	user.OTP = otp
	database.DB.Save(&user)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Otp sent to your Email", "OTP": otp})
}

func VerifyOTP(c *fiber.Ctx) error {
	var verifyRequest struct {
		Email string `json:"email"`
		OTP   int    `json:"otp"`
	}

	if err := c.BodyParser(&verifyRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"message": "Please pass in the correct data"})
	}

	if verifyRequest.Email == "" {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"message": "Please give a valid email"})
	}

	var user models.User
	database.DB.Find(&user, "email = ?", verifyRequest.Email)

	if user.ID == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User Not Found"})
	}

	if user.OTP != verifyRequest.OTP {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid OTP"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Verified OTP"})
}
