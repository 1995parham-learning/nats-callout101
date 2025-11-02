package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/1995parham-learning/nats-callout101/internal/authenticator"
)

const (
	natsURL  = "nats://<user>:<password>@localhost:4222"
	nkeySeed = "SAANDLKMXL6CUS3CP52WIXBEDN6YJ545GDKC65U5JZPPV6WH6ESWUA6YAI"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	auth, err := authenticator.New(logger, authenticator.Config{
		URL:      natsURL,
		NkeySeed: nkeySeed,
		Users: map[string]authenticator.User{
			"admin": {
				Password:   "admin",
				Account:    "APP",
				Privileged: true,
			},
			"test-1": {
				Password:   "test-1",
				Account:    "$G",
				Privileged: true,
			},
			"test-2": {
				Password:   "test-2",
				Account:    "$G",
				Privileged: false,
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create authenticator service %s", err)
	}

	err = auth.Start()
	if err != nil {
		log.Fatalf("authenticator service subscription failed %s", err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Received shutdown signal, exiting...")

	err = auth.Stop()
	if err != nil {
		log.Fatalf("authenticator service unsubscription failed %s", err)
	}

	log.Println("Bye bye!")
}
