package publicapi

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/baumac/reviews-app/app/internal/searchindex"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const maxSearchWords = 10

// scoreColumn is an index column and the score of a match that has all
// search words in it.
type scoreColumn struct {
	name   string
	weight int
}

// The scores depend only on the row. bm25 also depends on all other rows.
// A write then moves the bm25 ranks under an open cursor.
var (
	restaurantScores = []scoreColumn{{"name", 10}, {"tags", 4}, {"description", 1}}
	reviewScores     = []scoreColumn{{"title", 10}, {"tags", 4}, {"restaurant_name", 2}, {"feedback", 1}}
)

// ftsQuery changes a search into an FTS5 query. Each word becomes a quoted
// term. FTS5 operators in the search are then plain text. A word of two or
// more characters is a prefix term.
func ftsQuery(search string) (string, string) {
	if n := utf8.RuneCountInString(search); n < 1 || n > 255 {
		return "", "The search must have 1 to 255 characters."
	}
	var terms []string
	for _, word := range strings.Fields(search) {
		if !strings.ContainsFunc(word, isWordChar) {
			continue
		}
		term := `"` + strings.ReplaceAll(word, `"`, `""`) + `"`
		// The prefix indexes cover 2 and 3 characters. A one-character
		// prefix reads all index terms with that character.
		if countFunc(word, isWordChar) > 1 {
			term += "*"
		}
		terms = append(terms, term)
	}
	switch {
	case len(terms) == 0:
		return "", "The search must have a letter or a digit."
	case len(terms) > maxSearchWords:
		return "", fmt.Sprintf("The search can have at most %d words.", maxSearchWords)
	}
	return strings.Join(terms, " "), ""
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func countFunc(s string, f func(rune) bool) int {
	n := 0
	for _, r := range s {
		if f(r) {
			n++
		}
	}
	return n
}

// searchPage has the search of one list request and the scores of one page.
// The relevance sort reads the cursor values from it.
type searchPage struct {
	s       *Server
	table   string
	fts     string
	columns []scoreColumn
	query   string
	sorted  bool
	scores  map[string]int
}

func newSearchPage(s *Server, table string) *searchPage {
	p := &searchPage{s: s, table: table, fts: searchindex.Restaurants, columns: restaurantScores}
	if table == "reviews" {
		p.fts, p.columns = searchindex.Reviews, reviewScores
	}
	return p
}

// parse returns a detail for the 400 response. A nil search is no search.
func (p *searchPage) parse(search *string) string {
	if search == nil {
		return ""
	}
	query, detail := ftsQuery(*search)
	p.query = query
	return detail
}

func (p *searchPage) active() bool {
	return p.query != ""
}

// hits selects the matches with their score. The score adds the weights of
// the columns that have all search words. Each column subquery runs once.
func (p *searchPage) hits(w *where) string {
	parts := make([]string, len(p.columns))
	for i, c := range p.columns {
		parts[i] = fmt.Sprintf("(CASE WHEN [[id]] IN (SELECT [[id]] FROM {{%s}} WHERE {{%s}} MATCH %s) THEN %d ELSE 0 END)",
			p.fts, p.fts, w.param(c.name+" : ("+p.query+")"), c.weight)
	}
	return fmt.Sprintf("(SELECT [[id]], %s AS [[score]] FROM {{%s}} WHERE {{%s}} MATCH %s)",
		strings.Join(parts, " + "), p.fts, p.fts, w.param(p.query))
}

// filter keeps only the matches. It does nothing without a search.
func (p *searchPage) filter(w *where) {
	if !p.active() {
		return
	}
	w.joins = append(w.joins, join{
		inner: true,
		table: p.hits(w) + " [[search_hits]]",
		on:    dbx.NewExp("[[search_hits.id]] = " + column(p.table, "id")),
	})
}

// relevanceSort puts the highest score first. -relevance puts it last.
func (p *searchPage) relevanceSort(desc bool) []sortKey {
	p.sorted = true
	return []sortKey{{"[[search_hits.score]]", !desc, func(r *core.Record) any { return p.scores[r.Id] }}}
}

func (p *searchPage) load(records []*core.Record) error {
	if !p.sorted || len(records) == 0 {
		return nil
	}
	ids := make([]any, len(records))
	for i, r := range records {
		ids[i] = r.Id
	}
	w := newWhere()
	var rows []struct {
		ID    string `db:"id"`
		Score int    `db:"score"`
	}
	err := p.s.app.DB().
		Select("[[id]]", "[[score]]").
		From(p.hits(w) + " [[search_hits]]").
		Where(dbx.In("id", ids...)).
		Bind(w.params).
		All(&rows)
	if err != nil {
		return err
	}
	p.scores = make(map[string]int, len(rows))
	for _, row := range rows {
		p.scores[row.ID] = row.Score
	}
	return nil
}

// sortWithSearch gives the sort to parse. Without a sort, a search sorts by
// relevance. A relevance sort without a search returns a detail for 400.
func (p *searchPage) sortWithSearch(sort *string) (*string, string) {
	if sort == nil && p.active() {
		relevance := "relevance"
		return &relevance, ""
	}
	if sort != nil && sortsBy(*sort, "relevance") && !p.active() {
		return nil, "The relevance sort needs a search."
	}
	return sort, ""
}

func loadAll(loads ...func([]*core.Record) error) func([]*core.Record) error {
	return func(records []*core.Record) error {
		for _, load := range loads {
			if err := load(records); err != nil {
				return err
			}
		}
		return nil
	}
}
