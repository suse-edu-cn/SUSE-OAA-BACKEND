package service

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"time"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	Host     string        `json:"host"`
	Port     int           `json:"port"`
	User     string        `json:"user"`
	Pass     string        `json:"pass"`
	Expire   int           `json:"expire"`
	Cooldown time.Duration `json:"cooldown"`
}

func NewEmailService(host string, port int, user string, pass string, expire int, cooldown int) EmailService {
	return EmailService{
		Host:     host,
		Port:     port,
		User:     user,
		Pass:     pass,
		Expire:   expire,
		Cooldown: time.Duration(cooldown) * time.Minute,
	}
}
func (e *EmailService) NewVerificationCode(length uint) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := make([]byte, length)
	for i := uint(0); i < length; i++ {
		code[i] = byte('0' + r.Intn(10))
	}
	return string(code)
}

func (e *EmailService) SendVerificationCode(email string, code string) error {
	m := gomail.NewMessage()

	m.SetHeader("From", e.User)

	m.SetHeader("To", email)

	m.SetHeader("Subject", "SUSE-OAA 账号安全验证码")

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>您好！</h2>
			<p>您正在进行邮箱验证，您的专属验证码是：</p>
			<p><strong style="color: #4CAF50; font-size: 28px; letter-spacing: 2px;">%s</strong></p>
			<p style="color: #666; font-size: 14px;">该验证码在 <b>%d 分钟</b> 内有效。如果这不是您的操作，请忽略此邮件。</p>
			<hr style="border: none; border-top: 1px solid #eee; margin-top: 20px;" />
			<p style="color: #999; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
		</div>
	`, code, e.Expire)

	m.SetBody("text/html", body)

	d := gomail.NewDialer(e.Host, e.Port, e.User, e.Pass)

	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("邮件发送失败: %w", err)
	}

	return nil
}

func (e *EmailService) GetExpireTime() time.Duration {
	return time.Duration(e.Expire) * time.Minute
}
