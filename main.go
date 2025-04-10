package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

const (
	natsURL  = "nats://auth:auth@127.0.0.1:4222"
	nkeySeed = "SAANDLKMXL6CUS3CP52WIXBEDN6YJ545GDKC65U5JZPPV6WH6ESWUA6YAI"
)

func handler(msg *nats.Msg) {
	log.Printf("Received authentication request on subject: %s, reply: %s", msg.Subject, msg.Reply)

	rc, err := jwt.DecodeAuthorizationRequestClaims(string(msg.Data))
	if err != nil {
		log.Println("Error", err)
	}

	userId := rc.ConnectOptions.Username
	log.Printf("username: %s password: %s wants to connect", rc.ConnectOptions.Username, rc.ConnectOptions.Password)

	claims := jwt.NewUserClaims(rc.UserNkey)
	claims.Audience = "APP"
	claims.Name = userId
	claims.Permissions = jwt.Permissions{
		Pub: jwt.Permission{
			Allow: jwt.StringList{
				"$JS.API.INFO", // General JS Info

				// Chat permisions
				fmt.Sprintf("chat.*.%s", userId),            // Publishing chat messages for this user id
				"$JS.API.STREAM.INFO.chat_messages",         // Getting info on chat_messages stream
				"$JS.API.CONSUMER.CREATE.chat_messages.>",   // Creating consumers on chat_messages stream
				"$JS.API.CONSUMER.MSG.NEXT.chat_messages.>", // Creating consumers on chat_messages stream

				// Workspace permissions
				"$JS.API.DIRECT.GET.KV_chat_workspace.>",        // Gets from workspace KV
				"$JS.API.STREAM.INFO.KV_chat_workspace",         // Info about workspace KV
				"$JS.API.CONSUMER.CREATE.KV_chat_workspace.*.>", // Creating consumers/watchers on workspace KV
			},
		},
	}

	vr := jwt.CreateValidationResults()

	claims.Validate(vr)
	if len(vr.Errors()) > 0 {
		log.Printf("failed to validate claims %s", errors.Join(vr.Errors()...))
	}

	kp, err := nkeys.FromSeed([]byte(nkeySeed))
	if err != nil {
		log.Fatal(err)
	}

	token, err := claims.Encode(kp)
	if err != nil {
		log.Fatal(err)
	}

	{
		response := jwt.NewAuthorizationResponseClaims(rc.UserNkey)
		response.Audience = rc.Server.ID
		response.Jwt = token

		token, err := response.Encode(kp)
		if err != nil {
			log.Fatal(err)
		}

		msg.Respond([]byte(token))
	}

	log.Println(rc)
}

func main() {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}

	if _, err := nc.Subscribe("$SYS.REQ.USER.AUTH", handler); err != nil {
		log.Fatalf("Error subscribing to authentication subjec: %v", err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	log.Println("Received shutdown signal, exiting...")
}
