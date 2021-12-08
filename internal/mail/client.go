package mail

import (
	"crypto/tls"
	"fmt"

	"github.com/caarlos0/env"
	"go.uber.org/zap"
	"mts.teta.mailsender/internal/model"

	gomail "gopkg.in/mail.v2"
)

type Client interface {
	Send(entry *model.Mailing) error
}

func New(cfg *MailConfig, logger *zap.SugaredLogger) Client {
	return &internalClient{
		logger: logger,
		cfg:    cfg,
	}
}

type MailConfig struct {
	Email       string `env:"MAIL_EMAIL"`
	Description string `env:"MAIL_DESCRIPTION"`

	Host     string `env:"MAIL_HOST"`
	Port     int    `env:"MAIL_PORT"`
	Login    string `env:"MAIL_LOGIN"`
	Password string `env:"MAIL_PASSWORD"`
}

func ReadConfig() (*MailConfig, error) {
	var cfg MailConfig
	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

type ErrSending struct {
	Addrs              []string
	NotReceivePosition []int
}

func (err *ErrSending) Error() string {
	return fmt.Sprintf("Mailing not deliver to %s", err.Addrs)
}

type internalClient struct {
	Client
	logger *zap.SugaredLogger
	cfg    *MailConfig
}

func (c *internalClient) Send(mailing *model.Mailing) error {
	var sendingErr ErrSending
	smtp := gomail.NewDialer(c.cfg.Host, c.cfg.Port, c.cfg.Login, c.cfg.Password)
	smtp.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	for p, receiver := range mailing.Receivers {
		if receiver.IsSended {
			continue
		}

		m := gomail.NewMessage()
		m.SetHeader("From", c.cfg.Email, c.cfg.Description)
		m.SetAddressHeader("To", receiver.Addr, receiver.Addr)
		m.SetHeader("Subject", mailing.Subject)
		m.SetBody("text/html", mailing.Text)
		err := smtp.DialAndSend(m)
		if err != nil {
			sendingErr.Addrs = append(sendingErr.Addrs, fmt.Sprintf("%s (%s)", receiver.Addr, err.Error()))
			sendingErr.NotReceivePosition = append(sendingErr.NotReceivePosition, p)
		}
	}

	if len(sendingErr.Addrs) > 0 {
		return &sendingErr
	}

	return nil
}
