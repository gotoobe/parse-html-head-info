package main

type SiteInfo struct {
	Title, Description, Keywords, IconUrl, Host string
}

type Response struct {
	Code            int
	Message         string
	Data            SiteInfo
	RequestSiteCost string
}
