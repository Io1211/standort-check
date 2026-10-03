package backend

import "testing"

func validCampaign() CampaignRequest {
	return CampaignRequest{Name: "Dresden Herbst", UTMSource: "google", UTMMedium: "cpc", UTMCampaign: "dresden-herbst-2026"}
}

func TestCampaignValidate(t *testing.T) {
	if err := validCampaign().normalize().validate(); err != nil {
		t.Fatalf("valid campaign rejected: %v", err.Fields)
	}

	cases := []struct {
		name   string
		modify func(*CampaignRequest)
		field  string
	}{
		{"missing name", func(c *CampaignRequest) { c.Name = " " }, "name"},
		{"uppercase source", func(c *CampaignRequest) { c.UTMSource = "Google" }, "utmSource"},
		{"space in campaign", func(c *CampaignRequest) { c.UTMCampaign = "dresden herbst" }, "utmCampaign"},
		{"umlaut in campaign", func(c *CampaignRequest) { c.UTMCampaign = "münchen" }, "utmCampaign"},
		{"missing medium", func(c *CampaignRequest) { c.UTMMedium = "" }, "utmMedium"},
		{"ampersand in content", func(c *CampaignRequest) { c.UTMContent = "a&b" }, "utmContent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validCampaign()
			tc.modify(&req)
			err := req.normalize().validate()
			if err == nil || err.Fields[tc.field] == "" {
				t.Fatalf("expected error on %s, got %v", tc.field, err)
			}
		})
	}
}

func TestCampaignContentAllowsPlatformPlaceholders(t *testing.T) {
	for _, content := range []string{"{{ad.name}}", "{creative}", "video_a-1.2"} {
		req := validCampaign()
		req.UTMContent = content
		if err := req.normalize().validate(); err != nil {
			t.Errorf("content %q rejected: %v", content, err.Fields)
		}
	}
}

func TestBuildLeadLowercasesAttribution(t *testing.T) {
	l := buildLead(CreateLeadRequest{UTMSource: " Google ", UTMMedium: "CPC", UTMCampaign: "Dresden", UTMContent: "Video_A"})
	if l.UTMSource != "google" || l.UTMMedium != "cpc" || l.UTMCampaign != "dresden" {
		t.Errorf("source/medium/campaign not lowercased: %q %q %q", l.UTMSource, l.UTMMedium, l.UTMCampaign)
	}
	if l.UTMContent != "Video_A" {
		t.Errorf("content must keep its case, got %q", l.UTMContent)
	}
}
