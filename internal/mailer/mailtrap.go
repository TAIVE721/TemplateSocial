package mailer

import (
	"bytes"
	"text/template"

	gomail "gopkg.in/mail.v2"
)

type MailtrapClient struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (m MailtrapClient) Send(templateFile, _, email string, data any) (int, error) {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return 0, err
	}

	subject := new(bytes.Buffer)
	if err = tmpl.ExecuteTemplate(subject, "subject", data); err != nil {
		return 0, err
	}

	body := new(bytes.Buffer)
	if err = tmpl.ExecuteTemplate(body, "body", data); err != nil {
		return 0, err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.From)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/html", body.String())

	d := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)

	if err := d.DialAndSend(msg); err != nil {
		return 0, err
	}

	return 200, nil
}
