package migrations

import (
	"slices"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// subjectKind has the names of the collections and the tables of one kind
// of thing that users review. The names are a copy of the subjects package.
// A migration does not change when the code changes.
type subjectKind struct {
	parent    string
	parents   string
	reviews   string
	photos    string
	listItems string
	versions  string
	// reviewTriggers is the name prefix of the triggers on the reviews.
	reviewTriggers string
	// searchColumn is the last column of the FTS5 table of the parents.
	searchColumn string
	scores       []string
	newParent    func(users *core.Collection) *core.Collection
}

var (
	restaurantKind = subjectKind{
		parent: "restaurant", parents: "restaurants", reviews: "reviews", photos: "review_photos",
		listItems: "list_items", versions: "restaurant_versions", reviewTriggers: "review_overall",
		searchColumn: "address", scores: []string{"space", "service", "food", "price"}, newParent: newRestaurants,
	}
	recipeKind = subjectKind{
		parent: "recipe", parents: "recipes", reviews: "recipe_reviews", photos: "recipe_review_photos",
		listItems: "list_recipes", versions: "recipe_versions", reviewTriggers: "recipe_review_overall",
		searchColumn: "ingredients", scores: []string{"taste", "ease", "instructions", "value"}, newParent: newRecipes,
	}
	subjectKinds = []subjectKind{restaurantKind, recipeKind}
)

func (k subjectKind) create(app core.App, users, lists *core.Collection) error {
	parent := k.newParent(users)
	if err := app.Save(parent); err != nil {
		return err
	}
	reviews := k.newReviews(users, parent)
	for _, c := range []*core.Collection{
		reviews, k.newPhotos(reviews), k.newListItems(lists, parent), k.newVersions(users, parent),
	} {
		if err := app.Save(c); err != nil {
			return err
		}
	}
	return execAll(app, k.tables()...)
}

// drop deletes the tables of the collections first. Their triggers then go
// with them.
func (k subjectKind) drop(app core.App) error {
	for _, name := range []string{k.versions, k.listItems, k.photos, k.reviews, k.parents} {
		c, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return err
		}
		if err := app.Delete(c); err != nil {
			return err
		}
	}
	return execAll(app,
		"DROP TABLE {{"+k.overallTable()+"}}",
		"DROP TABLE {{"+k.tagsTable()+"}}",
		"DROP TABLE {{"+k.parents+"_fts}}",
		"DROP TABLE {{"+k.reviews+"_fts}}",
	)
}

func (k subjectKind) overallTable() string {
	return k.parent + "_overall"
}

func (k subjectKind) tagsTable() string {
	return k.parent + "_tags"
}

