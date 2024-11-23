package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/skip2/go-qrcode"
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
	qrCode    []byte
)

type Analytics struct {
	TotalSent    int
	TotalFailed  int
	TotalReplied int
	TotalReacted int
	Incoming     []string
}

// Define the Template structure
type Template struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

var templates = make(map[int]Template)
var nextID = 1 // Simple ID auto-increment

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
				var err error
				qrCode, err = qrcode.Encode(evt.Code, qrcode.Medium, 256)
				if err != nil {
					return err
				}
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

		time.Sleep(2 * time.Second) // Simulate reaction and reply for demonstration purposes
	}
}

// New handler for sending message to an array of numbers
func sendMessagesAPI(c *gin.Context) {
	var requestData struct {
		Numbers []string `json:"numbers"`
		Message string   `json:"message"`
	}

	// Parse the JSON request body
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Check if numbers and message are provided
	if len(requestData.Numbers) == 0 || strings.TrimSpace(requestData.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both numbers and message are required"})
		return
	}

	// Send the message to each number
	sendMessages(requestData.Numbers, requestData.Message)

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"status": "Messages sent successfully"})
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

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	if err != nil {
		log.Printf("Failed to open browser: %v\n", err)
	}
}

// Create a new template
func createTemplate(c *gin.Context) {
	var newTemplate Template
	if err := c.ShouldBindJSON(&newTemplate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	// Assign an ID and store the template
	newTemplate.ID = nextID
	templates[nextID] = newTemplate
	nextID++

	c.JSON(http.StatusCreated, newTemplate)
}

// Get all templates
func getTemplates(c *gin.Context) {
	var templateList []Template
	for _, tmpl := range templates {
		templateList = append(templateList, tmpl)
	}
	c.JSON(http.StatusOK, templateList)
}

// Get a single template by ID
func getTemplate(c *gin.Context) {
	id := c.Param("id")
	templateID, err := strconv.Atoi(id)
	if err != nil || templateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	template, exists := templates[templateID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// Update a template by ID
func updateTemplate(c *gin.Context) {
	id := c.Param("id")
	templateID, err := strconv.Atoi(id)
	if err != nil || templateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var updatedTemplate Template
	if err := c.ShouldBindJSON(&updatedTemplate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	template, exists := templates[templateID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Update the template
	template.Title = updatedTemplate.Title
	template.Content = updatedTemplate.Content
	templates[templateID] = template

	c.JSON(http.StatusOK, template)
}

// Delete a template by ID
func deleteTemplate(c *gin.Context) {
	id := c.Param("id")
	templateID, err := strconv.Atoi(id)
	if err != nil || templateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	_, exists := templates[templateID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	// Delete the template
	delete(templates, templateID)

	c.JSON(http.StatusOK, gin.H{"status": "Template deleted"})
}

// Helper function to render the template
func renderTemplate(content string, username, coupon string) (string, error) {
	tmpl, err := template.New("template").Parse(content)
	if err != nil {
		return "", err
	}

	data := struct {
		Username string
		Coupon   string
	}{
		Username: username,
		Coupon:   coupon,
	}

	var renderedContent strings.Builder
	err = tmpl.Execute(&renderedContent, data)
	if err != nil {
		return "", err
	}

	return renderedContent.String(), nil
}

func main() {
	// Create a CORS configuration allowing localhost:1212 (desktop app)
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:1212"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}

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

	router := gin.Default()

	router.Use(cors.New(corsConfig))

	router.LoadHTMLGlob("templates/*")
	router.Static("/assets", "./assets")

	router.GET("/qr", func(c *gin.Context) {
		if !loggedIn {
			c.HTML(http.StatusOK, "qr.html", nil)
			return
		}
		c.Redirect(http.StatusSeeOther, "/send")
	})

	router.GET("/", func(c *gin.Context) {
		if !loggedIn {
			c.HTML(http.StatusOK, "index.html", nil)
			return
		}
		c.Redirect(http.StatusSeeOther, "/send")
	})

	router.GET("/send", func(c *gin.Context) {
		c.HTML(http.StatusOK, "send.html", nil)
	})

	router.GET("/logout", func(c *gin.Context) {
		loggedIn = false
		client.Logout()
		client.Disconnect()
		c.Redirect(http.StatusSeeOther, "/")
	})

	router.GET("/analytics", func(c *gin.Context) {
		if !loggedIn {
			c.Redirect(http.StatusSeeOther, "/qr")
			return
		}
		c.HTML(http.StatusOK, "analytics.html", gin.H{
			"TotalSent":    analytics.TotalSent,
			"TotalFailed":  analytics.TotalFailed,
			"TotalReplied": len(analytics.Incoming),
			"TotalReacted": analytics.TotalReacted,
			"Incoming":     analytics.Incoming,
		})
	})

	router.GET("/view-messages", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"incomingMessages": analytics.Incoming})
	})

	router.GET("/messages", func(c *gin.Context) {
		if !loggedIn {
			c.Redirect(http.StatusSeeOther, "/qr")
			return
		}
		c.HTML(http.StatusOK, "messages.html", nil)
	})

	router.POST("/upload", uploadCSV)

	router.GET("/qr-code", func(c *gin.Context) {
		c.Header("Content-Type", "image/png")
		c.Writer.Write(qrCode)
	})

	router.GET("/login-status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"loggedIn": loggedIn})
	})

	// New route for sending message via POST /send-msg
	router.POST("/send-msg", sendMessagesAPI)

	// Templates Routes
	router.GET("/templates", getTemplates)          // List all templates
	router.GET("/templates/:id", getTemplate)       // Get a specific template
	router.POST("/templates", createTemplate)       // Create a new template
	router.PUT("/templates/:id", updateTemplate)    // Update an existing template
	router.DELETE("/templates/:id", deleteTemplate) // Delete a template

	srv := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}

	// Start the server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	// openBrowser("http://localhost:8000")

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	// Connect the WhatsApp client and render the QR code
	if err := connectClient(); err != nil {
		log.Fatalf("Failed to connect client: %v", err)
	}

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
