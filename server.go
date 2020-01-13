package main

import (
	"fmt"
	"github.com/unrolled/render"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tdewolff/minify/v2"
	mhtml "github.com/tdewolff/minify/v2/html"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var whitelistedTags = map[atom.Atom]struct{}{
	atom.Html:  struct{}{},
	atom.Head:  struct{}{},
	atom.Title: struct{}{},
	atom.Body:  struct{}{},
	atom.H1:    struct{}{},
	atom.H2:    struct{}{},
	atom.H3:    struct{}{},
	atom.H4:    struct{}{},
	atom.H5:    struct{}{},
	atom.H6:    struct{}{},
	atom.P:     struct{}{},
	atom.Br:    struct{}{},
	atom.Hr:    struct{}{},
	atom.A:     struct{}{},
	atom.Nav:   struct{}{},
	//atom.Meta:       struct{}{},
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
	atom.Form:       struct{}{},
	atom.Input:      struct{}{},
	atom.Textarea:   struct{}{},
	atom.Label:      struct{}{},
}

var whitelistedAttrs = map[string]struct{}{
	"href":    struct{}{},
	"colspan": struct{}{},
	"id":      struct{}{},
}

var whitelistedEmptyTags = map[atom.Atom]struct{}{
	atom.Td:       struct{}{},
	atom.Textarea: struct{}{},
	atom.Input:    struct{}{},
	atom.Form:     struct{}{},
	//atom.Head: struct{}{},
}

var keepAllAttributes = map[atom.Atom]struct{}{
	atom.Meta:  struct{}{},
	atom.Form:  struct{}{},
	atom.Input: struct{}{},
}

var u *url.URL

var view *render.Render

func main() {
	view = render.New(render.Options{
		Layout:          "layout",
		Extensions:      []string{".tmpl", ".html"},
		IsDevelopment:   true,
		RequirePartials: true,
	})

	http.HandleFunc("/", landing)
	http.HandleFunc("/search", purify)
	// TODO: form handler
	http.HandleFunc("/forms", forms)

	// File Server
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "/public")
	fs := http.FileServer(http.Dir(filesDir))
	http.Handle("/public/", http.StripPrefix("/public", fs))

	// TODO: switch to level based logging e.g. with https://github.com/sirupsen/logrus
	f, err := os.OpenFile("1web.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	fmt.Println("listening on localhost:8888 ..")
	http.ListenAndServe("localhost:8888", nil)
}

/**
 * Route Handlers
 */
func landing(w http.ResponseWriter, r *http.Request) {
	view.HTML(w, http.StatusOK, "index", nil)
}

/*
   Forms handler for e.g. search engines
*/
func forms(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		log.Fatalf("failed parsing form: %v", err)
	}

	f := r.Form

	var q string

	oa := r.FormValue("origin_action")
	om := r.FormValue("origin_method")

	//TODO: Double-Check URL. If relative extend to absolute.
	url, err := url.ParseRequestURI(oa)
	if err == nil {
		fmt.Printf("url parse error: %s\n", err)
		fmt.Printf("new url : %s\n", u.ResolveReference(url))
	}

	if om == "GET" || om == "get" {

		//q = oa + url.QueryEscape("?")
		q = oa + "?"

		for k, v := range f {
			if k == "origin_action" || k == "origin_method" {
				continue
			} else {

				q += k + "=" + strings.Join(v, "+") + "&"
				log.Printf("key: %s\tval: %v\n", k, v)
			}
		}

	}

	log.Printf("q: %s\n", q)

	client := &http.Client{}
	client.Timeout = time.Second * 15

	req, err := http.NewRequest("GET", q, nil)
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

	// TODO: https://github.com/tdewolff/minify
	m := minify.New()
	m.AddFunc("text/html", mhtml.Minify)

	// start timer
	start := time.Now()

	// remove all unwanted stuff
	doc.FindMatcher(WebOneMatcher{}).Remove()

	// modify all anchor tags
	selection := doc.Find("a")
	// TODO: exlude mailto
	updateAHref(selection)

	// extend head with url to css and set the viewport
	doc.Find("head").AppendHtml("<link href='/public/main.css' rel='stylesheet' type='text/css'/>")
	doc.Find("head").AppendHtml("<meta name='viewport' content='&#39;width=device-width, initial-scale=1.0&#39;' initial-scale='1.0'/>")

	// wrap all nav tags with details and summary tag to collapse all list elements
	doc.Find("nav").WrapHtml("<details class='list-container'>").Parent().PrependHtml("<summary>Click to expand</summary>")

	doc.Find("form").Each(func(index int, frm *goquery.Selection) {
		action, _ := frm.Attr("action")
		method, _ := frm.Attr("method")
		frm.SetAttr("method", "post")
		frm.AppendHtml(fmt.Sprintf("<input type='hidden' name='origin_action' value='%s'>", action))
		frm.AppendHtml(fmt.Sprintf("<input type='hidden' name='origin_method' value='%s'>", method))
		frm.SetAttr("action", "/forms")

	})

	htmlStr, err := doc.Html()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	// stop timer
	elapsed := time.Since(start)
	log.Printf("purifiying finished after %s (excluding network communication)\n", elapsed)

	w.Write([]byte(htmlStr))
	// view.HTML(w, http.StatusOK, "index", nil)
}

