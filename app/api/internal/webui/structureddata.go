package webui

import (
	"encoding/json"
	"html/template"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/baumac/reviews-app/app/api/internal/subjects"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// The scores of the API are from 1 to 10.
const (
	worstRating = 1
	bestRating  = 10
)

// jsonLD is one schema.org object. Decision 00021 has the properties.
type jsonLD map[string]any

// script returns the JSON text for a script element. json.Marshal escapes
// the characters that can end the element.
func (d jsonLD) script() template.JS {
	d["@context"] = "https://schema.org"
	data, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return template.JS(data)
}

func setText(d jsonLD, key, value string) {
	if value != "" {
		d[key] = value
	}
}

func restaurantData(app core.App, restaurant *core.Record, baseURL string) (jsonLD, error) {
	d := jsonLD{"@type": "Restaurant", "name": restaurant.GetString("name")}
	setText(d, "description", strings.Join(strings.Fields(restaurant.GetString("description")), " "))
	setText(d, "url", pageURL(baseURL, "/restaurants/"+restaurant.Id))
	// sameAs is the official site of the restaurant. url is the page of the
	// app. See decision 00038.
	setText(d, "sameAs", restaurant.GetString("website"))
	setText(d, "telephone", restaurant.GetString("phone"))
	if address := restaurant.GetString("address"); address != "" {
		d["address"] = address
		d["geo"] = jsonLD{
			"@type":     "GeoCoordinates",
			"latitude":  restaurant.GetFloat("latitude"),
			"longitude": restaurant.GetFloat("longitude"),
		}
	}
	return d, setAggregateRating(app, d, subjects.Restaurants, restaurant)
}

// recipeData has the schema.org properties of a recipe. totalTime is an ISO
// 8601 duration.
func recipeData(app core.App, recipe *core.Record, baseURL string) (jsonLD, error) {
	d := jsonLD{"@type": "Recipe", "name": recipe.GetString("name")}
	setText(d, "description", strings.Join(strings.Fields(recipe.GetString("description")), " "))
	setText(d, "url", pageURL(baseURL, "/recipes/"+recipe.Id))
	setText(d, "isBasedOn", recipe.GetString("source"))
	var ingredients, steps []string
	if err := recipe.UnmarshalJSONField("ingredients", &ingredients); err == nil && len(ingredients) > 0 {
		d["recipeIngredient"] = ingredients
	}
	if err := recipe.UnmarshalJSONField("steps", &steps); err == nil && len(steps) > 0 {
		instructions := make([]jsonLD, len(steps))
		for i, step := range steps {
			instructions[i] = jsonLD{"@type": "HowToStep", "text": step}
		}
		d["recipeInstructions"] = instructions
	}
	if minutes := recipe.GetInt("total_time"); minutes > 0 {
		d["totalTime"] = "PT" + strconv.Itoa(minutes) + "M"
	}
	if servings := recipe.GetInt("servings"); servings > 0 {
		d["recipeYield"] = strconv.Itoa(servings)
	}
	return d, setAggregateRating(app, d, subjects.Recipes, recipe)
}

// setAggregateRating sets no rating for a parent without reviews.
// schema.org does not allow a rating with a count of 0.
func setAggregateRating(app core.App, d jsonLD, k subjects.Tables, parent *core.Record) error {
	var rating struct {
		Count   int      `db:"count"`
		Overall *float64 `db:"overall"`
	}
	err := app.DB().
		NewQuery("SELECT (SELECT COUNT(*) FROM {{" + k.Reviews + "}} WHERE [[" + k.Parent + "]] = {:id}) AS [[count]], " +
			"(SELECT [[overall]] FROM {{" + k.Overall + "}} WHERE [[" + k.Parent + "]] = {:id}) AS [[overall]]").
		Bind(dbx.Params{"id": parent.Id}).
		One(&rating)
	if err != nil {
		return err
	}
	if rating.Count > 0 && rating.Overall != nil {
		d["aggregateRating"] = jsonLD{
			"@type":       "AggregateRating",
			"ratingValue": *rating.Overall,
			"reviewCount": rating.Count,
			"worstRating": worstRating,
			"bestRating":  bestRating,
		}
	}
	return nil
}

// reviewData has only the public fields of the author. A recipe has no
// address.
func reviewData(app core.App, kind reviewedKind, parent, review *core.Record, baseURL string) jsonLD {
	sum := 0
	for _, score := range kind.Scores {
		sum += review.GetInt(score)
	}
	item := jsonLD{"@type": kind.itemType, "name": parent.GetString("name")}
	setText(item, "url", pageURL(baseURL, kind.path(parent)))
	setText(item, "address", parent.GetString("address"))
	d := jsonLD{
		"@type":        "Review",
		"name":         review.GetString("title"),
		"itemReviewed": item,
		"reviewRating": jsonLD{
			"@type":       "Rating",
			"ratingValue": math.Round(float64(sum)/float64(len(kind.Scores))*10) / 10,
			"worstRating": worstRating,
			"bestRating":  bestRating,
		},
	}
	setText(d, "reviewBody", review.GetString("feedback"))
	setText(d, "url", pageURL(baseURL, kind.path(parent)+"/reviews/"+review.Id))
	if created := review.GetDateTime("created"); !created.IsZero() {
		d["datePublished"] = created.Time().UTC().Format(time.RFC3339)
	}
	if author, err := app.FindRecordById("users", review.GetString("creator")); err == nil {
		person := jsonLD{"@type": "Person", "name": author.GetString("name")}
		setText(person, "url", pageURL(baseURL, "/reviewers/"+author.Id))
		d["author"] = person
	}
	return d
}

func listData(list *core.Record, count int, baseURL string) jsonLD {
	d := jsonLD{"@type": "ItemList", "name": list.GetString("name"), "numberOfItems": count}
	setText(d, "url", pageURL(baseURL, "/lists/"+list.Id))
	return d
}

// reviewerData has only the public fields of the user. The count has the
// restaurant reviews and the recipe reviews.
func reviewerData(app core.App, user *core.Record, baseURL string) (jsonLD, error) {
	var reviews struct {
		Count int `db:"count"`
	}
	err := app.DB().
		NewQuery("SELECT (SELECT COUNT(*) FROM {{reviews}} WHERE [[creator]] = {:id}) + " +
			"(SELECT COUNT(*) FROM {{recipe_reviews}} WHERE [[creator]] = {:id}) AS [[count]]").
		Bind(dbx.Params{"id": user.Id}).
		One(&reviews)
	if err != nil {
		return nil, err
	}
	person := jsonLD{
		"@type": "Person",
		"name":  user.GetString("name"),
		"agentInteractionStatistic": jsonLD{
			"@type":                "InteractionCounter",
			"interactionType":      "https://schema.org/WriteAction",
			"userInteractionCount": reviews.Count,
		},
	}
	url := pageURL(baseURL, "/reviewers/"+user.Id)
	setText(person, "url", url)
	d := jsonLD{"@type": "ProfilePage", "mainEntity": person}
	setText(d, "url", url)
	return d, nil
}
