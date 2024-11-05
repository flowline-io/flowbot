package utils

import (
	"fmt"
	"net"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

const EmbedServerPort = "5678"

func CheckSingleton() {
	if !PortAvailable(EmbedServerPort) {
		resp, err := resty.New().SetTimeout(500 * time.Millisecond).R().
			Get(fmt.Sprintf("http://127.0.0.1:%s/health", EmbedServerPort))
		if err != nil {
			flog.Error(err)
			return
		}
		if resp.String() == "ok" {
			flog.Fatal("app exists")
		}
	}
}

func EmbedServer() {
	go func() {
		flog.Info("embed server http://127.0.0.1:%v", EmbedServerPort)

		app := fiber.New(fiber.Config{DisableStartupMessage: true})
		app.Use(cors.New())
		app.Use(recover.New())
		app.Use(requestid.New())

		app.Get("/", func(c *fiber.Ctx) error { return nil })
		app.Get("/health", func(c *fiber.Ctx) error { return c.SendString("ok") })

		err := app.Listen(net.JoinHostPort("127.0.0.1", EmbedServerPort))
		if err != nil {
			flog.Fatal("embed server error")
		}
	}()
}