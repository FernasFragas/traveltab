package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
	"strings"
	"time"
)

var botUserAgents = []string{"bot", "crawl", "spider", "slurp", "curl", "wget", "python-requests", "go-http-client"}

// trackVisit records a page view before handing the request to the next handler.
// Failures are logged and never block the request.
func (s *Server) trackVisit(ctx *fiber.Ctx) error {
	userAgent := ctx.Get(fiber.HeaderUserAgent)
	if s.storage != nil && !isBot(userAgent) {
		// Fly's proxy sets Fly-Client-IP to the real client address.
		ip := ctx.Get("Fly-Client-IP")
		if ip == "" {
			ip = ctx.IP()
		}

		if err := s.storage.SaveVisit(visitorID(ip, userAgent), ctx.Path(), ctx.FormValue("city_name")); err != nil {
			log.Printf("Error saving visit: %v", err)
		}
	}

	return ctx.Next()
}

// visitorID returns an anonymous, stable identifier so raw IPs are never stored.
func visitorID(ip, userAgent string) string {
	sum := sha256.Sum256([]byte(ip + "|" + userAgent))
	return hex.EncodeToString(sum[:8])
}

func isBot(userAgent string) bool {
	if userAgent == "" {
		return true
	}

	userAgent = strings.ToLower(userAgent)
	for _, bot := range botUserAgents {
		if strings.Contains(userAgent, bot) {
			return true
		}
	}

	return false
}

// showStats returns visit stats as JSON. It is only reachable with the STATS_TOKEN
// set in the environment, e.g. /stats?token=...&days=7
func (s *Server) showStats(ctx *fiber.Ctx) error {
	token := os.Getenv("STATS_TOKEN")
	if token == "" || ctx.Query("token") != token {
		return ctx.SendStatus(fiber.StatusNotFound)
	}

	days := ctx.QueryInt("days", 7)
	if days < 1 {
		days = 7
	}

	if s.storage == nil {
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}
	stats, err := s.storage.GetVisitStats(time.Now().UTC().AddDate(0, 0, -days))
	if err != nil {
		log.Printf("Error retrieving visit stats with error %s", err)
		return ctx.SendStatus(fiber.StatusInternalServerError)
	}

	return ctx.JSON(stats)
}
