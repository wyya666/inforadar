package application

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"inforadar/internal/config"
	"inforadar/internal/constants"
	handlerinterface "inforadar/internal/handler/interface"
	radarhandler "inforadar/internal/handler/radar"
	userhandler "inforadar/internal/handler/user"
	"inforadar/internal/middleware"
	middlewareinterface "inforadar/internal/middleware/interface"
	repositoryinterface "inforadar/internal/repository/interface"
	userrepository "inforadar/internal/repository/user"
	"inforadar/internal/router"
	serviceinterface "inforadar/internal/service/interface"
	"inforadar/internal/service/mail"
	"inforadar/internal/service/provider"
	radarsvc "inforadar/internal/service/radar"
	userservice "inforadar/internal/service/user"
	cryptoutil "inforadar/internal/utils/crypto"
)

type Application struct {
	UserHandler  handlerinterface.UserHandler
	RadarHandler handlerinterface.RadarHandler
	server       *http.Server
	repository   repositoryinterface.UserStore
	radar        serviceinterface.RadarService
}

var (
	_ serviceinterface.UserService       = (*userservice.Service)(nil)
	_ repositoryinterface.UserRepository = (*userrepository.Repository)(nil)
	_ repositoryinterface.UserStore      = (*userrepository.Store)(nil)
	_ serviceinterface.Mailer            = (*mail.SMTPMailer)(nil)
	_ handlerinterface.UserHandler       = (*userhandler.Handler)(nil)
	_ middlewareinterface.Authenticator  = (*userservice.Service)(nil)
)

// New is the composition root for configuration, persistence, mail, use cases, and HTTP.
func New(ctx context.Context) (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	repository, err := userrepository.OpenMySQL(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	mailer, err := mail.NewSMTPMailer(cfg.SMTP, cfg.Server.FrontendURL)
	if err != nil {
		repository.Close()
		return nil, err
	}
	service, err := userservice.NewService(repository, mailer, cfg.Auth.JWTSecret)
	if err != nil {
		repository.Close()
		return nil, err
	}
	box, err := cryptoutil.NewSecretBox(cfg.Radar.EncryptionKey)
	if err != nil {
		repository.Close()
		return nil, err
	}
	providers := provider.New()
	radar := radarsvc.New(repository.Radar(), providers, providers, box)
	limits := middleware.NewLimiter()
	app := &Application{UserHandler: userhandler.New(service, cfg, limits), RadarHandler: radarhandler.New(radar), repository: repository, radar: radar}
	app.server = &http.Server{
		Addr: cfg.Server.Address, Handler: router.New(cfg, app.UserHandler, service, limits, app.RadarHandler),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	return app, nil
}

func (a *Application) Run(ctx context.Context) error {
	defer a.repository.Close()
	workerCtx, stopWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); a.scanLoop(workerCtx) }()
	defer func() { stopWorker(); <-workerDone }()
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- a.server.ListenAndServe() }()
	log.Printf("InfoRadar API listening on %s", a.server.Addr)
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.server.Shutdown(shutdownCtx)
	}
}

func (a *Application) scanLoop(ctx context.Context) {
	ticker := time.NewTicker(constants.RadarPollInterval)
	defer ticker.Stop()
	for {
		for i := 0; i < 100; i++ {
			if ctx.Err() != nil {
				return
			}
			claimed, err := a.radar.RunDue(ctx, time.Now().UTC())
			if err != nil {
				log.Printf("radar worker failed: %v", err)
				break
			}
			if !claimed {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-a.radar.Wakeups():
		}
	}
}
