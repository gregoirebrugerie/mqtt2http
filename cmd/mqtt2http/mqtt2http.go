package main

import (
	"mqtt2http/hooks"
	"mqtt2http/lib"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	mqtt "github.com/mochi-co/mqtt/v2"
	"github.com/mochi-co/mqtt/v2/listeners"
)

func main() {
	var err error

	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create the new MQTT Server.
	server := mqtt.New(nil)

	// Create HTTP Client
	err = godotenv.Load()
	if err != nil {
		server.Log.Warn().Err(err).Msg("Failed to read .env file")
	}

	tcpAddr := getEnv("MQTT2HTTP_LISTEN_ADDRESS", ":1883")
	authorizeURL := getEnv("MQTT2HTTP_AUTHORIZE_URL", "http://example.com")
	publishURL := getEnv("MQTT2HTTP_PUBLISH_URL", "http://example.com/{topic}")
	contentType := getEnv("MQTT2HTTP_CONTENT_TYPE", "application/octet-stream")

	client := &lib.Client{
		Server:       server,
		AuthorizeURL: authorizeURL,
		PublishURL:   publishURL,
		ContentType:  contentType,
	}

	// Setup auth hook
	authHook := &hooks.AuthHook{Client: client}
	err = server.AddHook(authHook, nil)
	if err != nil {
		server.Log.Error().Err(err).Msg("Failed to add auth hook")
	}

	// Setup publish hook
	publishHook := &hooks.PublishHook{Client: client}
	err = server.AddHook(publishHook, map[string]any{})
	if err != nil {
		server.Log.Error().Err(err).Msg("Failed to add publish hook")
	}

	// Create a TCP listener on a standard port.
	tcp := listeners.NewTCP("t1", tcpAddr, nil)
	err = server.AddListener(tcp)
	if err != nil {
		server.Log.Error().Err(err).Msg("Failed to add TCP listener")
	}

	// Start
	err = server.Serve()
	if err != nil {
		server.Log.Error().Err(err).Msg("Failed to start server")
	}

	// Handle signals
	go func() {
		sig := <-sigs
		server.Log.Info().Msg(sig.String())
		done <- true
	}()

	server.Log.Info().Msg("awaiting signal")
	<-done
	server.Log.Info().Msg("exiting")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
