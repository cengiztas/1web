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
}

var whitelistedAttrs = map[string]struct{}{
	"href":    struct{}{},
	"colspan": struct{}{},
}

var whitelistedEmptyTags = map[atom.Atom]struct{}{
	atom.Td:   struct{}{},
	atom.Head: struct{}{},
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

	// File Server
	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "/public")
	fs := http.FileServer(http.Dir(filesDir))
	http.Handle("/public/", http.StripPrefix("/public", fs))

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
	log.Printf("node	: %q\n", node.Data)

	// if !isWhitelistedNode(node) && strings.TrimSpace(node.Data) != "" {
	if !isWhitelistedNode(node) && node.Type != html.TextNode {
		// It's neither a whitelisted nor an empty text node. Delete it.
		log.Printf("deleting empty or unallowed node: %q\n", node.Data)
		matches = append(matches, node)
	} else if node.Type == html.TextNode && strings.TrimSpace(node.Data) == "" {
		log.Printf("deleting empty node: %q\n", node.Data)
		matches = append(matches, node)

	} else if node.FirstChild == nil && !isWhitelistedEmptyNode(node) && node.Type != html.TextNode {
		// It's an empty node. Delete it.
		log.Printf("deleting empty node: %q\n", node.Data)
		matches = append(matches, node)

	} else {
		childrenCount := 0
		matchedChildrenCount := 0

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			childrenCount++

			if c.Type == html.TextNode && strings.TrimSpace(c.Data) != "" {
				continue
			}

			log.Printf("next node	: %q\n", c.Data)

			childMatches := WebOneMatcher{}.MatchAll(c)

			// If the child will be deleted, increment matchedChildrenCount.
			if contains(childMatches, c) {
				matchedChildrenCount++
			}

			matches = append(matches, childMatches...)
			log.Printf("back from rec: %q\n", c.Data)
		}

		// A node can be deleted, if all its children will be deleted and it's
		// not a text node.
		if childrenCount == matchedChildrenCount && !isWhitelistedEmptyNode(node) {
			log.Println("--------> COUNT EQUAL")
			matches = append(matches, node)
		}

		for i := 0; i < len(node.Attr); i++ {
			if !isWhitelistedAttr(node.Attr[i].Key) {
				last := len(node.Attr) - 1
				node.Attr[i] = node.Attr[last] // overwrite the target with the last attribute
				node.Attr = node.Attr[:last]   // then slice off the last attribute
				i--
			}
		}
	}



	/*
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			// log.Printf("digging deeper: %q\n", c.Data)
			matches = append(matches, WebOneMatcher{}.MatchAll(c)...)
		}
			if !isWhitelistedNode(node) && node.Type != html.TextNode {
				// log.Printf("removing not whitelisted node %q:\n", node.Data)
				// It's not a whitelisted tag. Delete it.
				matches = append(matches, node)
			}

			// It is a whitelisted tag. Dig deeper.
			for c := node.FirstChild; c != nil; c = c.NextSibling {
				// log.Printf("digging deeper: %q\n", c.Data)
				matches = append(matches, WebOneMatcher{}.MatchAll(c)...)
			}

			if node.Type == html.TextNode && strings.TrimSpace(node.Data) == "" {
				log.Printf("TEXTNODE is empty. Adding to match list.\n")
				// matches = append(matches, node)
				// TODO: clean up from bottom up
			}

			if node.FirstChild == nil && !isWhitelistedEmptyNode(node) && node.Type == html.ElementNode {
				var reverseRemove func(n *html.Node)

				reverseRemove = func(n *html.Node) {
					p := n.Parent

					if p != nil && n.NextSibling == nil {
						p.RemoveChild(n)
						log.Printf("empty node %q deleted\n", n.DataAtom)
						reverseRemove(p)
					}
				}

				reverseRemove(node)

			}

			for i := 0; i < len(node.Attr); i++ {
				if !isWhitelistedAttr(node.Attr[i].Key) {
					last := len(node.Attr) - 1
					node.Attr[i] = node.Attr[last] // overwrite the target with the last attribute
					node.Attr = node.Attr[:last]   // then slice off the last attribute
					i--
				}
			}
	*/

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

	// start timer
	start := time.Now()

	// remove all unwanted stuff
	doc.FindMatcher(WebOneMatcher{}).Remove()

	// modify all anchor tags
	selection := doc.Find("a")
	updateAHref(selection)

	// extend head with url to css and set the viewport
	doc.Find("head").AppendHtml("<link href='/css/styles.min.css' rel='stylesheet' type='text/css'/>")
	doc.Find("head").AppendHtml("<meta name='viewport' content='&#39;width=device-width, initial-scale=1.0&#39;' initial-scale='1.0'/>")

	htmlStr, err := doc.Html()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	// stop timer
	elapsed := time.Since(start)
	log.Printf("duration: %s\n", elapsed)

	w.Write([]byte(htmlStr))
}