// tables makes the SQL tables of decision 00005 and the FTS5 tables of
// decision 00002. Triggers keep the tables of decision 00005 equal to the
// records. The searchindex package keeps the FTS5 tables equal to them.
func (k subjectKind) tables() []string {
	overall, tags, parent := k.overallTable(), k.tagsTable(), "[["+k.parent+"]]"
	scoreColumns := ""
	for _, name := range k.scores {
		scoreColumns += "[[" + name + "]] REAL, "
	}
	sortColumns := "[[overall_or_0]], [[reviewers]], " + parent
	return slices.Concat([]string{
		// The overall column of the reviews is not a field of the collection.
		// A migration that drops a score column must drop it first.
		"ALTER TABLE {{" + k.reviews + "}} ADD COLUMN [[overall]] REAL AS " + k.overallColumn(),
		"CREATE INDEX {{idx_" + k.reviews + "_" + k.parent + "_overall}} ON {{" + k.reviews + "}} (" + parent + ", [[overall]], [[id]])",
		"CREATE INDEX {{idx_" + k.reviews + "_creator_overall}} ON {{" + k.reviews + "}} ([[creator]], [[overall]], [[id]])",
		"CREATE INDEX {{idx_" + k.reviews + "_overall}} ON {{" + k.reviews + "}} ([[overall]], [[id]])",

		// SQLite uses an index on columns for a row value range. It does not
		// use an index on expressions for it.
		"CREATE TABLE {{" + overall + "}} (" + parent + " TEXT PRIMARY KEY NOT NULL, [[overall]] REAL, " +
			"[[reviewers]] INTEGER NOT NULL DEFAULT 0, [[review_count]] INTEGER NOT NULL DEFAULT 0, " + scoreColumns +
			"[[unrated]] INTEGER AS ([[overall]] IS NULL), [[rated]] INTEGER AS ([[overall]] IS NOT NULL), " +
			"[[overall_or_0]] REAL AS (COALESCE([[overall]], 0))) WITHOUT ROWID",
		"CREATE INDEX {{idx_" + overall + "_asc}} ON {{" + overall + "}} ([[unrated]], " + sortColumns + ")",
		"CREATE INDEX {{idx_" + overall + "_desc}} ON {{" + overall + "}} ([[rated]], " + sortColumns + ")",
		"CREATE INDEX {{idx_" + overall + "_value}} ON {{" + overall + "}} ([[overall]])",
		"CREATE TABLE {{" + tags + "}} ([[tag]] TEXT NOT NULL, " + parent + " TEXT NOT NULL, PRIMARY KEY ([[tag]], " + parent + ")) WITHOUT ROWID",
		"CREATE INDEX {{idx_" + tags + "_" + k.parent + "}} ON {{" + tags + "}} (" + parent + ")",
	}, k.parentTriggers(), []string{
		"CREATE TRIGGER {{" + k.reviewTriggers + "_insert}} AFTER INSERT ON {{" + k.reviews + "}} BEGIN " + k.updateOverall("NEW") + " END",
		"CREATE TRIGGER {{" + k.reviewTriggers + "_update}} AFTER UPDATE OF " + strings.Join(k.watchedColumns(), ", ") +
			" ON {{" + k.reviews + "}} WHEN " + anyChanged(k.watchedColumns()) + " BEGIN " + k.updateOverall("NEW") + " " + k.updateOverall("OLD") + " END",
		"CREATE TRIGGER {{" + k.reviewTriggers + "_delete}} AFTER DELETE ON {{" + k.reviews + "}} BEGIN " + k.updateOverall("OLD") + " END",

		// The tokenizer ignores case and accents. The prefix indexes serve
		// prefix terms.
		"CREATE VIRTUAL TABLE {{" + k.parents + "_fts}} USING fts5(id UNINDEXED, name, description, tags, " + k.searchColumn + ", " +
			"tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
		"CREATE VIRTUAL TABLE {{" + k.reviews + "_fts}} USING fts5(id UNINDEXED, title, feedback, " + k.parent + "_name, " +
			"tokenize = 'unicode61 remove_diacritics 2', prefix = '2 3')",
	})
}

// parentTriggers keep the overall row and the tag rows of each parent.
func (k subjectKind) parentTriggers() []string {
	overall, tags, parent := k.overallTable(), k.tagsTable(), "[["+k.parent+"]]"
	return []string{
		"CREATE TRIGGER {{" + k.parent + "_index_insert}} AFTER INSERT ON {{" + k.parents + "}} BEGIN " +
			"INSERT INTO {{" + overall + "}} (" + parent + ", " + k.overallColumns() + ") " + k.byReviewer("NEW.[[id]], ", "NEW.[[id]]") + "; " +
			k.insertTags("NEW") + " END",
		"CREATE TRIGGER {{" + k.parent + "_index_update}} AFTER UPDATE OF [[tags]] ON {{" + k.parents + "}} WHEN OLD.[[tags]] IS NOT NEW.[[tags]] BEGIN " +
			"DELETE FROM {{" + tags + "}} WHERE " + parent + " = OLD.[[id]]; " +
			k.insertTags("NEW") + " END",
		"CREATE TRIGGER {{" + k.parent + "_index_delete}} AFTER DELETE ON {{" + k.parents + "}} BEGIN " +
			"DELETE FROM {{" + overall + "}} WHERE " + parent + " = OLD.[[id]]; " +
			"DELETE FROM {{" + tags + "}} WHERE " + parent + " = OLD.[[id]]; END",
	}
}

// dropParentTriggers lets a migration drop the tags column. SQLite refuses
// the drop of a column that a trigger uses.
func (k subjectKind) dropParentTriggers() []string {
	out := make([]string, 0, 3)
	for _, event := range []string{"insert", "update", "delete"} {
		out = append(out, "DROP TRIGGER {{"+k.parent+"_index_"+event+"}}")
	}
	return out
}

