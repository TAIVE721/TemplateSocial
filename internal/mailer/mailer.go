package mailer

import "embed"

var templateFS embed.FS

type Client interface {
	Send(templateFile, username, email string, data any) (int, error)
}
