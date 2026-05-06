package websocket

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ismael/qr-restaurant/internal/shared/models"
	"gorm.io/gorm"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	hub          *Hub
	conn         *websocket.Conn
	send         chan *Message
	sessionToken string
}

type Handler struct {
	hub             *Hub
	db              *gorm.DB
	frontendBaseURL string
	appEnv          string
}

func NewWSHandler(hub *Hub, db *gorm.DB, frontendBaseURL, appEnv string) *Handler {
	return &Handler{hub: hub, db: db, frontendBaseURL: frontendBaseURL, appEnv: appEnv}
}

func (h *Handler) ServeWS(c *gin.Context) {
	sessionToken := c.Param("sessionToken")
	c.Set("session_id", sessionToken)
	if sessionToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session token required"})
		return
	}

	sessionUUID, err := uuid.Parse(sessionToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}

	var session models.Session
	if err := h.db.First(&session, "session_token = ? AND status = ? AND (expires_at IS NULL OR expires_at > NOW())", sessionUUID, "ACTIVE").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Active session not found"})
		return
	}
	_ = h.db.Model(&models.Session{}).Where("id = ?", session.ID).Update("last_seen_at", gorm.Expr("NOW()")).Error

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// upgrader.Upgrade already sends an HTTP error response
		return
	}

	client := &Client{
		hub:          h.hub,
		conn:         conn,
		send:         make(chan *Message, 256),
		sessionToken: sessionToken,
	}

	client.hub.Register(client)

	go client.writePump()
	go client.readPump()
}

func (h *Handler) checkOrigin(r *http.Request) bool {
	if h.appEnv != "production" {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	allowed, err := url.Parse(h.frontendBaseURL)
	if err != nil || allowed.Scheme == "" || allowed.Host == "" {
		return false
	}
	got, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return got.Scheme == allowed.Scheme && got.Host == allowed.Host
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// For MVP, we only push down to the client. We ignore messages sent from client.
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
