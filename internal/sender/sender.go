package sender

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"mts.teta.mailsender/internal/mail"
	"mts.teta.mailsender/internal/model"
	"mts.teta.mailsender/internal/store"
)

type Sender struct {
	queue       store.MailingQueue
	logger      *zap.SugaredLogger
	running     chan int
	close       chan int
	stopped     bool
	stopDone    chan int
	stoppedLock sync.RWMutex
	mail        mail.Client
}

func New(
	queue store.MailingQueue,
	logger *zap.SugaredLogger,
	mail mail.Client,

) *Sender {
	sender := &Sender{
		queue:    queue,
		logger:   logger,
		running:  make(chan int, 1),
		close:    make(chan int),
		stopDone: make(chan int),
		mail:     mail,
	}

	sender.running <- 1

	return sender
}

// If server sleep than wake up server.
// Call from another go routine.
// Threadsafe.
func (s *Sender) Up() {
	s.logger.Info("start wake up...")
	select {
	case s.running <- 1:
	default:
	}
}

// Read data from queue. If queue is empty than sleep.
// If server is closed than do nothing.
func (s *Sender) Serve() {
	s.stoppedLock.RLock()
	stopped := s.stopped
	if stopped {
		s.stoppedLock.RUnlock()
		return
	}
	s.stoppedLock.RUnlock()

	for {
		select {
		case <-s.close:
			s.logger.Info("stop...")
			s.stopDone <- 1
			return
		case <-s.running:
			s.logger.Info("wake up...")
			for {
				success := s.send()
				if !success || s.stopped {
					// Wait wake up
					break
				}
			}
		}
	}
}

// Stopping serve and close channel.
func (s *Sender) Stop() {
	s.logger.Info("start stopping...")
	s.stoppedLock.Lock()
	s.stopped = true
	s.stoppedLock.Unlock()
	s.close <- 1
	//wait stop serving
	<-s.stopDone
	close(s.running)
	close(s.close)
	close(s.stopDone)
	s.logger.Info("stop for always...")
}

func (s *Sender) send() (result bool) {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	entry, ok, err := s.queue.Get(ctx)
	if err != nil {
		// do nothing. resend
		s.logger.Error(err)
		return
	}

	if !ok {
		return
	}

	result = true

	err = s.pushMail(entry)
	for i, _ := range entry.Receivers {
		entry.Receivers[i].IsSended = true
	}
	if err != nil {
		for _, p := range err.(*mail.ErrSending).NotReceivePosition {
			entry.Receivers[p].IsSended = false
		}
		s.error(entry, err)
		return
	}

	entry.Status = model.StatusMailingDone
	_, err = s.queue.Save(ctx, entry)
	if err != nil {
		// do nothing. resend
		s.logger.Error(err)
		return
	}

	return
}

func (s *Sender) error(entry model.QueueEntry, err error) {
	s.logger.Error(err)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	entry.Attempts++
	if entry.Attempts < model.AttemptsCount { // add to queue for next attempt
		entry.Status = model.StatusMailingPending
	} else {
		entry.Status = model.StatusMailingError
	}
	_, err = s.queue.Save(ctx, entry)
	if err != nil {
		s.logger.Error(err)
		return
	}
}

func (s *Sender) pushMail(entry model.QueueEntry) error {
	return s.mail.Send(&entry)
}
