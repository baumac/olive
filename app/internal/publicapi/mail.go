package publicapi

import (
	"cmp"
	"fmt"
	"html"
	"net/mail"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// defaultEmailLimit is the number of emails that one address gets in
// emailWindow.
const (
	defaultEmailLimit = 5
	emailWindow       = time.Hour
	kindConfirmation  = "confirmation"
)

// email is one message. Each paragraph is plain text. A paragraph that is
// only a URL becomes a link in the HTML part.
type email struct {
	// kind names the reason of the email. Each kind has its own count for
	// the email limit of an address.
	kind       string
	to         string
	subject    string
	paragraphs []string
}

// postman sends the emails of the Public API. See decision 00037.
type postman struct {
	app core.App
	// limiter counts the emails of each kind and address. A request without
	// a token then cannot fill the mailbox of a person.
	limiter *rateLimiter
	// sync sends each email before send returns. Tests use it.
	sync bool
}

func newPostman(app core.App, opts Options) *postman {
	return &postman{
		app:     app,
		limiter: newRateLimiter(cmp.Or(opts.EmailLimit, defaultEmailLimit), emailWindow),
		sync:    opts.SyncEmail,
	}
}

// send does not wait for the mail server. The response time of a request is
// then the same with and without an email.
func (p *postman) send(m email) {
	if wait, _ := p.limiter.allow(limitKey(m.kind, m.to)); wait > 0 {
		p.app.Logger().Warn("Email not sent. The address is at its email limit.", "subject", m.subject)
		return
	}
	if p.sync {
		p.deliver(m)
		return
	}
	go p.deliver(m)
}

func limitKey(kind, address string) string {
	return kind + " " + strings.ToLower(address)
}

// atLimit reports if send drops an email of kind to address.
func (p *postman) atLimit(kind, address string) bool {
	key := limitKey(kind, address)
	wait, start := p.limiter.allow(key)
	if wait > 0 {
		return true
	}
	p.limiter.giveBack(key, start)
	return false
}

func (p *postman) deliver(m email) {
	text := strings.Join(m.paragraphs, "\n\n")
	settings := p.app.Settings()
	if !settings.SMTP.Enabled {
		// A person without a mail server gets the link from the log.
		p.app.Logger().Info("Email not sent. SMTP is off.", "to", m.to, "subject", m.subject, "text", text)
		return
	}
	err := p.app.NewMailClient().Send(&mailer.Message{
		From:    mail.Address{Name: settings.Meta.SenderName, Address: settings.Meta.SenderAddress},
		To:      []mail.Address{{Address: m.to}},
		Subject: m.subject,
		Text:    text,
		HTML:    emailHTML(m.paragraphs),
	})
	if err != nil {
		p.app.Logger().Error("Email not sent", "subject", m.subject, "error", err)
	}
}

func emailHTML(paragraphs []string) string {
	var b strings.Builder
	for _, text := range paragraphs {
		safe := html.EscapeString(text)
		isURL := strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://")
		// A name of a user can start a paragraph. It must not become a link.
		if isURL && !strings.ContainsAny(text, " \t\r\n") {
			safe = `<a href="` + safe + `">` + safe + `</a>`
		}
		b.WriteString("<p>" + safe + "</p>\n")
	}
	return b.String()
}

func (p *postman) siteName() string {
	return p.app.Settings().Meta.AppName
}

// siteURL returns the Application URL with path. The token of a link is
// after the #. Browsers do not send this part to a server.
func (p *postman) siteURL(path string) string {
	return strings.TrimRight(p.app.Settings().Meta.AppURL, "/") + path
}

func (p *postman) sendConfirmation(user *core.Record) error {
	token, err := user.NewVerificationToken()
	if err != nil {
		return err
	}
	p.send(email{
		kind:    kindConfirmation,
		to:      user.Email(),
		subject: "Confirm your email address for " + p.siteName(),
		paragraphs: []string{
			"Hello,",
			"Open this link to confirm your email address and to finish your account:",
			p.siteURL("/confirm-email#token=" + token),
			"The link works for " + durationText(user.Collection().VerificationToken.DurationTime()) + ".",
			"If you did not make this account, ignore this email.",
		},
	})
	return nil
}

func (p *postman) sendAccountExists(user *core.Record) {
	p.send(email{
		kind:    "account-exists",
		to:      user.Email(),
		subject: "You already have an account at " + p.siteName(),
		paragraphs: []string{
			"Hello,",
			"Someone tried to make an account with this email address. You already have an account. Sign in here:",
			p.siteURL("/sign-in"),
			"If you do not know your password, set a new one here:",
			p.siteURL("/forgot-password"),
			"If this was not you, ignore this email. Your account did not change.",
		},
	})
}

func (p *postman) sendPasswordReset(user *core.Record) error {
	token, err := user.NewPasswordResetToken()
	if err != nil {
		return err
	}
	p.send(email{
		kind:    "password-reset",
		to:      user.Email(),
		subject: "Set a new password for " + p.siteName(),
		paragraphs: []string{
			"Hello,",
			"Open this link to set a new password:",
			p.siteURL("/reset-password#token=" + token),
			"The link works for " + durationText(user.Collection().PasswordResetToken.DurationTime()) + ".",
			"If you did not ask for a new password, ignore this email. Your password did not change.",
		},
	})
	return nil
}

// sendWaitsForApproval tells each superuser about user.
func (p *postman) sendWaitsForApproval(user *core.Record) error {
	superusers, err := p.app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}
	for _, su := range superusers {
		p.send(email{
			kind:    "waits-for-approval",
			to:      su.Email(),
			subject: p.siteName() + ": a new user waits for approval",
			paragraphs: []string{
				fmt.Sprintf("%s (%s) confirmed the email address and waits for approval.", user.GetString("name"), user.Email()),
				"Open the dashboard, and search for approved = false in the users collection:",
				p.siteURL("/_/"),
			},
		})
	}
	return nil
}

func (p *postman) sendApproved(user *core.Record) {
	p.send(email{
		kind:    "approved",
		to:      user.Email(),
		subject: "Your account at " + p.siteName() + " is approved",
		paragraphs: []string{
			"Hello,",
			"An administrator approved your account. You can add restaurants, write reviews, and save lists now.",
			p.siteURL("/"),
		},
	})
}

// notifyApproval sends an email to a confirmed user when a superuser turns
// approved on.
func notifyApproval(app core.App, p *postman) {
	app.OnRecordAfterUpdateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetBool("approved") && !e.Record.Original().GetBool("approved") && e.Record.Verified() {
			p.sendApproved(e.Record)
		}
		return e.Next()
	})
}

func durationText(d time.Duration) string {
	switch {
	case d >= 48*time.Hour:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	case d >= 2*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	default:
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
}
