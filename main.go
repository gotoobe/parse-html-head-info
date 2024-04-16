package main

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

func main() {
	server := gin.Default()

	server.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hi~")
	})
	server.GET("/site-info", getSiteInfo)
	server.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	log.Fatalln(server.Run(":8788"))

}

func getSiteInfo(c *gin.Context) {
	var siteUrl string
	uri := c.DefaultQuery("url", "http://example.com")
	parsedUrl, err := url.Parse(uri)
	if err != nil {
		fmt.Println(err)
	} else if parsedUrl.IsAbs() {
		if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
			siteUrl = parsedUrl.String()
			siteInfo, err := requestHtml(siteUrl)
			iconPath, _ := parsedUrl.Parse(siteInfo.Data.IconUrl)

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    http.StatusBadRequest,
					"message": siteInfo.Message,
					"data":    nil,
				})
			} else {
				c.JSON(http.StatusOK,
					gin.H{
						"code":    http.StatusOK,
						"message": siteInfo.Message,
						"data": gin.H{
							"title":           siteInfo.Data.Title,
							"description":     siteInfo.Data.Description,
							"keywords":        siteInfo.Data.Keywords,
							"iconUrl":         iconPath.String(),
							"host":            parsedUrl.Host,
							"requestHtmlCost": siteInfo.RequestSiteCost,
						},
					})
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "查询网址需完整。形式需如 http(s)://example.com/**",
				"data":    nil,
			})
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "网址协议错误。形式需如 http(s)://example.com/**",
			"data":    nil,
		})
	}
}

func requestHtml(siteUrl string) (Response, error) {
	request, err := http.NewRequest("GET", siteUrl, nil)
	if err != nil {
		log.Println(err)
		return Response{
			Code:    http.StatusInternalServerError,
			Message: "服务器内部异常",
		}, err
	}

	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en-CN;q=0.8,en;q=0.7,fr-FR;q=0.6,fr;q=0.5")
	request.Header.Set("Accept-Encoding", "gzip, deflate, br")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	request.Header.Set("Connection", "keep-alive")

	//proxy, _ := url.Parse("http://127.0.0.1:7890")
	//tr := &http.Transport{
	//	Proxy:           http.ProxyURL(proxy),
	//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//}

	client := &http.Client{
		//Transport: tr,
		Timeout: time.Second * 5, //超时时间
	}

	requestHtmlStart := time.Now()
	resp, err := client.Do(request)

	if err != nil {
		log.Println("无法获取网页内容：", err)
		return Response{
			Code:    http.StatusInternalServerError,
			Message: "无法获取网页内容",
		}, err
	}
	defer resp.Body.Close()

	requestHtmlCost := time.Since(requestHtmlStart)

	// 处理压缩响应体
	var bodyReader io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		bodyReader, err = gzip.NewReader(resp.Body)
	case "deflate":
		bodyReader = flate.NewReader(resp.Body)
	case "br":
		bodyReader = brotli.NewReader(resp.Body)
	default:
		bodyReader = resp.Body
	}

	doc, err := html.Parse(bodyReader)
	if err != nil {
		log.Println("无法解析 HTML 内容：", err)
		return Response{
			Code:    http.StatusInternalServerError,
			Message: "无法解析 HTML 内容",
		}, err
	}

	head := findHeadElement(doc)
	var siteInfo SiteInfo
	if head != nil {
		siteInfo = processHeadElement(head)
	}

	return Response{
		Code:            http.StatusOK,
		Message:         "ok",
		Data:            siteInfo,
		RequestSiteCost: requestHtmlCost.String(),
	}, nil
}

func findHeadElement(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "head" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := findHeadElement(c)
		if result != nil {
			return result
		}
	}
	return nil
}

func processHeadElement(headNode *html.Node) SiteInfo {
	title, linkTags, metaTags := extractHeadTags(headNode)
	var iconUrl string
	var description string

	// TODO: more info, e.g. opengraph info, twitter info
	for _, meta := range metaTags {
		if meta["name"] == "description" {
			description = meta["content"]
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
	}
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
