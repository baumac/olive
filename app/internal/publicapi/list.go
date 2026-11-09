package publicapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	defaultSort  = "-created"
)

var comparisonOps = []struct{ name, sql string }{
	{"lt", "<"}, {"lte", "<="}, {"eq", "="}, {"gte", ">="}, {"gt", ">"},
}

// The array filters have a maximum number of values. Each value is one SQL
// parameter. A tag array has at most maxTags items.
const (
	maxTagValues   = maxTags
	maxOtherValues = 100
)

func withParams(query, params map[string]int) map[string]int {
	out := maps.Clone(query)
	maps.Copy(out, params)
	return out
}

func withRanges(query map[string]int, names ...string) map[string]int {
	ranges := map[string]int{}
	for _, name := range names {
		for _, op := range comparisonOps {
			ranges[name+"["+op.name+"]"] = 1
		}
	}
	return withParams(query, ranges)
}

var listQuery = withRanges(map[string]int{
	"limit": 1, "cursor": 1, "sort": 1, "search": 1,
	"tags[has]": 1, "tags[has_any]": maxTagValues, "tags[has_all]": maxTagValues,
	"creator[eq]": 1, "creator[in]": maxOtherValues,
}, "created", "updated")

var restaurantQuery = withRanges(listQuery, "overall")

var reviewQuery = withRanges(listQuery, "space", "service", "food", "price", "cleanliness")

// allReviewQuery adds the restaurant filters of GET /v1/reviews.
var allReviewQuery = withParams(reviewQuery, map[string]int{"restaurant[eq]": 1, "restaurant[in]": maxOtherValues})

// sortKey is one ORDER BY term. The page query also selects it for the
// cursor.
type sortKey struct {
	expr string
	desc bool
}

type sortField func(desc bool) []sortKey

func column(table, name string) string {
	return "[[" + table + "." + name + "]]"
}

// textSort ignores case only for ASCII letters. This is the SQLite NOCASE
// collation.
func textSort(table, name string) sortField {
	return func(desc bool) []sortKey {
		return []sortKey{{column(table, name) + " COLLATE NOCASE", desc}}
	}
}

func columnSort(table, name string) sortField {
	return func(desc bool) []sortKey {
		return []sortKey{{column(table, name), desc}}
	}
}

// parseSort returns the sort keys and the sort value that the cursor uses.
// The last key is always id with the direction of the key before it. SQLite
// can then read an index on the column and id in one direction.
func parseSort(raw *string, table string, fields map[string]sortField) ([]sortKey, string, string) {
	spec := defaultSort
	if raw != nil {
		spec = *raw
	}
	seen := map[string]bool{}
	var keys []sortKey
	for part := range strings.SplitSeq(spec, ",") {
		name, desc := strings.CutPrefix(part, "-")
		field, ok := fields[name]
		if !ok {
			return nil, "", fmt.Sprintf("The sort property %q is not supported.", name)
		}
		if seen[name] {
			return nil, "", fmt.Sprintf("The sort property %q occurs more than one time.", name)
		}
		seen[name] = true
		keys = append(keys, field(desc)...)
	}
	keys = append(keys, sortKey{column(table, "id"), keys[len(keys)-1].desc})
	return keys, spec, ""
}

func parseLimit(raw *int) (int, string) {
	if raw == nil {
		return defaultLimit, ""
	}
	if *raw < 1 || *raw > maxLimit {
		return 0, fmt.Sprintf("limit must be between 1 and %d.", maxLimit)
	}
	return *raw, ""
}

// where collects the conditions and joins of a list query. All conditions
// use one set of parameters.
type where struct {
	parts  []string
	params dbx.Params
	joins  []join
}

// join is a LEFT JOIN, or an INNER JOIN that also filters the rows.
type join struct {
	inner bool
	table string
	on    dbx.Expression
}

func newWhere() *where {
	return &where{params: dbx.Params{}}
}

func (w *where) param(v any) string {
	name := fmt.Sprintf("p%d", len(w.params))
	w.params[name] = v
	return "{:" + name + "}"
}

func (w *where) add(condition string) {
	w.parts = append(w.parts, condition)
}

