package utils

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"resty.dev/v3"
)

const EmbedServerPort = "5678"

func CheckSingleton() {
	if !PortAvailable(EmbedServerPort) {
		resp, err := resty.New().SetTimeout(500 * time.Millisecond).R().
			Get(fmt.Sprintf("http://127.0.0.1:%s/health", EmbedServerPort))
		if err != nil {
			log.Print(err)
			return
		}
		if resp.String() == "ok" {
			log.Fatal("app exists")
		}
	}
}

func EmbedServer() {
	log.Printf("embed server http://127.0.0.1:%v", EmbedServerPort)

	app := fiber.New(fiber.Config{})
	app.Use(cors.New())
	app.Use(recover.New())
	app.Use(requestid.New())

	app.Get("/", func(c fiber.Ctx) error { return nil })
	app.Get("/health", func(c fiber.Ctx) error { return c.SendString("ok") })

	err := app.Listen(net.JoinHostPort("127.0.0.1", EmbedServerPort), fiber.ListenConfig{
		DisableStartupMessage: true,
	})
	if err != nil {
		log.Fatal("embed server error")
	}
}