type WebOneMatcher struct{}

func (WebOneMatcher) Match(node *html.Node) bool {
	return false
}

// Brilliant example why Go sux (sometimes).
func contains(nodes []*html.Node, node *html.Node) bool {
	for _, n := range nodes {
		if n == node {
			return true
		}
	}
	return false
}

func (WebOneMatcher) MatchAll(node *html.Node) []*html.Node {
	var matches []*html.Node
	// log.Printf("node	: %q\n", node.Data)

	// if !isWhitelistedNode(node) && strings.TrimSpace(node.Data) != "" {
	if !isWhitelistedNode(node) && node.Type != html.TextNode {
		// It's neither a whitelisted nor an empty text node. Delete it.
		// log.Printf("deleting empty or unallowed node: %q\n", node.Data)
		matches = append(matches, node)
	} else if node.Type == html.TextNode && strings.TrimSpace(node.Data) == "" {
		// log.Printf("deleting empty node: %q\n", node.Data)
		matches = append(matches, node)

	} else if node.FirstChild == nil && !isWhitelistedEmptyNode(node) && node.Type != html.TextNode {
		// It's an empty node. Delete it.
		// log.Printf("deleting empty node: %q\n", node.Data)
		matches = append(matches, node)

	} else {
		childrenCount := 0
		matchedChildrenCount := 0

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			childrenCount++

			if c.Type == html.TextNode && strings.TrimSpace(c.Data) != "" {
				continue
			}

			// log.Printf("next node	: %q\n", c.Data)

			childMatches := WebOneMatcher{}.MatchAll(c)

			// If the child will be deleted, increment matchedChildrenCount.
			if contains(childMatches, c) {
				matchedChildrenCount++
			}

			// s := goquery.Selection{Nodes: childMatches}

			// if s.Contains(c) {
			// 	matchedChildrenCount++
			// }

			matches = append(matches, childMatches...)
			// log.Printf("children matched: %q\n", s)

		}

		// A node can be deleted, if all its children will be deleted and it's
		// not a text node.
		if childrenCount == matchedChildrenCount && !isWhitelistedEmptyNode(node) {
			// log.Println("--------> COUNT EQUAL")
			matches = append(matches, node)
		}

		if !isWhitelistedAttryNode(node) {
			for i := 0; i < len(node.Attr); i++ {
				if !isWhitelistedAttr(node.Attr[i].Key) {
					last := len(node.Attr) - 1
					node.Attr[i] = node.Attr[last] // overwrite the target with the last attribute
					node.Attr = node.Attr[:last]   // then slice off the last attribute
					i--
				}
			}
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

func isWhitelistedEmptyNode(node *html.Node) bool {
	_, whitelisted := whitelistedEmptyTags[node.DataAtom]
	return whitelisted
}

func isWhitelistedAttr(attr string) bool {
	_, whitelisted := whitelistedAttrs[attr]
	return whitelisted
}

func isWhitelistedAttryNode(node *html.Node) bool {
	_, whitelisted := keepAllAttributes[node.DataAtom]
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

		item.SetAttr("href", ("/search?query=" + url.QueryEscape(base.ResolveReference(href).String())))
	})

	return sel

}

func purify(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")

	var err error

	//u, err = url.ParseRequestURI(query)
	u, err = url.Parse(query)
	if err != nil {
		panic(err)
	}

	s := u.Scheme

	if s == "" {
		u.Scheme = "http"
	}

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

	// TODO: https://github.com/tdewolff/minify
	m := minify.New()
	m.AddFunc("text/html", mhtml.Minify)

	// htmlS, err := doc.Html()
	// str := strings.TrimRight(htmlS, "\r\n")
	// fmt.Printf("%s", str)

	// start timer
	start := time.Now()

	// remove all unwanted stuff
	doc.FindMatcher(WebOneMatcher{}).Remove()

	// modify all anchor tags
	selection := doc.Find("a")
	// TODO: exlude mailto
	updateAHref(selection)

	// extend head with url to css and set the viewport
	doc.Find("head").AppendHtml("<link href='/public/main.css' rel='stylesheet' type='text/css'/>")
	doc.Find("head").AppendHtml("<meta name='viewport' content='&#39;width=device-width, initial-scale=1.0&#39;' initial-scale='1.0'/>")

	// wrap all nav tags with details and summary tag to collapse all list elements
	// "*:not(nav) > ul" css selector: all ul tags without a nav parent
	doc.Find("*:not(nav) > ul").WrapHtml("<details class='list-container'>").Parent().PrependHtml("<summary>Click to expand</summary>")

	doc.Find("form").Each(func(index int, frm *goquery.Selection) {
		action, _ := frm.Attr("action")
		method, _ := frm.Attr("method")

		frm.SetAttr("method", "post")
		frm.AppendHtml(fmt.Sprintf("<input type='hidden' name='origin_action' value='%s'>", action))
		frm.AppendHtml(fmt.Sprintf("<input type='hidden' name='origin_method' value='%s'>", method))
		frm.SetAttr("action", "/forms")

	})

	htmlStr, err := doc.Html()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	// stop timer
	elapsed := time.Since(start)
	log.Printf("purifiying finished after %s (excluding network communication)\n", elapsed)

	w.Write([]byte(htmlStr))
}
