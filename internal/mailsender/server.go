package mailsender

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"mts.teta.mailsender/docs"
	"mts.teta.mailsender/internal/config"
	"mts.teta.mailsender/internal/model"
	"mts.teta.mailsender/internal/sender"
	"mts.teta.mailsender/internal/store"
	"mts.teta.mailsender/pkg/logger"
)

const (
	pageSize = 20
)

type Server struct {
	server *http.Server
	config *config.Config
	logger *zap.SugaredLogger
	queue  store.MailingQueue
	router chi.Router
	sender *sender.Sender
}

func New(
	config *config.Config,
	logger *zap.SugaredLogger,
	queue store.MailingQueue,
	sender *sender.Sender,
) *Server {
	server := Server{
		config: config,
		logger: logger,
		router: chi.NewRouter(),
		queue:  queue,
		sender: sender,
	}

	docs.SwaggerInfo.Host = config.Addr
	docs.SwaggerInfo.BasePath = "/"

	server.server = &http.Server{
		Addr:    config.Addr,
		Handler: server.router,
	}
	server.configureRouters()

	return &server
}

// Start server.
func (s *Server) Start() error {
	s.logger.Info("Server started at: ", s.config.Addr)
	return s.server.ListenAndServe()
}

// Stop server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Server stopping...")
	return s.server.Shutdown(ctx)
}

func (s *Server) configureRouters() {
	r := s.router
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.Logger(s.logger.Desugar()))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/mailing", func(r chi.Router) {
		r.Post("/", s.Create)
		r.Get("/", s.List)
		r.Get("/{mailing_id}", s.Get)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("%s/swagger/doc.json", s.config.Addr)),
	))
}

//MailingCreate godoc
//@Summary Add mailing in queue.
//@Accept json
//@Param template body model.Mailing true "mailing info"
//@Success 200 {string} string "id"
//@Success 400 {string} string "Not valid mailing info"
//@Success 500 {string} string "DB error"
//@Router /mailing [post]
func (s *Server) Create(w http.ResponseWriter, r *http.Request) {
	var mailing model.Mailing

	data, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error(err)
		http.Error(w, errors.Wrap(err, "read request error").Error(), http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(data, &mailing)
	if err != nil {
		s.logger.Error(err)
		http.Error(w, errors.Wrapf(err, "parse request error with data: %v", string(data)).Error(), http.StatusBadRequest)
		return
	}

	mailing.Status = model.StatusMailingPending // set status if client set other value
	id, err := s.queue.Save(r.Context(), mailing)
	if err != nil {
		s.logger.Error(err)
		http.Error(w, errors.Wrap(err, "save in queue error").Error(), http.StatusInternalServerError)
		return
	}

	s.sender.Up()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(id))
}

type Info struct {
	Id     string `json:"id"`     // mailing id
	Status string `json:"status"` // mailing status
}

type MailingList struct {
	Total int64  `json:"total"` // total pages
	Data  []Info `json:"data"`
}

//MailingList godoc
//@Summary Get mailing list.
//@Param p query int false "page number, start from 1"
//@Success 200 {object} mailsender.MailingList "Mailings list"
//@Success 400 {string} string "page error"
//@Success 500 {string} string "DB error"
//@Router /mailing [get]
func (s *Server) List(w http.ResponseWriter, r *http.Request) {
	startPosition, err := strconv.ParseInt(chi.URLParam(r, "p"), 10, 64)
	if err != nil {
		startPosition = 0
	} else {
		startPosition = startPosition - 1
	}

	if startPosition < 0 {
		http.Error(w, "Wrong page", http.StatusBadRequest)
		return
	}

	lists, err := s.queue.FindAll(r.Context(), startPosition*pageSize, pageSize)
	if err != nil {
		http.Error(w, errors.Wrap(err, "Mailing FindAll error").Error(), http.StatusInternalServerError)
		return
	}

	count, _ := s.queue.Count(r.Context())

	responseData := &MailingList{
		Data:  make([]Info, len(lists)),
		Total: 1 + count/pageSize,
	}
	for i, list := range lists {
		responseData.Data[i] = mailingListView(list)
	}

	body, err := json.Marshal(responseData)
	if err != nil {
		http.Error(w, errors.Wrap(err, "Response marshal error").Error(), http.StatusInternalServerError)
		return
	}

	w.Write(body)
}

//MailingGet godoc
//@Summary Get mailing by id.
//@Param id path string true "mailing id"
//@Success 200 {object} model.Mailing  "mailing"
//@Success 400 {string} string "Not found mailing with id"
//@Success 500 {string} string "DB error"
//@Router /mailing/{mailing_id} [get]
func (s *Server) Get(w http.ResponseWriter, r *http.Request) {
	mailId := chi.URLParam(r, "mailing_id")
	if mailId == "" {
		s.logger.Error(errors.New("Not set mail id"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mail, err := s.queue.Find(r.Context(), model.EntryId(mailId))

	var errNotFound store.ErrNotFound
	if errors.Is(err, &errNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(mail)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(body)
}

func mailingListView(list model.Mailing) Info {
	return Info{
		Id:     string(list.Id),
		Status: list.StatusToString(),
	}
}