// rebuildTags fills the tag table of k again from the tags of each parent.
func (k subjectKind) rebuildTags() []string {
	tags := "{{" + k.parents + "}}.[[tags]]"
	return []string{
		"DELETE FROM {{" + k.tagsTable() + "}}",
		"INSERT OR IGNORE INTO {{" + k.tagsTable() + "}} ([[tag]], [[" + k.parent + "]]) SELECT [[value]], {{" + k.parents + "}}.[[id]] FROM {{" + k.parents + "}}, " +
			"json_each(CASE WHEN json_valid(" + tags + ") AND json_type(" + tags + ") = 'array' THEN " + tags + " ELSE '[]' END) " +
			"WHERE [[type]] = 'text'",
	}
}

func (k subjectKind) scoreSum() string {
	return "[[" + strings.Join(k.scores, "]] + [[") + "]]"
}

// overallColumn is the generated overall column of a review. The division in
// a double gives the value of the responses.
func (k subjectKind) overallColumn() string {
	return "((" + k.scoreSum() + ") / " + k.scoreCount() + ".0)"
}

func (k subjectKind) scoreCount() string {
	return strconv.Itoa(len(k.scores))
}

// overallColumns are the stored values of byReviewer in the same order.
func (k subjectKind) overallColumns() string {
	return "[[overall]], [[reviewers]], [[review_count]], [[" + strings.Join(k.scores, "]], [[") + "]]"
}

// byReviewer selects the values of overallColumns for one parent. Each
// creator counts once with the average of the own reviews.
func (k subjectKind) byReviewer(first, parent string) string {
	outer := []string{"ROUND(AVG([[reviewer_overall]]), 1)", "COUNT(*)", "COALESCE(SUM([[reviewer_reviews]]), 0)"}
	inner := []string{
		"SUM(" + k.scoreSum() + ") * 1.0 / (" + k.scoreCount() + " * COUNT(*)) AS [[reviewer_overall]]",
		"COUNT(*) AS [[reviewer_reviews]]",
	}
	for _, name := range k.scores {
		outer = append(outer, "ROUND(AVG([["+name+"]]), 1)")
		inner = append(inner, "AVG([["+name+"]]) AS [["+name+"]]")
	}
	return "SELECT " + first + strings.Join(outer, ", ") + " FROM (SELECT " + strings.Join(inner, ", ") +
		" FROM {{" + k.reviews + "}} WHERE [[" + k.reviews + "." + k.parent + "]] = " + parent + " GROUP BY [[creator]])"
}

// updateOverall changes no row for a deleted parent.
func (k subjectKind) updateOverall(row string) string {
	parent := row + ".[[" + k.parent + "]]"
	return "UPDATE {{" + k.overallTable() + "}} SET (" + k.overallColumns() + ") = (" + k.byReviewer("", parent) + ") " +
		"WHERE [[" + k.parent + "]] = " + parent + ";"
}

// watchedColumns has the creator too. A change of the creator moves a
// review to another reviewer.
func (k subjectKind) watchedColumns() []string {
	columns := []string{"[[" + k.parent + "]]"}
	for _, name := range k.scores {
		columns = append(columns, "[["+name+"]]")
	}
	return append(columns, "[[creator]]")
}

// insertTags keeps only the strings of a JSON array. The API accepts only
// arrays of strings. The dashboard accepts all JSON.
func (k subjectKind) insertTags(row string) string {
	tags := row + ".[[tags]]"
	return "INSERT OR IGNORE INTO {{" + k.tagsTable() + "}} ([[tag]], [[" + k.parent + "]]) SELECT [[value]], " + row + ".[[id]] FROM " +
		"json_each(CASE WHEN json_valid(" + tags + ") AND json_type(" + tags + ") = 'array' THEN " + tags + " ELSE '[]' END) " +
		"WHERE [[type]] = 'text';"
}

func anyChanged(columns []string) string {
	parts := make([]string, len(columns))
	for i, c := range columns {
		parts[i] = "OLD." + c + " IS NOT NEW." + c
	}
	return strings.Join(parts, " OR ")
}

func execAll(app core.App, queries ...string) error {
	for _, q := range queries {
		if _, err := app.DB().NewQuery(q).Execute(); err != nil {
			return err
		}
	}
	return nil
}
