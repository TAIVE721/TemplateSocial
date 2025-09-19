package mailer

import "embed"

//go:embed "templates"
var templateFS embed.FS

type Client interface {
	Send(templateFile, username, email string, data any) (int, error)
}
