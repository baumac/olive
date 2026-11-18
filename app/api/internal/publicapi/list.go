package publicapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	reviewsapi "github.com/baumac/reviews-app/app/sdk/go"
	"github.com/oapi-codegen/nullable"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	defaultLimit = 20
	maxLimit     = 100
	defaultSort  = "-created"
)

var comparisonOps = []string{"<", "<=", "=", ">=", ">"}

// sortKey is one ORDER BY term. The page query also selects it for the
// cursor. A unique key orders all rows. parseSort adds no key after it.
type sortKey struct {
	expr   string
	desc   bool
	unique bool
}

type sortField func(desc bool) []sortKey

func column(table, name string) string {
	return "[[" + table + "." + name + "]]"
}

// textSort ignores case only for ASCII letters. This is the SQLite NOCASE
// collation.
func textSort(table, name string) sortField {
	return func(desc bool) []sortKey {
		return []sortKey{{expr: column(table, name) + " COLLATE NOCASE", desc: desc}}
	}
}

func columnSort(table, name string) sortField {
	return func(desc bool) []sortKey {
		return []sortKey{{expr: column(table, name), desc: desc}}
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
		if len(keys) == 0 || !keys[len(keys)-1].unique {
			keys = append(keys, field(desc)...)
		}
	}
	if last := keys[len(keys)-1]; !last.unique {
		keys = append(keys, sortKey{expr: column(table, "id"), desc: last.desc, unique: true})
	}
	return keys, spec, ""
}