func (w *where) expression() dbx.Expression {
	if len(w.parts) == 0 {
		return nil
	}
	return dbx.NewExp(strings.Join(w.parts, " AND "), w.params)
}

type comparison struct {
	op    string
	value any
}

func comparisons[T any](lt, lte, eq, gte, gt *T) []comparison {
	var out []comparison
	for i, v := range []*T{lt, lte, eq, gte, gt} {
		if v != nil {
			out = append(out, comparison{comparisonOps[i].sql, *v})
		}
	}
	return out
}

func (w *where) compare(expr string, cs []comparison) {
	for _, c := range cs {
		w.add(expr + " " + c.op + " " + w.param(c.value))
	}
}

func (w *where) in(expr string, values []string) {
	ps := make([]string, len(values))
	for i, v := range values {
		ps[i] = w.param(v)
	}
	w.add(expr + " IN (" + strings.Join(ps, ", ") + ")")
}

// hasTags adds a condition that the tags have at least one of tags. It
// compares whole items with json_each. The PocketBase ~ operator compares
// parts of the JSON text.
func (w *where) hasTags(table string, tags []string) {
	ps := make([]string, len(tags))
	for i, t := range tags {
		ps[i] = w.param(t)
	}
	c := column(table, "tags")
	w.add(fmt.Sprintf("EXISTS (SELECT 1 FROM json_each(CASE WHEN json_valid(%s) THEN %s ELSE '[]' END) WHERE value IN (%s))",
		c, c, strings.Join(ps, ", ")))
}

func (w *where) tagFilters(table string, has *string, hasAny, hasAll *[]string) string {
	all := []string{}
	if has != nil {
		all = append(all, *has)
	}
	if hasAll != nil {
		all = append(all, *hasAll...)
	}
	for _, tag := range all {
		if !validTag(tag) {
			return fmt.Sprintf("The tag %q is not valid.", tag)
		}
		w.hasTags(table, []string{tag})
	}
	if hasAny != nil {
		for _, tag := range *hasAny {
			if !validTag(tag) {
				return fmt.Sprintf("The tag %q is not valid.", tag)
			}
		}
		w.hasTags(table, *hasAny)
	}
	return ""
}

func (w *where) creatorFilters(table string, eq *string, in *[]string) string {
	ids := []string{}
	if eq != nil {
		ids = append(ids, *eq)
	}
	if in != nil {
		ids = append(ids, *in...)
	}
	for _, id := range ids {
		if !recordIDPattern.MatchString(id) {
			return fmt.Sprintf("The creator ID %q is not valid.", id)
		}
	}
	if eq != nil {
		w.compare(column(table, "creator"), []comparison{{"=", *eq}})
	}
	if in != nil {
		w.in(column(table, "creator"), *in)
	}
	return ""
}

func (w *where) timestampFilter(table, name string, cs []comparison) string {
	for _, c := range cs {
		if !timestampPattern.MatchString(c.value.(string)) {
			return fmt.Sprintf("The %s timestamp %q is not valid.", name, c.value)
		}
	}
	w.compare(column(table, name), cs)
	return ""
}

// keyset adds the condition that selects the rows after the cursor values
// in the sort order. Keys with one direction use a row value comparison.
// SQLite can use an index for it.
func (w *where) keyset(keys []sortKey, after []any) {
	if slices.IndexFunc(keys, func(k sortKey) bool { return k.desc != keys[0].desc }) < 0 {
		exprs := make([]string, len(keys))
		ps := make([]string, len(keys))
		for i, k := range keys {
			exprs[i], ps[i] = k.expr, w.param(after[i])
		}
		op := ">"
		if keys[0].desc {
			op = "<"
		}
		w.add("((" + strings.Join(exprs, ", ") + ") " + op + " (" + strings.Join(ps, ", ") + "))")
		return
	}
	var or []string
	for i, k := range keys {
		var and []string
		for j := range i {
			and = append(and, keys[j].expr+" = "+w.param(after[j]))
		}
		op := ">"
		if k.desc {
			op = "<"
		}
		and = append(and, k.expr+" "+op+" "+w.param(after[i]))
		or = append(or, "("+strings.Join(and, " AND ")+")")
	}
	w.add("(" + strings.Join(or, " OR ") + ")")
}

