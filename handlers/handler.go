package handlers

import (
	"codenames-server/handlers/room"
	"net/http"

	"github.com/labstack/echo/v4"
)

func InitHandler(app *echo.Group) {
	app.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Welcome to Codenames Backend")
	})

	rm := app.Group("/room")
	{
		rm.GET("/join", room.JoinRoom)
		rm.GET("/create", room.CreateRoom)
	}
}
