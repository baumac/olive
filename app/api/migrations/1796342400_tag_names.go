package migrations

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"golang.org/x/text/unicode/norm"
)

func init() {
	m.Register(tagNamesUp, func(core.App) error { return nil })
}

// tagNamesUp changes the existing tag names to the rules of decision 00011.
// Tags with one new name become the oldest of them. The down migration
// cannot split them again.
func tagNamesUp(app core.App) error {
	var rows []struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	if err := app.DB().Select("id", "name").From("tags").OrderBy("created", "id").All(&rows); err != nil {
		return err
	}
	kept := map[string]string{}
	merged := map[string]string{}
	renamed := map[string]string{}
	for _, row := range rows {
		name := tagNameOf(row.Name)
		// A compatibility character can grow to more than 63 characters.
		if name == "" || len([]rune(name)) > 63 {
			name = row.Name
		}
		if id, ok := kept[name]; ok {
			merged[row.ID] = id
			continue
		}
		kept[name] = row.ID
		if name != row.Name {
			renamed[row.ID] = name
		}
	}
	if len(merged) == 0 && len(renamed) == 0 {
		return nil
	}

	for _, k := range subjectKinds {
		if err := k.mergeTags(app, merged); err != nil {
			return err
		}
	}
	for id := range merged {
		if _, err := app.DB().Delete("tags", dbx.HashExp{"id": id}).Execute(); err != nil {
			return err
		}
	}
	for id, name := range renamed {
		if _, err := app.DB().Update("tags", dbx.Params{"name": name, "name_key": tagKey(name)}, dbx.HashExp{"id": id}).Execute(); err != nil {
			return err
		}
	}
	for _, k := range subjectKinds {
		if err := k.indexTagNames(app, merged, renamed); err != nil {
			return err
		}
	}
	return nil
}

// mergeTags changes the merged IDs of each parent to the kept IDs. The
// triggers then change the tag table. The SQL write keeps the updated time.
func (k subjectKind) mergeTags(app core.App, merged map[string]string) error {
	lists, err := k.storedTags(app, "[[tags]]")
	if err != nil {
		return err
	}
	for parent, ids := range lists {
		next := make([]string, 0, len(ids))
		for _, id := range ids {
			if to, ok := merged[id]; ok {
				id = to
			}
			if !slices.Contains(next, id) {
				next = append(next, id)
			}
		}
		if slices.Equal(next, ids) {
			continue
		}
		value, err := json.Marshal(next)
		if err != nil {
			return err
		}
		if _, err := app.DB().Update(k.parents, dbx.Params{"tags": string(value)}, dbx.HashExp{"id": parent}).Execute(); err != nil {
			return err
		}
	}
	return nil
}

// indexTagNames writes the new names into the FTS5 rows of the parents with
// a changed tag.
func (k subjectKind) indexTagNames(app core.App, merged, renamed map[string]string) error {
	var changed []any
	for _, id := range merged {
		changed = append(changed, id)
	}
	for id := range renamed {
		changed = append(changed, id)
	}
	var ids []string
	if err := app.DB().Select("[[" + k.parent + "]]").Distinct(true).From(k.tagsTable()).Where(dbx.In("tag", changed...)).Column(&ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	parents := make([]any, len(ids))
	for i, id := range ids {
		parents[i] = id
	}
	fts := k.parents + "_fts"
	names := dbx.NewExp("COALESCE((SELECT group_concat([[name]], ' ') FROM (" +
		"SELECT [[t.name]] AS [[name]] FROM {{" + k.parents + "}} p, json_each(p.[[tags]]) j JOIN {{tags}} t ON [[t.id]] = [[j.value]] " +
		"WHERE [[p.id]] = {{" + fts + "}}.[[id]] ORDER BY [[j.key]])), '')")
	_, err := app.DB().Update(fts, dbx.Params{"tags": names}, dbx.In("id", parents...)).Execute()
	return err
}

// tagNameOf is a copy of tagname.Of of decision 00011. A migration does not
// change when the code changes.
func tagNameOf(text string) string {
	var b strings.Builder
	inWord, gap := false, false
	for _, r := range strings.ToLower(norm.NFKC.String(text)) {
		switch {
		case tagIgnorable(r):
		case unicode.In(r, unicode.Ll, unicode.Lm, unicode.Lo, unicode.Nd):
			if gap && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			inWord, gap = true, false
		case unicode.In(r, unicode.Mn, unicode.Mc):
			if inWord {
				b.WriteRune(r)
			}
		default:
			inWord, gap = false, true
		}
	}
	words := strings.Split(norm.NFC.String(b.String()), "-")
	for i, w := range words {
		rs := []rune(w)
		if n := len(rs); n >= 2 && rs[n-1] == 'σ' && unicode.IsLetter(rs[n-2]) {
			rs[n-1] = 'ς'
			words[i] = string(rs)
		}
	}
	return strings.Join(words, "-")
}

func tagIgnorable(r rune) bool {
	switch {
	case unicode.In(r, unicode.Other_Default_Ignorable_Code_Point, unicode.Variation_Selector):
		return true
	case !unicode.Is(unicode.Cf, r):
		return false
	case unicode.In(r, unicode.White_Space, unicode.Prepended_Concatenation_Mark):
		return false
	case r >= 0xfff9 && r <= 0xfffb, r >= 0x13430 && r <= 0x1343f:
		return false
	}
	return true
}
