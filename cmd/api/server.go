package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     log.New(app.logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// The shutdownError channel will be used to receive any errors returned by
	// the graceful Shutdown() function.
	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		// Use signal.Notify() to listen for incoming SIGINT and SIGTERM signals
		// and relay them to the quit channel. Any other signals will not be
		// caught by signal.Notify() and will retain their default behavior.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Read the signal from the quit channel.
		// NOTE: This code will block until a signal is received.
		s := <-quit

		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		// Create a context with a 5-second timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Call Shutdown() on the server, but we only send on the shutdownError
		// channel if it returns an error.
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		// Log a message to say that we're waiting for any background goroutines
		// to complete their tasks.
		app.logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		// Call Wait() to block until our WaitGroup counter is zero ---
		// essentially blocking until the background goroutines have finished.
		// Then we return nil on the shutdownError channel to indicate that the
		// shutdown completed without any issues.
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// Calling Shutdown() on our server will cause ListenAndServe() to
	// immediately return a http.ErrServerClosed error. So if we see this error,
	// it is actually a good thing and an indication that the graceful shutdown
	// has started. So we check specifically for this, only returning the error
	// if it is NOT http.ErrServerClosed.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, we wait to receive the return value from Shutdown() on the
	// shutdownError channel. If return value is an error, we know that there
	// was a problem with the graceful shutdown and we return the error.
	// NOTE: This code will block until an error or nil is received.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// At this point we know that the graceful shutdown completed successfully
	// and we log a "stopped server" message.
	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
