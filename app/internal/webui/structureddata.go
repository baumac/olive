package webui

import (
	"encoding/json"
	"html/template"
	"math"
	"strings"
	"time"

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

// restaurantData has no rating for a restaurant without reviews.
// schema.org does not allow a rating with a count of 0.
func restaurantData(app core.App, restaurant *core.Record, baseURL string) (jsonLD, error) {
	d := jsonLD{"@type": "Restaurant", "name": restaurant.GetString("name")}
	setText(d, "description", strings.Join(strings.Fields(restaurant.GetString("description")), " "))
	setText(d, "url", pageURL(baseURL, "/restaurants/"+restaurant.Id))
	if address := restaurant.GetString("address"); address != "" {
		d["address"] = address
		d["geo"] = jsonLD{
			"@type":     "GeoCoordinates",
			"latitude":  restaurant.GetFloat("latitude"),
			"longitude": restaurant.GetFloat("longitude"),
		}
	}
	var rating struct {
		Count   int      `db:"count"`
		Overall *float64 `db:"overall"`
	}
	err := app.DB().
		NewQuery("SELECT (SELECT COUNT(*) FROM {{reviews}} WHERE [[restaurant]] = {:id}) AS [[count]], " +
			"(SELECT [[overall]] FROM {{restaurant_overall}} WHERE [[restaurant]] = {:id}) AS [[overall]]").
		Bind(dbx.Params{"id": restaurant.Id}).
		One(&rating)
	if err != nil {
		return nil, err
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
	return d, nil
}

// reviewData has only the public fields of the author.
func reviewData(app core.App, restaurant, review *core.Record, baseURL string) jsonLD {
	sum := 0
	for _, score := range []string{"space", "service", "food", "price", "cleanliness"} {
		sum += review.GetInt(score)
	}
	item := jsonLD{"@type": "Restaurant", "name": restaurant.GetString("name")}
	setText(item, "url", pageURL(baseURL, "/restaurants/"+restaurant.Id))
	setText(item, "address", restaurant.GetString("address"))
	d := jsonLD{
		"@type":        "Review",
		"name":         review.GetString("title"),
		"itemReviewed": item,
		"reviewRating": jsonLD{
			"@type":       "Rating",
			"ratingValue": math.Round(float64(sum)/5*10) / 10,
			"worstRating": worstRating,
			"bestRating":  bestRating,
		},
	}
	setText(d, "reviewBody", review.GetString("feedback"))
	setText(d, "url", pageURL(baseURL, "/restaurants/"+restaurant.Id+"/reviews/"+review.Id))
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

// reviewerData has only the public fields of the user.
func reviewerData(app core.App, user *core.Record, baseURL string) (jsonLD, error) {
	var reviews struct {
		Count int `db:"count"`
	}
	err := app.DB().
		NewQuery("SELECT COUNT(*) AS [[count]] FROM {{reviews}} WHERE [[creator]] = {:id}").
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
