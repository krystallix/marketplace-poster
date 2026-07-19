package apis

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func logStep(scope, format string, args ...any) {
	log.Printf("\033[36m[%s]\033[0m %s", scope, fmt.Sprintf(format, args...))
}

func logOK(scope, format string, args ...any) {
	log.Printf("\033[32m[%s]\033[0m %s", scope, fmt.Sprintf(format, args...))
}

func logWarn(scope, format string, args ...any) {
	log.Printf("\033[33m[%s]\033[0m %s", scope, fmt.Sprintf(format, args...))
}

func logErr(scope, format string, args ...any) {
	log.Printf("\033[31m[%s]\033[0m %s", scope, fmt.Sprintf(format, args...))
}

func saveCookiesToEnv(cookies []*proto.NetworkCookie) {
	envMap := make(map[string]string)
	content, err := os.ReadFile(".env")
	if err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				envMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	for _, c := range cookies {
		switch c.Name {
		case "c_user":
			envMap["FB_C_USER"] = c.Value
		case "datr":
			envMap["FB_DATR"] = c.Value
		case "fr":
			envMap["FB_FR"] = c.Value
		case "presence":
			envMap["FB_PRESENCE"] = c.Value
		case "sb":
			envMap["FB_SB"] = c.Value
		case "wd":
			envMap["FB_WD"] = c.Value
		case "xs":
			envMap["FB_XS"] = c.Value
		}
	}

	var sb strings.Builder
	for k, v := range envMap {
		sb.WriteString(fmt.Sprintf("%s=%s\n", k, v))
	}
	_ = os.WriteFile(".env", []byte(sb.String()), 0644)
	log.Println("Cookies saved to .env file successfully!")
}

type FacebookPoster struct {
	CUser    string
	Datr     string
	Fr       string
	Presence string
	Sb       string
	Wd       string
	Xs       string
	Headless bool
}

func NewFacebookPoster(cUser, datr, fr, presence, sb, wd, xs string, headless bool) *FacebookPoster {
	return &FacebookPoster{
		CUser:    cUser,
		Datr:     datr,
		Fr:       fr,
		Presence: presence,
		Sb:       sb,
		Wd:       wd,
		Xs:       xs,
		Headless: headless,
	}
}

func findElementByLabels(page *rod.Page, labels ...string) (*rod.Element, error) {
	regexPattern := fmt.Sprintf("/%s/i", strings.Join(labels, "|"))
	labelEl, err := page.Timeout(8*time.Second).ElementR("label", regexPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find label with text matching %q: %v", regexPattern, err)
	}
	inputEl, err := labelEl.Element("input, textarea, div[role='combobox']")
	if err == nil {
		return inputEl, nil
	}
	return labelEl, nil
}

func findElementByAriaAndLabels(page *rod.Page, labels ...string) (*rod.Element, error) {
	for _, l := range labels {
		el, err := page.Timeout(3 * time.Second).Element(fmt.Sprintf(`label[aria-label="%s"]`, l))
		if err == nil {
			return el, nil
		}
	}
	return findElementByLabels(page, labels...)
}

func findButtonByLabels(page *rod.Page, labels ...string) (*rod.Element, error) {
	regexPattern := fmt.Sprintf("/%s/i", strings.Join(labels, "|"))
	el, err := page.Timeout(8*time.Second).ElementR("div[role='button'], button, div", regexPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find button with text matching %q: %v", regexPattern, err)
	}
	return el, nil
}

func findButtonByAriaAndLabels(page *rod.Page, labels ...string) (*rod.Element, error) {
	for _, l := range labels {
		el, err := page.Timeout(3 * time.Second).Element(fmt.Sprintf(`div[aria-label="%s"], button[aria-label="%s"]`, l, l))
		if err == nil {
			return el, nil
		}
	}
	return findButtonByLabels(page, labels...)
}

