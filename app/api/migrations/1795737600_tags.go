package migrations

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// tagPattern is the pattern of the Tag schema. The dashboard then cannot
// save a name that the API refuses.
const tagPattern = `^[\p{Ll}\p{Lm}\p{Lo}\p{Nd}][\p{Ll}\p{Lm}\p{Lo}\p{Mn}\p{Mc}\p{Nd}]*(-[\p{Ll}\p{Lm}\p{Lo}\p{Nd}][\p{Ll}\p{Lm}\p{Lo}\p{Mn}\p{Mc}\p{Nd}]*)*$`

// maxTags is the maxItems of the Tags schema.
const maxTags = 20

func init() {
	m.Register(tagsUp, tagsDown)
}

// tagsUp makes the tags collection of decision 00011. The tags of each
// parent become a relation to it.
func tagsUp(app core.App) error {
	tags := core.NewBaseCollection("tags")
	tags.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 63, Pattern: tagPattern},
		&core.TextField{Name: "name_key", Max: 255},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	tags.AddIndex("idx_tags_name", true, "name", "")
	if err := app.Save(tags); err != nil {
		return err
	}
	ids := map[string]string{}
	for _, k := range subjectKinds {
		names, err := k.storedTags(app, "[[tags]]")
		if err != nil {
			return err
		}
		lists := make(map[string][]string, len(names))
		for parent, list := range names {
			for _, name := range list {
				if _, ok := ids[name]; ok {
					continue
				}
				id, err := insertTag(app, name)
				if err != nil {
					return err
				}
				ids[name] = id
			}
			lists[parent] = mapAll(list, ids)
		}
		field := &core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: maxTags}
		if err := k.replaceTags(app, field, lists); err != nil {
			return err
		}
	}
	return nil
}

func tagsDown(app core.App) error {
	for _, k := range subjectKinds {
		names, err := k.storedTags(app, "(SELECT json_group_array([[name]]) FROM (SELECT [[t.name]] AS [[name]] FROM json_each({{"+k.parents+"}}.[[tags]]) j "+
			"JOIN {{tags}} t ON [[t.id]] = [[j.value]] ORDER BY [[j.key]]))")
		if err != nil {
			return err
		}
		if err := k.replaceTags(app, &core.JSONField{Name: "tags"}, names); err != nil {
			return err
		}
	}
	tags, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		return err
	}
	return app.Delete(tags)
}

// storedTags gets the strings of the tags of each parent. The column
// expression gives a JSON array.
func (k subjectKind) storedTags(app core.App, column string) (map[string][]string, error) {
	var rows []struct {
		ID   string        `db:"id"`
		Tags types.JSONRaw `db:"tags"`
	}
	if err := app.DB().NewQuery("SELECT [[id]], " + column + " AS [[tags]] FROM {{" + k.parents + "}}").All(&rows); err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(rows))
	for _, row := range rows {
		var values []any
		if json.Unmarshal(row.Tags, &values) != nil {
			continue
		}
		var list []string
		for _, v := range values {
			if s, ok := v.(string); ok && s != "" && !slices.Contains(list, s) {
				list = append(list, s)
			}
		}
		if len(list) > 0 {
			out[row.ID] = list
		}
	}
	return out, nil
}

// replaceTags puts field in the place of the tags field. The SQL writes
// keep the updated time of the parents. They write no version.
func (k subjectKind) replaceTags(app core.App, field core.Field, lists map[string][]string) error {
	if err := execAll(app, k.dropParentTriggers()...); err != nil {
		return err
	}
	c, err := app.FindCollectionByNameOrId(k.parents)
	if err != nil {
		return err
	}
	pos := slices.IndexFunc(c.Fields, func(f core.Field) bool { return f.GetName() == "tags" })
	c.Fields.RemoveByName("tags")
	c.Fields.AddAt(pos, field)
	if err := app.Save(c); err != nil {
		return err
	}
	for id, list := range lists {
		value, err := json.Marshal(list)
		if err != nil {
			return err
		}
		if _, err := app.DB().Update(k.parents, dbx.Params{"tags": string(value)}, dbx.HashExp{"id": id}).Execute(); err != nil {
			return err
		}
	}
	return execAll(app, slices.Concat(k.parentTriggers(), k.rebuildTags())...)
}

func insertTag(app core.App, name string) (string, error) {
	id := core.GenerateDefaultRandomId()
	_, err := app.DB().Insert("tags", dbx.Params{
		"id": id, "name": name, "name_key": tagKey(name), "created": types.NowDateTime().String(),
	}).Execute()
	return id, err
}

func mapAll(names []string, ids map[string]string) []string {
	out := make([]string, len(names))
	for i, name := range names {
		out[i] = ids[name]
	}
	return out
}

// tagKey is a copy of the name key of the Public API. A migration does not
// change when the code changes.
func tagKey(name string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	key, _, err := transform.String(t, strings.ToLower(name))
	if err != nil {
		return strings.ToLower(name)
	}
	return key
}
