package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func ScrapeHomepage(ctx context.Context) ([]CompanyPreview, error) {
	var result []interface{}

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://startups.gallery"),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a[href*="/companies/"]')).map(card => {
				const href = card.getAttribute('href');
				const slug = href.split('/').pop();
				const h3 = card.querySelector('h3');
				const p = card.querySelector('p');
				return {
					slug: slug,
					name: h3 ? h3.innerText.trim() : '',
					tagline: p ? p.innerText.trim() : ''
				};
			})
		`, &result),
	)

	if err != nil {
		return nil, err
	}

	companies := []CompanyPreview{}
	for _, item := range result {
		if company, ok := item.(map[string]interface{}); ok {
			slug, _ := company["slug"].(string)
			name, _ := company["name"].(string)
			tagline, _ := company["tagline"].(string)

			companies = append(companies, CompanyPreview{
				Slug:    slug,
				Name:    name,
				Tagline: tagline,
			})
		}
	}

	seen := make(map[string]bool)
	unique := []CompanyPreview{}
	for _, c := range companies {
		if !seen[c.Slug] && c.Slug != "" {
			seen[c.Slug] = true
			unique = append(unique, c)
		}
	}

	return unique, nil
}
func ScrapeCompany(ctx context.Context, slug string) (*Startup, error) {
	companyURL := fmt.Sprintf("https://startups.gallery/companies/%s", slug)
	var result map[string]interface{}
	err := chromedp.Run(ctx,
		chromedp.Navigate(companyURL),
		chromedp.Sleep(5*time.Second),
		chromedp.Evaluate(`
            ({
                title: document.title,
                bodyText: document.body.innerText,
                links: Array.from(document.querySelectorAll('a')).map(a => ({
                    text: a.innerText.trim(),
                    href: a.href
                })),
                images: Array.from(document.querySelectorAll('img')).map(img => ({
                    src: img.src,
                    alt: img.alt
                }))
            })
        `, &result),
	)
	if err != nil {
		return nil, err
	}
	return parseCompanyData(slug, result), nil
}

func parseCompanyData(slug string, result map[string]interface{}) *Startup {

	bodyText := result["bodyText"].(string)
	title := result["title"].(string)
	links := result["links"].([]interface{})
	images := result["images"].([]interface{})
	startup := &Startup{
		Name:         strings.TrimSuffix(title, " | startups.gallery"),
		Slug:         slug,
		FullyScraped: true,
		LastFetched:  time.Now(),
	}
	if len(images) > 0 {
		if img, ok := images[0].(map[string]interface{}); ok {
			startup.BannerURL = img["src"].(string)
		}
	}

	if len(images) > 1 {
		if img, ok := images[1].(map[string]interface{}); ok {
			startup.LogoURL = img["src"].(string)
		}
	}

	for _, link := range links {
		linkMap := link.(map[string]interface{})
		text := strings.TrimSpace(linkMap["text"].(string))
		href := linkMap["href"].(string)
		if text == "Visit Website" {
			startup.WebsiteURL = href
		} else if text == "View Jobs" {
			startup.JobsURL = href
		}
		if strings.Contains(href, "/categories/locations/") {
			startup.Location = text
		} else if strings.Contains(href, "/categories/stages/") {
			startup.FundingStage = text
		} else if strings.Contains(href, "/categories/industries/") {
			startup.Industry = text
		} else if strings.Contains(href, "/categories/work-type/") {
			startup.WorkType = text
		}
	}
	lines := strings.Split(bodyText, "\n")
	foundName := false
	fundingAnnouncementRegex := regexp.MustCompile(`^Raised \$[\d.]+[MBK]+ (Seed|Series [A-Z]|Pre-Seed|Venture) on .+$`)
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == startup.Name {
			foundName = true
			continue
		}

		if startup.FundingAnnouncement == "" && fundingAnnouncementRegex.MatchString(line) {
			startup.FundingAnnouncement = line
		}

		if foundName && startup.Tagline == "" {
			if strings.Contains(line, "Visit") ||
				strings.Contains(line, "View") ||
				strings.Contains(line, "Raised") ||
				strings.Contains(line, "Backed by") ||
				strings.Contains(line, "Get Updates") ||
				len(line) == 0 ||
				line == "·" {
				continue
			}

			if len(line) > 1 && len(line) < 100 {
				startup.Tagline = line
				continue
			}
		}

		if startup.Description == "" &&
			len(line) > 100 &&
			!strings.Contains(line, "Raised") &&
			!strings.Contains(line, "Posted on") &&
			!strings.Contains(line, "Explore similar") {
			startup.Description = line
			break
		}
	}

	teamSizeRegex := regexp.MustCompile(`\b(\d+[-–]\d+)\b`)
	if match := teamSizeRegex.FindStringSubmatch(bodyText); len(match) > 1 {
		startup.TeamSize = match[1]
	}
	return startup
}
