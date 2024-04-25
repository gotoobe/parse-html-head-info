package parsehtmlheadinfo

import (
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"github.com/andybalholm/brotli"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ParseInfoConfig contains options for getting information within the head tag of the target site
type ParseInfoConfig struct {
	// website url
	URL string
	// timeout in ms
	Timeout      time.Duration
	ProxyAddress string
	// same as *tls.Config
	TLSClientConfig *tls.Config
	OnlyBasicInfo   bool
}

type SiteInfo struct {
	Title, Description, Keywords, IconUrl, RequestSiteCost string
}

// GetSiteHeadInfo Get information about the head of the website
func (info *ParseInfoConfig) GetSiteHeadInfo() (SiteInfo, error) {
	respStatusCode, doc, requestHtmlCost, reqErr := info.requestHtml()
	if reqErr != nil {
		return SiteInfo{}, reqErr
	}

	if respStatusCode != http.StatusOK {
		cErr := fmt.Errorf("website request error with %d response code", respStatusCode)
		return SiteInfo{}, cErr
	}

	head := getHtmlHead(doc)
	returnInfo := processHeadTags(head)

	return SiteInfo{
		Title:           returnInfo.Title,
		Description:     returnInfo.Description,
		Keywords:        returnInfo.Keywords,
		IconUrl:         returnInfo.IconUrl,
		RequestSiteCost: requestHtmlCost.String(),
	}, nil
}

func (info *ParseInfoConfig) requestHtml() (int, *html.Node, time.Duration, error) {
	request, err := http.NewRequest("GET", info.URL, nil)
	if err != nil {
		return 0, nil, 0, err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-CN;q=0.8,en;q=0.7,fr-FR;q=0.6,fr;q=0.5")
	request.Header.Set("Accept-Encoding", "gzip, deflate, br")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	request.Header.Set("Connection", "keep-alive")

	var proxy *url.URL
	var transport *http.Transport
	if info.ProxyAddress != "" {
		proxy, err = url.Parse(info.ProxyAddress)
		if err != nil {
			return 0, nil, 0, err
		}
		transport = &http.Transport{
			Proxy:           http.ProxyURL(proxy),
			TLSClientConfig: info.TLSClientConfig,
		}
	}

	var client *http.Client
	var timeout time.Duration
	if info.Timeout != 0 {
		timeout = time.Millisecond * info.Timeout
	} else {
		timeout = time.Millisecond * 5000
	}
	if transport != nil {
		client = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	} else {
		client = &http.Client{
			Timeout: timeout,
		}
	}

	requestHtmlStart := time.Now()
	resp, clientErr := client.Do(request)
	if clientErr != nil {
		cErr := fmt.Errorf("website request error: %s", clientErr)
		return 0, nil, 0, cErr
	}
	defer resp.Body.Close()
	requestHtmlCost := time.Since(requestHtmlStart)

	// handling compressed response bodies
	var bodyReader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":

		bodyReader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return 0, nil, 0, err
		}
	case "deflate":
		bodyReader = flate.NewReader(resp.Body)
	case "br":
		bodyReader = brotli.NewReader(resp.Body)
	default:
		bodyReader = resp.Body
	}

	doc, parseErr := html.Parse(bodyReader)
	if parseErr != nil {
		return 0, nil, 0, parseErr
	}

	return resp.StatusCode, doc, requestHtmlCost, nil
}

func getHtmlHead(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "head" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := getHtmlHead(c)
		if result != nil {
			return result
		}
	}
	return nil
}

func extractHeadTags(headNode *html.Node) (Title string, LinkTags []map[string]string, MetaTags []map[string]string) {
	var title string
	var metaTags []map[string]string
	var linkTags []map[string]string

	for child := headNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "meta" {
			meta := make(map[string]string)
			for _, attr := range child.Attr {
				meta[attr.Key] = attr.Val
			}
			metaTags = append(metaTags, meta)
		}
		if child.Type == html.ElementNode && child.Data == "title" {
			title = child.FirstChild.Data
		}
		if child.Type == html.ElementNode && child.Data == "link" {
			link := make(map[string]string)
			for _, attr := range child.Attr {
				link[attr.Key] = attr.Val
			}
			linkTags = append(linkTags, link)
		}
	}
	return title, linkTags, metaTags
}

func processHeadTags(headNode *html.Node) SiteInfo {
	title, linkTags, metaTags := extractHeadTags(headNode)
	var iconUrl string
	var description string
	var keywords string

	// TODO: more info, e.g. keywords, opengraph info, twitter info
	for _, meta := range metaTags {
		if meta["name"] == "description" {
			description = meta["content"]
		}
		if meta["keywords"] == "keywords" {
			keywords = meta["content"]
		}
	}
	for _, linkTag := range linkTags {
		if linkTag["rel"] == "icon" {
			iconUrl = linkTag["href"]
		}
	}

	return SiteInfo{
		Title:       title,
		IconUrl:     iconUrl,
		Description: description,
		Keywords:    keywords,
	}
}
