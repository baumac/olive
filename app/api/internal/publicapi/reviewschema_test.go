package publicapi_test

import (
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestOverallColumnEqualsResponse(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	ana := e.signUp("ana@example.com", "Ana")
	for i, body := range []string{
		`{"space":1,"service":1,"food":1,"price":1,"cleanliness":2,"title":"T"}`,
		`{"space":7,"service":9,"food":8,"price":6,"cleanliness":8,"title":"T"}`,
		`{"space":10,"service":10,"food":10,"price":10,"cleanliness":9,"title":"T"}`,
		`{"space":3,"service":3,"food":4,"price":3,"cleanliness":3,"title":"T"}`,
	} {
		restaurant := e.createRestaurant(ana.Token, `{"name":"R`+string(rune('a'+i))+`"}`)["id"].(string)
		review := e.createReview(ana.Token, restaurant, body)
		var stored float64
		err := e.app.DB().Select("overall").From("reviews").Where(dbx.HashExp{"id": review["id"]}).Row(&stored)
		if err != nil {
			t.Fatal(err)
		}
		if stored != review["overall"] {
			t.Errorf("%s: column is %v, response has %v", body, stored, review["overall"])
		}
	}
}

func TestReviewSchemaGuard(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	for name, change := range map[string]func(c *core.Collection){
		"overall field": func(c *core.Collection) { c.Fields.Add(&core.NumberField{Name: "overall"}) },
		"no price":      func(c *core.Collection) { c.Fields.RemoveByName("price") },
	} {
		t.Run(name, func(t *testing.T) {
			reviews, err := e.app.FindCollectionByNameOrId("reviews")
			if err != nil {
				t.Fatal(err)
			}
			change(reviews)
			err = e.app.Save(reviews)
			if err == nil || !strings.Contains(err.Error(), "generated overall column") {
				t.Errorf("save error is %v, want the overall column message", err)
			}
		})
	}

	reviews, err := e.app.FindCollectionByNameOrId("reviews")
	if err != nil {
		t.Fatal(err)
	}
	reviews.Fields.Add(&core.TextField{Name: "note"})
	if err := e.app.Save(reviews); err != nil {
		t.Errorf("a new field: %v", err)
	}
}

func TestRecipeReviewSchemaGuard(t *testing.T) {
	t.Parallel()
	e := newEnv(t, testOptions)
	reviews, err := e.app.FindCollectionByNameOrId("recipe_reviews")
	if err != nil {
		t.Fatal(err)
	}
	reviews.Fields.RemoveByName("value")
	err = e.app.Save(reviews)
	if err == nil || !strings.Contains(err.Error(), "recipe_reviews table uses the value field") {
		t.Errorf("save error is %v, want the overall column message", err)
	}
}