type cursorData struct {
	Query  string `json:"q"`
	Values []any  `json:"v"`
}

// fingerprint identifies the filters and the sort of a list request. A
// cursor is valid only for the same fingerprint.
func fingerprint(parts ...any) (string, error) {
	b, err := json.Marshal(parts)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8]), nil
}

func encodeCursor(query string, values []any) (string, error) {
	b, err := json.Marshal(cursorData{query, values})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeCursor(raw *string, query string, n int) ([]any, string) {
	if raw == nil {
		return nil, ""
	}
	const invalid = "The cursor is not valid for this request. Use the same filters and sort that made it."
	b, err := base64.RawURLEncoding.DecodeString(*raw)
	if err != nil {
		return nil, invalid
	}
	var c cursorData
	if json.Unmarshal(b, &c) != nil || c.Query != query || len(c.Values) != n {
		return nil, invalid
	}
	for _, v := range c.Values {
		switch v.(type) {
		case string, float64:
		default:
			return nil, invalid
		}
	}
	return c.Values, ""
}

// page gets one page of records with one more row than limit for the next
// page check. The cursor values come from the query that sorts the page. A
// write during the request then cannot move the cursor.
func (s *Server) page(collection string, w *where, keys []sortKey, limit int, query string) ([]*core.Record, nullable.Nullable[string], error) {
	rows, err := s.sortValues(collection, w, keys, limit+1)
	if err != nil {
		return nil, nil, err
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		id, ok := row[len(row)-1].(string)
		if !ok {
			return nil, nil, fmt.Errorf("the id sort value %v is not a string", row[len(row)-1])
		}
		ids[i] = id
	}
	records, err := s.recordsInOrder(collection, ids)
	if err != nil {
		return nil, nil, err
	}

	next := nullable.NewNullNullable[string]()
	if more {
		cursor, err := encodeCursor(query, rows[limit-1])
		if err != nil {
			return nil, nil, err
		}
		next = nullable.NewNullableWithValue(cursor)
	}
	return records, next, nil
}

// sortValues gets the values of the sort keys for at most n rows in the sort
// order. The last value of a row is the record ID.
func (s *Server) sortValues(collection string, w *where, keys []sortKey, n int) ([][]any, error) {
	columns := make([]string, len(keys))
	for i, k := range keys {
		columns[i] = fmt.Sprintf("%s AS [[k%d]]", k.expr, i)
	}
	q := s.app.DB().Select(columns...).From(collection)
	for _, j := range w.joins {
		if j.inner {
			q.InnerJoin(j.table, j.on)
		} else {
			q.LeftJoin(j.table, j.on)
		}
	}
	// A join can have parameters without a condition in parts.
	q.Bind(w.params)
	if e := w.expression(); e != nil {
		q.AndWhere(e)
	}
	for _, k := range keys {
		dir := " ASC"
		if k.desc {
			dir = " DESC"
		}
		q.AndOrderBy(k.expr + dir)
	}
	q.Limit(int64(n))

	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]any
	for rows.Next() {
		values := make([]any, len(keys))
		ptrs := make([]any, len(keys))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		out = append(out, values)
	}
	return out, rows.Err()
}

// recordsInOrder gets the records in the order of ids. A record that is
// deleted after the sort query is not in the result.
func (s *Server) recordsInOrder(collection string, ids []string) ([]*core.Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	found, err := s.app.FindRecordsByIds(collection, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*core.Record, len(found))
	for _, r := range found {
		byID[r.Id] = r
	}
	out := make([]*core.Record, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Server) creators(records ...*core.Record) (map[string]*core.Record, error) {
	ids := map[string]bool{}
	for _, r := range records {
		if id := r.GetString("creator"); id != "" {
			ids[id] = true
		}
	}
	if len(ids) == 0 {
		return map[string]*core.Record{}, nil
	}
	users, err := s.app.FindRecordsByIds("users", slices.Collect(maps.Keys(ids)))
	if err != nil {
		return nil, err
	}
	out := make(map[string]*core.Record, len(users))
	for _, u := range users {
		out[u.Id] = u
	}
	return out, nil
}
