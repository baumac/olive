package publicapi

import (
	"testing"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
)

func TestSpecHasTheSessionCookie(t *testing.T) {
	doc, err := reviewsapi.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}
	if name := doc.Components.SecuritySchemes["cookieAuth"].Value.Name; name != sessionCookieName {
		t.Errorf("cookieAuth name %q, want %q", name, sessionCookieName)
	}
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			if op.Security == nil {
				continue
			}
			bearer, cookie := false, false
			for _, req := range *op.Security {
				_, b := req["bearerAuth"]
				_, c := req["cookieAuth"]
				bearer, cookie = bearer || b, cookie || c
			}
			if bearer != cookie {
				t.Errorf("%s %s: bearerAuth %v, cookieAuth %v", method, path, bearer, cookie)
			}
		}
	}
}
