package main

import "github.com/flowline-io/flowbot/internal/server"

// @title			Flowbot API
// @version		1.0
// @description	Flowbot Chatbot API
// @termsOfService	http://swagger.io/terms/
// @contact.name	API Support
// @contact.email	dev@tsundata.com
// @license.name	GPL 3.0
// @license.url	https://github.com/flowline-io/flowbot/blob/master/LICENSE
// @host			localhost:6060
// @BasePath		/bot
func main() {
	server.Run()
}
