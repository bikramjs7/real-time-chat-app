package email

import (
	"encoding/json"
	"fmt"

	"workerservice/config"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type EmailSender struct {
	logger *zap.Logger
}

type RegistrationEmail struct {
	UserID  string `json:"userID"`
	EmailID string `json:"emailID"`
}

func NewEmailSender(logger *zap.Logger) *EmailSender {
	return &EmailSender{
		logger: logger,
	}
}

func (s *EmailSender) SendRegistrationEmail(data []byte) error {
	var email RegistrationEmail
	if err := json.Unmarshal(data, &email); err != nil {
		return fmt.Errorf("failed to unmarshal email data: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", config.FromEmail)
	m.SetHeader("To", email.EmailID)
	m.SetHeader("Subject", "Welcome to Our Service")
	m.SetBody("text/html", fmt.Sprintf(`
        <p>Dear User,</p>
        <p>Thank you for registering with our service. Your account has been created successfully.</p>
        <p>User ID: %s</p>
        <p>Best regards,<br>Bikram</p>
    `, email.UserID))

	d := gomail.NewDialer(
		config.SMTPHost,
		config.SMTPPort,
		config.SMTPUser,
		config.SMTPPassword,
	)

	s.logger.Info("Sending registration email", zap.String("email", email.EmailID))
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Registration email sent successfully", zap.String("email", email.EmailID))
	return nil
}
