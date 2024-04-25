# Parse Html Head Info
This is a simple widget that just gets the basic information inside the head tag of the target website

## Usage
### Start using it
Download and install it:

```bash
go get github.com/gotoobe/parse-html-head-info
```

Import it in your code:

```bash
import ParseSite "github.com/gotoobe/parse-html-head-info"
```

## Example
Here's a simple example as follows:

```go
package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ParseSite "github.com/gotoobe/parse-html-head-info"
	"log"
	"net/http"
)

func main() {
	server := gin.Default()

	server.Use(cors.Default())

	server.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hi~")
	})
	siteUrl := "https://www.example.com/"
	parseSite := ParseSite.ParseInfoConfig{
		URL:     siteUrl,
		Timeout: 5000,
		//ProxyAddress: "http://127.0.0.1:7890",
	}
	server.GET("/site-info", func(c *gin.Context) {
		siteInfo, err := parseSite.GetSiteHeadInfo()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusRequestTimeout,
				"message": "timeout",
				"data":    nil,
			})
			log.Printf("Error: %s", err)
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusOK,
				"message": "ok",
				"data": gin.H{
					"title":           siteInfo.Title,
					"description":     siteInfo.Description,
					"keywords":        siteInfo.Keywords,
					"iconUrl":         siteInfo.IconUrl,
					"requestHtmlCost": siteInfo.RequestSiteCost,
				}})
		}
	})
	server.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	log.Fatalln(server.Run(":8788"))
}
```