func formatDescription(text, category string) string {
	parts := strings.Split(text, "...")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	switch {
	case strings.EqualFold(category, "Mobile phones"):
		if len(parts) > 0 {
			parts[0] = "📱 " + parts[0]
		}
		for i := 1; i < len(parts)-1; i++ {
			parts[i] = "🔥 " + parts[i]
		}
		if len(parts) > 1 {
			parts[len(parts)-1] = "📞 " + parts[len(parts)-1]
		}
	default:
		if len(parts) > 0 {
			parts[0] = "✅ " + parts[0]
		}
		for i := 1; i < len(parts)-1; i++ {
			parts[i] = "✅ " + parts[i]
		}
		if len(parts) > 1 {
			parts[len(parts)-1] = "✅ " + parts[len(parts)-1]
		}
	}
	return strings.Join(parts, "\n\n")
}

func groupKeywords() []string {
	value := strings.TrimSpace(os.Getenv("GROUP_KEYWORDS"))
	if value == "" {
		value = "jual,beli,jogja"
	}
	parts := strings.Split(value, ",")
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			keywords = append(keywords, part)
		}
	}
	return keywords
}

func groupMatches(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (f *FacebookPoster) Post(title, price, category, condition, description string, tags []string, images []string) error {
	logStep("BROWSER", "Launching browser for Facebook posting...")
	l := launcher.New().Leakless(true).Headless(f.Headless).NoSandbox(true)
	u, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %v", err)
	}

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://web.facebook.com/").MustWaitLoad().MustWaitDOMStable().MustSetViewport(1920, 1080, 1, false)

	needManualLogin := false
	if f.CUser == "" || f.Xs == "" {
		needManualLogin = true
	} else {
		// Try to set cookies and verify
		cookies := []*proto.NetworkCookieParam{
			{Name: "c_user", Value: f.CUser, Domain: ".facebook.com", Path: "/", HTTPOnly: true, Secure: true, SameSite: "None", Priority: "Medium"},
			{Name: "datr", Value: f.Datr, Domain: ".facebook.com", Path: "/", HTTPOnly: true, Secure: true, SameSite: "None", Priority: "Medium"},
			{Name: "fr", Value: f.Fr, Domain: ".facebook.com", Path: "/", HTTPOnly: true, Secure: true, SameSite: "None", Priority: "Medium"},
			{Name: "presence", Value: f.Presence, Domain: ".facebook.com", Path: "/", HTTPOnly: true, Secure: true, SameSite: "None", Priority: "Medium"},
			{Name: "sb", Value: f.Sb, Domain: ".facebook.com", Path: "/", HTTPOnly: true, Secure: true, SameSite: "None", Priority: "Medium"},
			{Name: "wd", Value: f.Wd, Domain: ".facebook.com", Path: "/", HTTPOnly: true, Secure: true, SameSite: "None", Priority: "Medium"},
			{Name: "xs", Value: f.Xs, Domain: ".facebook.com", Path: "/", HTTPOnly: true, Secure: true, SameSite: "None", Priority: "Medium"},
		}

		if err := browser.SetCookies(cookies); err != nil {
			log.Printf("Failed to set cookies: %v", err)
			needManualLogin = true
		} else {
			logStep("AUTH", "Navigating to Facebook home...")
			page = page.MustNavigate("https://web.facebook.com/").MustWaitLoad().MustWaitDOMStable()
			if hasLogin, _, err := page.Has(`form[data-testid="royal_login_form"]`); err == nil && hasLogin {
				log.Println("Cookies invalid/expired.")
				needManualLogin = true
			}
		}
	}

	if needManualLogin {
		log.Println("Opening headful browser for manual Facebook login...")
		browser.MustClose()

		// Launch in headful mode (headless: false)
		l2 := launcher.New().Leakless(true).Headless(false).NoSandbox(true)
		u2, err := l2.Launch()
		if err != nil {
			return fmt.Errorf("failed to launch headful browser: %v", err)
		}
		browser = rod.New().ControlURL(u2).MustConnect()
		// Redefine defer to close the new browser
		defer browser.MustClose()

		page = browser.MustPage("https://web.facebook.com/").MustWaitLoad().MustWaitDOMStable().MustSetViewport(1920, 1080, 1, false)

		log.Println("Please log in manually in the browser window.")
		// Wait until login is successful (c_user cookie is set)
		for {
			cookiesList, err := page.Cookies([]string{})
			if err == nil {
				hasCUser := false
				for _, c := range cookiesList {
					if c.Name == "c_user" && c.Value != "" {
						hasCUser = true
						break
					}
				}
				if hasCUser {
					log.Println("Detected login! Saving cookies...")
					saveCookiesToEnv(cookiesList)
					// Update current values
					for _, c := range cookiesList {
						switch c.Name {
						case "c_user":
							f.CUser = c.Value
						case "datr":
							f.Datr = c.Value
						case "fr":
							f.Fr = c.Value
						case "presence":
							f.Presence = c.Value
						case "sb":
							f.Sb = c.Value
						case "wd":
							f.Wd = c.Value
						case "xs":
							f.Xs = c.Value
						}
					}
					break
				}
			}
			time.Sleep(2 * time.Second)
		}
	}

	// Navigate to create listing page
	logStep("MARKETPLACE", "Navigating to Facebook Marketplace item creation...")
	page = page.MustNavigate("https://www.facebook.com/marketplace/create/item").MustWaitLoad().MustWaitDOMStable()
	page.MustScreenshot("debug_create_listing.png")

	// 1. Upload Images
	logStep("MEDIA", "Uploading images...")
	logStep("MEDIA", "Finding image file input...")
	fileInput, err := page.Element(`input[type="file"][accept="image/*,image/heif,image/heic"]`)
	if err != nil {
		log.Printf("Failed to find image file input: %v\n", err)
		return fmt.Errorf("failed to find image file input: %v", err)
	}

	var absImages []string
	for _, img := range images {
		absPath, err := filepath.Abs(img)
		if err == nil {
			absImages = append(absImages, absPath)
			log.Printf("Image file added: %s\n", absPath)
		} else {
			absImages = append(absImages, img)
			log.Printf("Image file added (default): %s\n", img)
		}
	}

	if len(absImages) > 0 {
		fileInput.MustSetFiles(absImages...)
		time.Sleep(3 * time.Second) // wait for upload
		log.Println("Images uploaded successfully.")
	} else {
		log.Println("No images to upload!")
	}

	// 2. Input Title
	logStep("FORM", "Inputting title...")

	titleInput, err := findElementByAriaAndLabels(page, "Title", "Judul")
	if err != nil {
		log.Printf("Failed to find title input: %v\n", err)
		return err
	}
	titleInput.MustInput(title)
	logOK("FORM", "Title inserted successfully.")

	// 3. Input Price
	logStep("FORM", "Inputting price...")
	priceInput, err := findElementByAriaAndLabels(page, "Price", "Harga")
	if err != nil {
		return err
	}
	priceInput.MustInput(price)

	// 4. Select Category
	logStep("FORM", "Selecting category...")
	catInput, err := findElementByAriaAndLabels(page, "Category", "Kategori")
	if err != nil {
		return err
	}
	catInput.MustClick()

	time.Sleep(1 * time.Second)
	cats, err := page.Elements(`div[data-visualcompletion="ignore-dynamic"]`)
	var catClicked bool
	if err == nil {
		for _, cat := range cats {
			if strings.EqualFold(strings.TrimSpace(cat.MustText()), category) {
				cat.MustClick()
				catClicked = true
				break
			}
		}
	}

	if !catClicked {
		var catTerms []string
		catTerms = append(catTerms, category)
		if strings.EqualFold(category, "auto parts") {
			catTerms = append(catTerms, "suku cadang", "otomotif")
		}
		catRegex := fmt.Sprintf("/%s/i", strings.Join(catTerms, "|"))
		catOpt, err := page.ElementR("div[role='option'], div[data-visualcompletion='ignore-dynamic'], span, div", catRegex)
		if err == nil {
			catOpt.MustClick()
		} else {
			log.Printf("Warning: failed to select category option: %v\n", err)
		}
	}
	time.Sleep(1 * time.Second)

	// 5. Select Condition
	logStep("FORM", "Selecting condition...")
	condInput, err := findElementByAriaAndLabels(page, "Condition", "Kondisi")
	if err != nil {
		return err
	}
	condInput.MustClick()

	time.Sleep(1 * time.Second)
	options, err := page.Elements(`div[role="option"]`)
	var condClicked bool
	if err == nil {
		for _, option := range options {
			if strings.EqualFold(strings.TrimSpace(option.MustText()), condition) {
				option.MustClick()
				condClicked = true
				break
			}
		}
	}

	if !condClicked {
		var condTerms []string
		condTerms = append(condTerms, condition)
		if strings.EqualFold(condition, "new") {
			condTerms = append(condTerms, "baru", "anyar")
		} else if strings.EqualFold(condition, "used") {
			condTerms = append(condTerms, "bekas", "seken")
		}
		condRegex := fmt.Sprintf("/%s/i", strings.Join(condTerms, "|"))
		condOpt, err := page.ElementR("div[role='option'], span, div", condRegex)
		if err == nil {
			condOpt.MustClick()
		} else {
			log.Printf("Warning: failed to select condition option: %v\n", err)
		}
	}
	time.Sleep(1 * time.Second)

	// 6. Input Description
	logStep("FORM", "Inputting description...")
	descInput, err := findElementByAriaAndLabels(page, "Description", "Keterangan", "Deskripsi")
	if err != nil {
		return err
	}
	formattedDesc := formatDescription(description, category)
	if ta, err := descInput.Element("textarea"); err == nil {
		ta.MustInput(formattedDesc)
	} else {
		descInput.MustInput(formattedDesc)
	}

	// 7. Input Tags
	if len(tags) > 0 {
		logStep("FORM", "Inputting tags...")
		tagsInput, err := findElementByAriaAndLabels(page, "Product tags", "Tag produk")
		if err == nil {
			var ta *rod.Element
			if t, err := tagsInput.Element("textarea, input"); err == nil {
				ta = t
			} else {
				ta = tagsInput
			}

			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					ta.MustInput(tag)
					_ = page.Keyboard.Press(input.Enter)
					time.Sleep(200 * time.Millisecond)
				}
			}
		}
	}

	// 8. Click Next
	logStep("FORM", "Clicking Next...")
	page.MustScreenshot("debug_filled.png")
	nextBtn, err := findButtonByAriaAndLabels(page, "Next", "Selanjutnya")
	if err != nil {
		return err
	}
	nextBtn.MustClick()
	time.Sleep(5 * time.Second)

	// Select Groups (like aronk254/Facebook-Marketplace-Auto-Poster)
	logStep("GROUP", "Selecting groups...")
	keywords := groupKeywords()
	if groups, err := page.Elements(`div[role="checkbox"]`); err == nil && len(groups) > 0 {
		log.Printf("Found %d groups. Selecting groups matching %q...\n", len(groups), strings.Join(keywords, ","))
		selected := 0
		for idx, group := range groups {
			text, _ := group.Text()
			if !groupMatches(text, keywords) {
				continue
			}
			_ = group.Click("left", 1)
			selected++
			log.Printf("Selected group %d: %s\n", idx+1, strings.TrimSpace(text))
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("Selected %d matching groups.\n", selected)
	} else {
		log.Println("No groups/checkboxes found to select.")
	}
	page.MustScreenshot("debug_groups_selected.png")
	if os.Getenv("DRY_RUN") == "true" {
		log.Println("DRY_RUN=true, stopping before Publish.")
		return nil
	}

	// 9. Click Publish
	logStep("PUBLISH", "Clicking Publish...")
	publishBtn, err := findButtonByAriaAndLabels(page, "Publish", "Terbitkan")
	if err != nil {
		return err
	}
	publishBtn.MustClick()

	// Wait for posting to complete
	time.Sleep(30 * time.Second)
	log.Printf("Successfully listed: %s\n", title)

	return nil
}
