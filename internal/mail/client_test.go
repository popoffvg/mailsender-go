//go:build integration
// +build integration

package mail

import (
	"testing"

	"go.uber.org/zap"
	"mts.teta.mailsender/internal/model"
)

func TestMailing(t *testing.T) {
	logger := zap.NewNop().Sugar()
	cfg := &MailConfig{
		Host:     "smtp.gmail.com",
		Port:     587,
		Login:    "mts.cloud.interview",
		Password: "zsqthfhiqsxlookz",
		Email:    "mts.cloud.interview@gmail.com",
	}
	client := New(cfg, logger)
	err := client.Send(&model.QueueEntry{
		Subject: "Test",
		Receivers: []model.Receiver{
			{Addr: "popoffvg@gmail.com"},
			{Addr: "popoffvg@gmail.com"},
		},
		Text: "Test mail",
	})
	if err != nil {
		t.Fatal(err)
	}
}
