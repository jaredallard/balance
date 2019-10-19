package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/jaredallard/balance/pkg/account"
	"github.com/jaredallard/balance/pkg/handlers"
	"github.com/jaredallard/balance/pkg/social/telegram"
	log "github.com/sirupsen/logrus"
)

func main() {
	db := pg.Connect(&pg.Options{
		User:     "postgres",
		Database: "balance",
		Password: os.Getenv("POSTGRES_PASSWORD"),
	})
	defer db.Close()

	a := account.NewClient(db)

	for _, model := range []interface{}{(*account.User)(nil), (*account.Account)(nil)} {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			Temp: false,
		})
		if err != nil {
			log.Warnf("failed to create tables in database: %v", err)
		}
	}

	t, err := telegram.NewProvider(a)
	if err != nil {
		log.Fatalf("failed to create Telegram provider: %v", err)
	}

	msgs, err := t.CreateStream()
	if err != nil {
		log.Fatalf("failed to create Telegram stream: %v", err)
	}

	h := handlers.NewHandlers(a)

	log.Infof("processing message(s)")
	for msg := range msgs {
		log.Infof("got message from %v: %s", msg.From, msg.Text)

		// TODO(jaredallard): better entity handling
		tokens := strings.Split(msg.Text, " ")
		tokens[0] = strings.Replace(tokens[0], "/", "", 1)

		var reply string
		var err error

		if msg.From == nil {
			reply, err = h.HandleNewUser(&msg)
		} else if tokens[0] == "help" || tokens[0] == "start" {
			reply, err = h.HandleHelp(&msg)
		} else if tokens[0] == "add" {
			reply, err = h.HandleAdd(&msg, tokens)
		} else if tokens[0] == "history" {
			reply, err = h.HandleBalance(&msg)
		} else {
			reply = fmt.Sprintf("Unknown command '%s'", msg.Text)
		}

		if err != nil {
			log.Fatalf("failed to process message: %v", err)
		}

		if reply != "" {
			err := msg.Reply(reply)
			if err != nil {
				log.Warnf("failed to send reply: %v", err)
			}
		}
	}
}