// limitOf keeps the limit in the range of the spec. The spec validation can
// be off. A limit below 1 would then break the cursor of page.
func limitOf(raw *int) int {
	if raw == nil {
		return defaultLimit
	}
	return min(max(*raw, 1), maxLimit)
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
			out = append(out, comparison{comparisonOps[i], *v})
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

// hasTags adds a condition that the parent has at least one of tags.
func (w *where) hasTags(sub subject, tags []string) {
	ps := make([]string, len(tags))
	for i, t := range tags {
		ps[i] = w.param(t)
	}
	w.add(fmt.Sprintf("%s IN (SELECT [[%s]] FROM {{%s}} WHERE [[tag]] IN (%s))",
		column(sub.Table, "id"), sub.Parent, sub.Tags, strings.Join(ps, ", ")))
}

func (w *where) tagFilters(sub subject, has *string, hasAny, hasAll *[]string) {
	all := []string{}
	if has != nil {
		all = append(all, *has)
	}
	if hasAll != nil {
		all = append(all, *hasAll...)
	}
	for _, tag := range all {
		w.hasTags(sub, []string{tag})
	}
	if hasAny != nil {
		w.hasTags(sub, *hasAny)
	}
}

// metersPerDegree is the length of one degree of latitude.
const metersPerDegree = 111320.0

// squaredDistance is the SQL of the squared distance in meters from a point
// to a restaurant, on a flat surface. The longitude difference is the short
// way around the earth. The arguments are SQL values.
func squaredDistance(latitude, longitude, metersPerLatitude, metersPerLongitude string) string {
	lat, lon := column("restaurants", "latitude"), column("restaurants", "longitude")
	dy := fmt.Sprintf("((%s - %s) * %s)", lat, latitude, metersPerLatitude)
	dx := fmt.Sprintf("((180 - abs(abs(%s - %s) - 180)) * %s)", lon, longitude, metersPerLongitude)
	return fmt.Sprintf("(%s * %s + %s * %s)", dy, dy, dx, dx)
}

func metersPerLongitude(latitude float64) float64 {
	return metersPerDegree * math.Cos(latitude*math.Pi/180)
}

// near keeps the restaurants at most the radius from the point. Without a
// radius, it keeps all restaurants.
func (w *where) near(n *reviewsapi.NearFilter) {
	if n == nil || n.Radius == nil {
		return
	}
	lat := column("restaurants", "latitude")
	radius := float64(*n.Radius)
	reach := radius / metersPerDegree
	w.add(column("restaurants", "address") + " != ''")
	w.add(fmt.Sprintf("%s BETWEEN %s AND %s", lat, w.param(n.Latitude-reach), w.param(n.Latitude+reach)))
	distance := squaredDistance(w.param(n.Latitude), w.param(n.Longitude), w.param(metersPerDegree), w.param(metersPerLongitude(n.Latitude)))
	w.add(fmt.Sprintf("%s <= %s", distance, w.param(radius*radius)))
}

// distanceSort sorts by the squared distance from the point of n.
// Restaurants without a location are last in both directions. The SQL text
// has the numbers, not parameters.
func distanceSort(n *reviewsapi.NearFilter) sortField {
	sql := func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
	expr := squaredDistance(sql(n.Latitude), sql(n.Longitude), sql(metersPerDegree), sql(metersPerLongitude(n.Latitude)))
	return func(desc bool) []sortKey {
		return []sortKey{{expr: "(" + column("restaurants", "address") + " = '')"}, {expr: expr, desc: desc}}
	}
}

// within keeps the restaurants in the box. A box with west after east goes
// across the 180th meridian.
func (w *where) within(b *reviewsapi.WithinFilter) {
	if b == nil {
		return
	}
	lat, lon := column("restaurants", "latitude"), column("restaurants", "longitude")
	w.add(column("restaurants", "address") + " != ''")
	w.add(fmt.Sprintf("%s BETWEEN %s AND %s", lat, w.param(b.South), w.param(b.North)))
	if b.West <= b.East {
		w.add(fmt.Sprintf("%s BETWEEN %s AND %s", lon, w.param(b.West), w.param(b.East)))
		return
	}
	w.add(fmt.Sprintf("(%s >= %s OR %s <= %s)", lon, w.param(b.West), lon, w.param(b.East)))
}

// idFilters adds the eq and in filters of an ID column.
func (w *where) idFilters(table, name string, eq *string, in *[]string) {
	if eq != nil {
		w.compare(column(table, name), []comparison{{"=", *eq}})
	}
	if in != nil {
		w.in(column(table, name), *in)
	}
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

// page gets the records of one page in the order of pageIDs.
func (s *Server) page(collection string, w *where, keys []sortKey, limit int, query string) ([]*core.Record, nullable.Nullable[string], error) {
	ids, next, err := s.pageIDs(collection, w, keys, limit, query)
	if err != nil {
		return nil, nil, err
	}
	records, err := s.recordsInOrder(collection, ids)
	if err != nil {
		return nil, nil, err
	}
	return records, next, nil
}

// pageIDs reads one more row than limit for the next page check. The cursor
// values come from the query that sorts the page. A write during the request
// then cannot move the cursor.
func (s *Server) pageIDs(collection string, w *where, keys []sortKey, limit int, query string) ([]string, nullable.Nullable[string], error) {
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

	next := nullable.NewNullNullable[string]()
	if more {
		cursor, err := encodeCursor(query, rows[limit-1])
		if err != nil {
			return nil, nil, err
		}
		next = nullable.NewNullableWithValue(cursor)
	}
	return ids, next, nil
}

// sortValues gets the values of the sort keys for at most n rows in the sort
// order. The last value of a row is the record ID.
func (s *Server) sortValues(collection string, w *where, keys []sortKey, n int) ([][]any, error) {
	rows, err := s.sortQuery(collection, w, keys, n).Rows()
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

func (s *Server) sortQuery(collection string, w *where, keys []sortKey, n int) *dbx.SelectQuery {
	columns := make([]string, len(keys))
	for i, k := range keys {
		columns[i] = fmt.Sprintf("%s AS [[k%d]]", k.expr, i)
	}
	q := s.selectWhere(collection, w, columns...)
	for _, term := range orderTerms(keys) {
		q.AndOrderBy(term)
	}
	return q.Limit(int64(n))
}

// selectWhere selects the columns from the rows that match the joins and
// the conditions of w.
func (s *Server) selectWhere(collection string, w *where, columns ...string) *dbx.SelectQuery {
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
	return q
}

func orderTerms(keys []sortKey) []string {
	terms := make([]string, len(keys))
	for i, k := range keys {
		terms[i] = k.expr + " ASC"
		if k.desc {
			terms[i] = k.expr + " DESC"
		}
	}
	return terms
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
