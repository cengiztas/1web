package main

import (
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/html/atom"

	"github.com/PuerkitoBio/goquery"
	// "github.com/go-chi/chi"
	"golang.org/x/net/html"
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
	atom.Link:       struct{}{},
}

var whitelistedAttrs = map[string]struct{}{
	"href":    struct{}{},
	"colspan": struct{}{},
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
		if node.FirstChild == nil {
			// It is an empty node. Delete it.
			// fmt.Printf("empty node: %s\n", node.Data)
			matches = append(matches, node)
		}

		// If relative path, join url with host name.
		if node.DataAtom == atom.A {
			href, err := url.Parse(node.Attr[0].Val)
			if err != nil {
				panic(err)
			}

			base, err := url.Parse(u.String())
			if err != nil {
				panic(err)
			}

			node.Attr[0].Val = "?query=" + base.ResolveReference(href).String()
			// fmt.Println(node.Attr[0].Val)
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

		if node.DataAtom == atom.Head {
			appendLinkNode(node)
			appendMetaNode(node)
		}
		// fmt.Printf("Element Node: %q\n", node.Data)
	case html.TextNode:
		// fmt.Printf("Text Node: %q\n", node.Data)
		return []*html.Node{}
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

func isWhitelistedAttr(attr string) bool {
	_, whitelisted := whitelistedAttrs[attr]
	return whitelisted
}

func appendLinkNode(node *html.Node) {
	attrs := []html.Attribute{
		html.Attribute{
			Key: "href",
			Val: "/css/styles.min.css",
		},
		html.Attribute{
			Key: "rel",
			Val: "stylesheet",
		},
		html.Attribute{
			Key: "type",
			Val: "text/css",
		},
	}

	newNode := html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Link,
		Data:     "link",
		Attr:     attrs,
	}

	node.AppendChild(&newNode)

}

func appendMetaNode(node *html.Node) {
	attrs := []html.Attribute{
		html.Attribute{
			Key: "name",
			Val: "viewport",
		},
		html.Attribute{
			Key: "content",
			Val: "'width=device-width, initial-scale=1.0'",
		},
		html.Attribute{
			Key: "initial-scale",
			Val: "1.0",
		},
	}

	newNode := html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Meta,
		Data:     "meta",
		Attr:     attrs,
	}

	node.AppendChild(&newNode)

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

	// client := &http.Client{
	// 	CheckRedirect: func(req *http.Request, via []*http.Request) error {
	// 		return http.ErrUseLastResponse
	// 	},
	// }

	// resp, err := client.Get(query)

	// req, err := http.NewRequest("GET", query, nil)
	// req.Header.Add("If-None-Match", `W/"wyzzy"`)
	// resp, err := client.Do(req)

	// Request the HTML page.
	res, err := http.Get(u.String())
	//res, err := http.Get(query)
	if err != nil {
		fmt.Println("StatusCode		:", res.StatusCode)
		fmt.Println("Redirect URL	:", res.Header.Get("Location"))
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprint("Status code is", res.StatusCode, res.Status)))
		return
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	// purify
	doc.FindMatcher(WebOneMatcher{}).Remove()

	htmlStr, err := doc.Html()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte(htmlStr))
}
