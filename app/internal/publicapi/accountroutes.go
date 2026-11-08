package publicapi

import (
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// closedUserActions are the PocketBase account routes of users. Decision
// 00001 has no password reset, verification, or email change. These routes
// would also skip the rate limit and the rules of the Public API.
var closedUserActions = []string{
	"request-password-reset", "confirm-password-reset",
	"request-verification", "confirm-verification",
	"request-email-change", "confirm-email-change",
	"request-otp",
}

// closeUserAccountRoutes returns 404 before PocketBase looks up a record.
// The response is then the same for all email addresses.
func closeUserAccountRoutes(e *core.RequestEvent) error {
	parts := strings.Split(strings.Trim(e.Request.URL.Path, "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "collections" && slices.Contains(closedUserActions, parts[3]) {
		if c, err := e.App.FindCachedCollectionByNameOrId(parts[2]); err == nil && c.Name == "users" {
			return e.NotFoundError("", nil)
		}
	}
	return e.Next()
}
