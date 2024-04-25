package parsehtmlheadinfo

import (
	"fmt"
	"testing"
)

func TestParseInfo_GetSiteHeadInfo(t *testing.T) {
	siteUrl := "https://www.youtube.com"
	parseSite := ParseInfoConfig{
		URL: siteUrl,
	}
	siteInfo, err := parseSite.GetSiteHeadInfo()
	if err != nil {
		t.Fatalf("failed to get header information of %s, err: %s", siteUrl, err)
	}
	fmt.Printf("Get website (%s) information successfully: %s\n", siteUrl, siteInfo)
}
