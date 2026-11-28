package webui

import (
	"bytes"
	"database/sql"
	"errors"
	"html"
	"html/template"
	"io/fs"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/baumac/olive/app/api/internal/publicapi"
	"github.com/baumac/olive/app/api/internal/subjects"
	"github.com/baumac/olive/app/api/internal/tagname"
	reviewsapi "github.com/baumac/olive/specs/api/sdk/server/go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/text/unicode/norm"
)

// The page-meta block of index.html starts and ends with these comments.
// Decision 00002 has the details.
const (
	metaStart = "<!-- page-meta -->"
	metaEnd   = "<!-- /page-meta -->"
)

// maxDescriptionRunes is near the length that search results show.
const maxDescriptionRunes = 200

// recordID is the RecordId pattern of the API spec. The path
// /restaurants/new then does not match a restaurant.
var recordID = regexp.MustCompile(`^[a-z0-9]{15}$`)

var (
	titleTag = regexp.MustCompile(`(?is)<title[^>]*>([^<]*)</title>`)
	metaTag  = regexp.MustCompile(`(?is)<meta\s[^>]*>`)
	attr     = regexp.MustCompile(`(?s)([a-zA-Z:-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
)

// metaAttrs returns the attributes of a meta tag with lowercase names.
func metaAttrs(tag string) map[string]string {
	out := map[string]string{}
	for _, m := range attr.FindAllStringSubmatch(tag, -1) {
		out[strings.ToLower(m[1])] = html.UnescapeString(m[2] + m[3] + m[4])
	}
	return out
}

// publicBaseURL returns the scheme, host, and path of the Application URL.
// It returns an empty string for a host that the public cannot reach. The
// PocketBase default is on such a host.
func publicBaseURL(appURL string) string {
	u, err := url.Parse(appURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast()) {
		return ""
	}
	base := url.URL{Scheme: u.Scheme, Host: u.Host, Path: strings.TrimSuffix(u.Path, "/")}
	return base.String()
}

// pageMeta has the values of the page-meta block. html/template escapes
// each value for its HTML context.
type pageMeta struct {
	Title       string
	OGTitle     string
	Description string
	Type        string
	URL         string
	Image       string
	ImageWidth  int
	ImageHeight int
	ImageAlt    string
}

var metaTemplate = template.Must(template.New("meta").Parse(`<title>{{.Title}}</title>
    <meta name="description" content="{{.Description}}" />
    <meta property="og:type" content="{{.Type}}" />
    <meta property="og:title" content="{{.OGTitle}}" />
    <meta property="og:description" content="{{.Description}}" />
    {{- if .URL}}
    <meta property="og:url" content="{{.URL}}" />
    {{- end}}
    <meta property="og:image" content="{{.Image}}" />
    {{- if .ImageWidth}}
    <meta property="og:image:width" content="{{.ImageWidth}}" />
    <meta property="og:image:height" content="{{.ImageHeight}}" />
    {{- end}}
    <meta property="og:image:alt" content="{{.ImageAlt}}" />
    <meta name="twitter:card" content="summary_large_image" />
    <meta name="twitter:title" content="{{.OGTitle}}" />
    <meta name="twitter:description" content="{{.Description}}" />`))

// indexPage is index.html cut around the page-meta block. siteImage comes
// from the default tags of the block. siteName comes from og:site_name or
// from the default title.
type indexPage struct {
	whole, before, after []byte
	siteName             string
	siteImage            pageMeta
	// zone has the weeks of the week pages.
	zone *time.Location
	now  func() time.Time
}

// readIndex returns false if index.html has no page-meta block with a title
// and an og:image. The placeholder page has no block.
func readIndex(files fs.FS) (indexPage, bool) {
	data, err := fs.ReadFile(files, "index.html")
	if err != nil {
		return indexPage{}, false
	}
	start := bytes.Index(data, []byte(metaStart))
	end := bytes.Index(data, []byte(metaEnd))
	if start < 0 || end < start {
		return indexPage{}, false
	}
	block := string(data[start:end])
	title := titleTag.FindStringSubmatch(block)
	if title == nil {
		return indexPage{}, false
	}
	p := indexPage{whole: data, before: data[:start], after: data[end+len(metaEnd):], siteName: html.UnescapeString(strings.TrimSpace(title[1]))}
	for _, tag := range metaTag.FindAllString(string(data[:end]), -1) {
		if a := metaAttrs(tag); a["property"] == "og:site_name" && strings.TrimSpace(a["content"]) != "" {
			p.siteName = strings.TrimSpace(a["content"])
		}
	}
	for _, tag := range metaTag.FindAllString(block, -1) {
		a := metaAttrs(tag)
		switch value := a["content"]; a["property"] {
		case "og:image":
			p.siteImage.Image = value
		case "og:image:width":
			p.siteImage.ImageWidth, _ = strconv.Atoi(value)
		case "og:image:height":
			p.siteImage.ImageHeight, _ = strconv.Atoi(value)
		case "og:image:alt":
			p.siteImage.ImageAlt = value
		}
	}
	return p, p.siteImage.Image != ""
}

func (p indexPage) render(meta pageMeta) ([]byte, error) {
	var b bytes.Buffer
	b.Write(p.before)
	if err := metaTemplate.Execute(&b, meta); err != nil {
		return nil, err
	}
	b.Write(p.after)
	return b.Bytes(), nil
}

// pageTitle has the format of usePageTitle in the UI.
func (p indexPage) pageTitle(title string) string {
	return title + " · " + p.siteName
}

// errNoRecord tells that the path names a restaurant, a recipe, a pantry
// item, a review, a user, a list, a tag, or a week that does not exist.
var errNoRecord = errors.New("no record")

// metaFor returns false for a path that is not a restaurant, recipe, pantry
// item, review, reviewer, list, tag, or week page. appURL is the Application
// URL of the PocketBase settings.
// The literal path parts ignore case like Vue Router.
func (p indexPage) metaFor(app core.App, appURL, path string) (pageMeta, bool, error) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) == 2 && strings.EqualFold(parts[0], "reviewers") && recordID.MatchString(parts[1]) {
		user, err := app.FindRecordById("users", parts[1])
		if err != nil {
			return pageMeta{}, true, notFound(err)
		}
		// A user without approval has no public profile. See decision 00003.
		if !user.GetBool("approved") {
			return pageMeta{}, true, errNoRecord
		}
		return p.reviewerMeta(user, publicBaseURL(appURL)), true, nil
	}
	if len(parts) == 2 && strings.EqualFold(parts[0], "lists") && recordID.MatchString(parts[1]) {
		list, err := app.FindRecordById("lists", parts[1])
		if err != nil {
			return pageMeta{}, true, notFound(err)
		}
		creator, err := app.FindRecordById("users", list.GetString("creator"))
		if err != nil {
			return pageMeta{}, true, notFound(err)
		}
		if !creator.GetBool("approved") {
			return pageMeta{}, true, errNoRecord
		}
		counts, err := countsOf(app, list.Id, "[[list]]", func(k subjects.Tables) string { return k.ListItems })
		if err != nil {
			return pageMeta{}, true, err
		}
		return p.listMeta(list, creator, counts, publicBaseURL(appURL)), true, nil
	}
	if len(parts) == 2 && strings.EqualFold(parts[0], "weeks") {
		return p.weekPage(app, appURL, parts[1])
	}
	if len(parts) == 2 && strings.EqualFold(parts[0], "tags") && parts[1] != "" {
		return p.tagPage(app, appURL, norm.NFC.String(parts[1]))
	}
	if len(parts) < 2 || !recordID.MatchString(parts[1]) {
		return pageMeta{}, false, nil
	}
	i := slices.IndexFunc(reviewedKinds, func(k reviewedKind) bool { return strings.EqualFold(parts[0], k.Path) })
	isReview := len(parts) == 4 && strings.EqualFold(parts[2], "reviews") && recordID.MatchString(parts[3])
	if i < 0 || (len(parts) != 2 && !isReview) {
		return pageMeta{}, false, nil
	}
	kind := reviewedKinds[i]
	baseURL := publicBaseURL(appURL)
	parent, err := app.FindRecordById(kind.Table, parts[1])
	if err != nil {
		return pageMeta{}, true, notFound(err)
	}
	if !isReview {
		return p.parentMeta(kind, parent, baseURL), true, nil
	}
	review, err := app.FindRecordById(kind.Reviews, parts[3])
	if err != nil {
		return pageMeta{}, true, notFound(err)
	}
	if review.GetString(kind.Parent) != parent.Id {
		return pageMeta{}, true, errNoRecord
	}
	photo := &core.Record{}
	err = app.RecordQuery(kind.Photos).
		AndWhere(dbx.HashExp{"review": review.Id}).
		OrderBy("created ASC", "id ASC").
		Limit(1).
		One(photo)
	if errors.Is(err, sql.ErrNoRows) {
		photo = nil
	} else if err != nil {
		return pageMeta{}, true, err
	}
	return p.reviewMeta(kind, parent, review, photo, baseURL), true, nil
}

// reviewedKind has the parts of the pages of one kind of reviewed thing.
type reviewedKind struct {
	subjects.Tables
	photoFilePath func(parentID, reviewID, photoID string, variant reviewsapi.PhotoVariant) string
	noun          string
	// entry is the word of one review on a week page. See decision 00013.
	entry string
	// reviews and aReview start the default descriptions of the pages.
	reviews, aReview string
}

var (
	restaurantKind = reviewedKind{
		Tables: subjects.Restaurants, photoFilePath: publicapi.PhotoFilePath,
		noun: "restaurant", entry: "visit", reviews: "Reviews", aReview: "A review",
	}
	recipeKind = reviewedKind{
		Tables: subjects.Recipes, photoFilePath: publicapi.RecipePhotoFilePath,
		noun: "recipe", entry: "cook", reviews: "Reviews", aReview: "A review",
	}
	pantryItemKind = reviewedKind{
		Tables: subjects.PantryItems, photoFilePath: publicapi.PantryItemPhotoFilePath,
		noun: "pantry item", entry: "tasting", reviews: "Tastings", aReview: "A tasting",
	}
	reviewedKinds = []reviewedKind{restaurantKind, recipeKind, pantryItemKind}
)

func (k reviewedKind) path(parent *core.Record) string {
	return "/" + k.Path + "/" + parent.Id
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errNoRecord
	}
	return err
}

// pageURL is empty without a baseURL.
func pageURL(baseURL, path string) string {
	if baseURL == "" {
		return ""
	}
	return baseURL + path
}

func (p indexPage) parentMeta(kind reviewedKind, parent *core.Record, baseURL string) pageMeta {
	name := parent.GetString("name")
	description := shorten(parent.GetString("description"))
	if description == "" {
		description = kind.reviews + " of " + name + " on " + p.siteName + "."
	}
	meta := p.siteImage
	meta.Title = p.pageTitle(name)
	meta.OGTitle = name
	meta.Description = description
	meta.Type = "website"
	meta.URL = pageURL(baseURL, kind.path(parent))
	return meta
}

// itemCounts has the number of items of each kind in the order of
// reviewedKinds.
type itemCounts []int

// countsOf counts the rows of the table of each kind where the column is id.
func countsOf(app core.App, id, column string, table func(subjects.Tables) string) (itemCounts, error) {
	parts := make([]string, len(reviewedKinds))
	for i, k := range reviewedKinds {
		parts[i] = "(SELECT COUNT(*) FROM {{" + table(k.Tables) + "}} WHERE " + column + " = {:id}) AS [[n" + strconv.Itoa(i) + "]]"
	}
	row := dbx.NullStringMap{}
	if err := app.DB().NewQuery("SELECT " + strings.Join(parts, ", ")).Bind(dbx.Params{"id": id}).One(row); err != nil {
		return nil, err
	}
	counts := make(itemCounts, len(reviewedKinds))
	for i := range counts {
		counts[i], _ = strconv.Atoi(row["n"+strconv.Itoa(i)].String)
	}
	return counts, nil
}

// items names the counts that are not 0. A list without items has 0
// restaurants.
func (c itemCounts) items() string {
	var names []string
	for i, n := range c {
		if n > 0 {
			names = append(names, countOf(n, reviewedKinds[i].noun))
		}
	}
	if len(names) == 0 {
		return countOf(0, reviewedKinds[0].noun)
	}
	return joinAnd(names)
}

func joinAnd(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

func (c itemCounts) empty() bool {
	return !slices.ContainsFunc(c, func(n int) bool { return n > 0 })
}

func countOf(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// listMeta has only the public fields of the creator.
func (p indexPage) listMeta(list, creator *core.Record, counts itemCounts, baseURL string) pageMeta {
	title := list.GetString("name") + " by " + creator.GetString("name")
	meta := p.siteImage
	meta.Title = p.pageTitle(title)
	meta.OGTitle = title
	meta.Description = "A list of " + counts.items() + " by " + creator.GetString("name") + " on " + p.siteName + "."
	meta.Type = "website"
	meta.URL = pageURL(baseURL, "/lists/"+list.Id)
	return meta
}

// tagPage has no record for a tag without a restaurant, a recipe, or a
// pantry item. It changes free text to a tag name. The Public API does the
// same. See decision 00011.
func (p indexPage) tagPage(app core.App, appURL, name string) (pageMeta, bool, error) {
	tag, err := app.FindFirstRecordByData(subjects.TagCollection, "name", tagname.Of(name))
	if err != nil {
		return pageMeta{}, true, notFound(err)
	}
	counts, err := countsOf(app, tag.Id, "[[tag]]", func(k subjects.Tables) string { return k.Tags })
	if err != nil {
		return pageMeta{}, true, err
	}
	if counts.empty() {
		return pageMeta{}, true, errNoRecord
	}
	return p.tagMeta(tag.GetString("name"), counts, publicBaseURL(appURL)), true, nil
}

func (p indexPage) tagMeta(name string, counts itemCounts, baseURL string) pageMeta {
	title := "Tag " + name
	meta := p.siteImage
	meta.Title = p.pageTitle(title)
	meta.OGTitle = title
	meta.Description = counts.items() + " with the tag " + name + " on " + p.siteName + "."
	meta.Type = "website"
	meta.URL = pageURL(baseURL, "/tags/"+url.PathEscape(name))
	return meta
}

// reviewerMeta has only the public fields of the user.
func (p indexPage) reviewerMeta(user *core.Record, baseURL string) pageMeta {
	name := user.GetString("name")
	meta := p.siteImage
	meta.Title = p.pageTitle(name)
	meta.OGTitle = name
	meta.Description = "Reviews by " + name + " on " + p.siteName + "."
	meta.Type = "profile"
	meta.URL = pageURL(baseURL, "/reviewers/"+user.Id)
	return meta
}

// reviewMeta uses the site image without a baseURL. og:image needs an
// absolute URL.
func (p indexPage) reviewMeta(kind reviewedKind, parent, review, photo *core.Record, baseURL string) pageMeta {
	title := review.GetString("title")
	description := shorten(review.GetString("feedback"))
	if description == "" {
		description = kind.aReview + " of " + parent.GetString("name") + " on " + p.siteName + "."
	}
	meta := p.siteImage
	if photo != nil && baseURL != "" {
		meta.Image = baseURL + kind.photoFilePath(parent.Id, review.Id, photo.Id, reviewsapi.Full)
		meta.ImageWidth = photo.GetInt("width")
		meta.ImageHeight = photo.GetInt("height")
		meta.ImageAlt = photo.GetString("alt")
		if meta.ImageAlt == "" {
			meta.ImageAlt = "A photo of the review " + title
		}
	}
	meta.Title = p.pageTitle(title)
	meta.OGTitle = title + " · " + parent.GetString("name")
	meta.Description = description
	meta.Type = "article"
	meta.URL = pageURL(baseURL, kind.path(parent)+"/reviews/"+review.Id)
	return meta
}

// shorten puts all whitespace on one line. It cuts a long text at a word
// boundary and adds an ellipsis.
func shorten(text string) string {
	runes := []rune(strings.Join(strings.Fields(text), " "))
	if len(runes) <= maxDescriptionRunes {
		return string(runes)
	}
	cut := string(runes[:maxDescriptionRunes])
	if runes[maxDescriptionRunes] != ' ' {
		if i := strings.LastIndexByte(cut, ' '); i > 0 {
			cut = cut[:i]
		}
	}
	return strings.TrimRight(cut, ".,;:!? ") + "…"
}
