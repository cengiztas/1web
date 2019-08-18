package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var whitelistedTags = map[atom.Atom]struct{}{
	atom.Html:       struct{}{},
	atom.Head:       struct{}{},
	atom.Title:      struct{}{},
	atom.Body:       struct{}{},
	atom.H1:         struct{}{},
	atom.H2:         struct{}{},
	atom.H3:         struct{}{},
	atom.H4:         struct{}{},
	atom.H5:         struct{}{},
	atom.H6:         struct{}{},
	atom.P:          struct{}{},
	atom.Br:         struct{}{},
	atom.Hr:         struct{}{},
	atom.A:          struct{}{},
	atom.Nav:        struct{}{},
	atom.Meta:       struct{}{},
	atom.Div:        struct{}{},
	atom.Span:       struct{}{},
	atom.Header:     struct{}{},
	atom.Footer:     struct{}{},
	atom.Section:    struct{}{},
	atom.Article:    struct{}{},
	atom.Summary:    struct{}{},
	atom.Table:      struct{}{},
	atom.Caption:    struct{}{},
	atom.Th:         struct{}{},
	atom.Tr:         struct{}{},
	atom.Td:         struct{}{},
	atom.Thead:      struct{}{},
	atom.Tbody:      struct{}{},
	atom.Tfoot:      struct{}{},
	atom.Col:        struct{}{},
	atom.Colgroup:   struct{}{},
	atom.Ul:         struct{}{},
	atom.Ol:         struct{}{},
	atom.Li:         struct{}{},
	atom.Dl:         struct{}{},
	atom.Dt:         struct{}{},
	atom.Dd:         struct{}{},
	atom.B:          struct{}{},
	atom.Blockquote: struct{}{},
	atom.Code:       struct{}{},
	atom.Center:     struct{}{},
	atom.Pre:        struct{}{},
	atom.I:          struct{}{},
	atom.Q:          struct{}{},
	atom.S:          struct{}{},
	atom.Strong:     struct{}{},
	atom.U:          struct{}{},
	atom.Main:       struct{}{},
	atom.Link:       struct{}{},
}

var whitelistedAttrs = map[string]struct{}{
	"href":    struct{}{},
	"colspan": struct{}{},
}

var whitelistedEmptyTags = map[atom.Atom]struct{}{
	atom.Td: struct{}{},
}

var u *url.URL

func main() {
	http.HandleFunc("/search", purify)
	http.Handle("/", http.FileServer(http.Dir(".")))

	fmt.Println("listening on localhost:8888 ..")
	http.ListenAndServe(":8888", nil)
}

type WebOneMatcher struct{}

func (WebOneMatcher) Match(node *html.Node) bool {
	return false
}

func (WebOneMatcher) MatchAll(node *html.Node) []*html.Node {
	var matches []*html.Node

	if !isWhitelistedNode(node) {
		// It's not a whitelisted tag. Delete it.
		matches = append(matches, node)
	}

	switch node.Type {
	case html.ElementNode:
		if isEmptyNode(node) && !isWhitelistedEmptyNode(node) {
			matches = append(matches, node)
		}

		// It is a whitelisted tag. Dig deeper.
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			matches = append(matches, WebOneMatcher{}.MatchAll(c)...)
		}

		// TODO: It should be faster to delete all attributes at once if array does not contain whitelisted attribute.
		for i := 0; i < len(node.Attr); i++ {
			if !isWhitelistedAttr(node.Attr[i].Key) {
				last := len(node.Attr) - 1
				node.Attr[i] = node.Attr[last] // overwrite the target with the last attribute
				node.Attr = node.Attr[:last]   // then slice off the last attribute
				i--
			}
		}

	case html.TextNode:
		node.Data = strings.TrimSpace(node.Data)
		if node.Data == "" {
			// matches = append(matches, node)
		} else {

			return []*html.Node{}
		}

	case html.CommentNode:
		fmt.Printf("Comment node found: %q\n", node.Data)
		matches = append(matches, node)
	case html.DoctypeNode:
		fmt.Printf("Doctype node found: %q\n", node.Data)
	case html.ErrorNode:
		fmt.Printf("Error node found: %q\n", node.Data)
	case html.DocumentNode:
		fmt.Printf("Document node found: %q\n", node.Data)
	default:
		fmt.Printf("Unknown node found: %q\n", node.Data)
		return []*html.Node{}

	}

	return matches
}

func (WebOneMatcher) Filter(nodes []*html.Node) []*html.Node {
	return nil
}

func isWhitelistedNode(node *html.Node) bool {
	_, whitelisted := whitelistedTags[node.DataAtom]
	return whitelisted
}

func isWhitelistedEmptyNode(node *html.Node) bool {
	_, whitelisted := whitelistedEmptyTags[node.DataAtom]
	return whitelisted
}

func isEmptyNode(node *html.Node) bool {
	return node.FirstChild == nil
}

func isWhitelistedAttr(attr string) bool {
	_, whitelisted := whitelistedAttrs[attr]
	return whitelisted
}

func updateAHref(sel *goquery.Selection) *goquery.Selection {
	sel.Each(func(i int, item *goquery.Selection) {

		// If relative path, join url with host name.
		ahref, _ := item.Attr("href")

		href, err := url.Parse(ahref)
		if err != nil {
			panic(err)
		}

		base, err := url.Parse(u.String())
		if err != nil {
			panic(err)
		}

		item.SetAttr("href", ("?query=" + base.ResolveReference(href).String()))
	})

	return sel

}

func purify(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")

	var err error

	u, err = url.Parse(query)
	if err != nil {
		panic(err)
	}

	s := u.Scheme

	if s == "" {
		u.Scheme = "http"
	}

	// resp, err := client.Get(query)

	client := &http.Client{}
	client.Timeout = time.Second * 15

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_1) AppleWebKit/602.2.14 (KHTML, like Gecko) Version/10.0.1 Safari/602.2.14")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	fmt.Printf("Status %q\n", resp.Status)
	if err != nil {
		fmt.Println("StatusCode		:", resp.StatusCode)
		fmt.Println("Redirect URL	:", resp.Header.Get("Location"))
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprint(resp.Status)))
		return
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	// doc.Find("*").Each(func(i int, node *goquery.Selection) {
	// 	// fmt.Printf("node: %q\n", goquery.NodeName(node))
	// 	if _, whitelisted := whitelistedTags[atom.Lookup([]byte(goquery.NodeName(node)))]; !whitelisted {
	// 		node.Remove()
	// 		// fmt.Printf("Removing node: %q\n", node.Text())
	// 	} else {
	// 	}

	// })

	// purify
	doc.FindMatcher(WebOneMatcher{}).Remove()

	selection := doc.Find("a")
	updateAHref(selection)

	doc.Find("head").AppendHtml("<link href='/css/styles.min.css' rel='stylesheet' type='text/css'/>")
	doc.Find("head").AppendHtml("<meta name='viewport' content='&#39;width=device-width, initial-scale=1.0&#39;' initial-scale='1.0'/>")

	htmlStr, err := doc.Html()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(htmlStr))
}
