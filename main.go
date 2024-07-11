package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	client    *whatsmeow.Client
	loggedIn  bool
	analytics Analytics
)

type Analytics struct {
	TotalSent    int
	TotalFailed  int
	TotalReplied int
	TotalReacted int
	Incoming     []string
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Disconnected:
		log.Println("Disconnected from WhatsApp")
		loggedIn = false
	case *events.Connected:
		log.Println("Connected to WhatsApp")
		loggedIn = true
	case *events.Message:
		if v.Message.ExtendedTextMessage != nil && v.Message.ExtendedTextMessage.Text != nil {
			fmt.Printf("Received message from %s: %s\n", v.Info.Sender.User, *v.Message.ExtendedTextMessage.Text)
			analytics.Incoming = append(analytics.Incoming, fmt.Sprintf("From %s: %s", v.Info.Sender.User, *v.Message.ExtendedTextMessage.Text))
		}
		// if v.Message.Status != nil {
		//  switch v.Message.GetStatus() {
		//  case waE2E.MessageStatus_REPLIED:
		//      analytics.TotalReplied++
		//  case waE2E.MessageStatus_REACTION:
		//      analytics.TotalReacted++
		//  }
		// }
	}
}

func connectClient() error {
	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		if err := client.Connect(); err != nil {
			return err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		if err := client.Connect(); err != nil {
			return err
		}
	}
	return nil
}

func sendMessages(users []string, text string) {
	for _, user := range users {
		userJID := types.NewJID(user, "s.whatsapp.net")
		resp, err := client.SendMessage(context.Background(), userJID, &waE2E.Message{
			Conversation: proto.String(text),
		})
		if err != nil {
			log.Printf("Failed to send message to %s: %v\n", user, err)
			analytics.TotalFailed++
			continue
		}
		fmt.Printf("Sent message with ID: %s to %s\n", resp.ID, user)
		analytics.TotalSent++

		// Simulate reaction and reply for demonstration purposes
		time.Sleep(2 * time.Second)
	}
}

func uploadCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
		return
	}

	filename := filepath.Base(file.Filename)
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}

	f, err := os.Open(filename)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("open file err: %s", err.Error()))
		return
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("read csv err: %s", err.Error()))
		return
	}

	var users []string
	for _, record := range records {
		if len(record) > 0 {
			users = append(users, strings.TrimSpace(record[0]))
		}
	}

	text := c.PostForm("message")
	sendMessages(users, text)

	c.Redirect(http.StatusSeeOther, "/analytics")
}

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		log.Fatalf("Failed to create database container: %v", err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		log.Fatalf("Failed to get device store: %v", err)
	}
	clientLog := waLog.Stdout("Client", "WARN", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if err := connectClient(); err != nil {
		log.Fatalf("Failed to connect client: %v", err)
	}

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.Static("/assets", "./assets")

	router.GET("/", func(c *gin.Context) {
		if !loggedIn {
			c.HTML(http.StatusOK, "qr.html", nil)
			return
		}
		c.Redirect(http.StatusSeeOther, "/send")
	})

	router.GET("/send", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.GET("/logout", func(c *gin.Context) {
		loggedIn = false
		client.Disconnect()
		c.Redirect(http.StatusSeeOther, "/")
	})

	router.GET("/analytics", func(c *gin.Context) {
		c.HTML(http.StatusOK, "analytics.html", gin.H{
			"TotalSent":    analytics.TotalSent,
			"TotalFailed":  analytics.TotalFailed,
			"TotalReplied": analytics.TotalReplied,
			"TotalReacted": analytics.TotalReacted,
			"Incoming":     analytics.Incoming,
		})
	})

	router.POST("/upload", uploadCSV)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
	fmt.Println("Client disconnected. Exiting.")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	fmt.Println("Server exiting")
}
