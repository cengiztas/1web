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
	// attr := map[string]string{"link": "href='/css/styles.min.css'"} // , "rel"="stylesheet', type='text/css'}

	// fmt.Printf("New Node: %q\n", newNode.DataAtom)

	// switch node.Type {
	// case html.ElementNode:
	// 	fmt.Printf("Element Node: %q\n", node.DataAtom)
	// case html.CommentNode:
	// 	fmt.Printf("Comment Node: %q\n", node.DataAtom)
	// case html.DoctypeNode:
	// 	fmt.Printf("Doctype Node: %q\n", node.DataAtom)
	// case html.TextNode:
	// 	fmt.Printf("Text Node: %q - %q\n", node.DataAtom, node.Data)
	// case html.ErrorNode:
	// 	fmt.Printf("Error Node: %q\n", node.DataAtom)
	// case html.DocumentNode:
	// 	fmt.Printf("Document Node: %q\n", node.DataAtom)
	// default:
	// 	fmt.Printf("Unknown Node: %q\n", node.Data)

	// }

	if node.Type != html.ElementNode {
		// It's not an HTML tag.
		return []*html.Node{}
	} else if !isWhitelistedNode(node) {
		// It's not a whitelisted tag. Delete it.
		matches = append(matches, node)

	} else if node.Type == html.CommentNode {
		// this does not work
		matches = append(matches, node)
		// this works
		node.Parent.RemoveChild(node)

	} else {
		// Its a whitelisted tag. Dig deeper.
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			matches = append(matches, WebOneMatcher{}.MatchAll(c)...)
		}

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
	fmt.Println("QUERY	: <", query, ">")

	u, err := url.Parse(query)
	if err != nil {
		panic(err)
	}

	s := u.Scheme

	if s == "" {
		query = "http://" + query
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
	res, err := http.Get(query)
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
