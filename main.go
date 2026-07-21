package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	cron "github.com/robfig/cron/v3"
	"marketplace/apis"
)

// Product defines product data structure
const postingSchedule = "0 7,12,17 * * SUN,TUE,THU,SAT"

type Product struct {
	Title       string   `json:"title"`
	Price       string   `json:"price"`
	Category    string   `json:"category"`
	Condition   string   `json:"condition"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Images      []string `json:"images"`
}

func cliLog(scope, color, format string, args ...any) {
	log.Printf("\033[%sm[%s]\033[0m %s", color, scope, fmt.Sprintf(format, args...))
}

func showCommands() {
	fmt.Println("\033[36mType command and press Enter:\033[0m")
	fmt.Println("  \033[32mR\033[0m - Run Post Immediately")
	fmt.Println("  \033[33mT\033[0m - Dry Run Test")
}

func loadEnv() string {
	_ = godotenv.Load()
	return os.Getenv("OPENAI_API_KEY")
}

func randomJitter(duration time.Duration) time.Duration {
	min := -15
	max := 15
	jitter := time.Duration(rand.Intn(max-min+1)+min) * time.Minute
	return duration + jitter
}

func imagePicker() []string {
	files, err := filepath.Glob("images/*")
	if err != nil || len(files) == 0 {
		return nil
	}
	images := make([]string, 0, len(files))
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif":
			images = append(images, file)
		}
	}
	rand.Shuffle(len(images), func(i, j int) {
		images[i], images[j] = images[j], images[i]
	})
	if len(images) <= 3 {
		return images
	}
	limit := rand.Intn(3) + 3
	if len(images) < limit {
		limit = len(images)
	}
	return images[:limit]
}

func randomPrice() string {
	return fmt.Sprint(rand.Intn(201) + 400)
}

func runPosting(aiClient *apis.OpenAI, fbPoster *apis.FacebookPoster, baseDescription string, tags []string) {
	cliLog("AI", "35", "Generating title...")
	title, err := aiClient.GenerateTitle()
	if err != nil {
		cliLog("AI", "31", "Failed to generate title: %v", err)
		title = "Aki Mobil Baru Bergaransi antar pasang gratis"
	}

	cliLog("AI", "35", "Paraphrasing description...")
	desc, err := aiClient.ParaphraseDescription(baseDescription)
	if err != nil {
		cliLog("AI", "31", "Failed to paraphrase description: %v", err)
		desc = baseDescription
	}

	imgs := imagePicker()
	cliLog("MEDIA", "36", "Selected %d image(s)", len(imgs))

	product := Product{
		Title:       title,
		Price:       randomPrice(),
		Category:    "Auto Parts",
		Condition:   "new",
		Description: desc,
		Tags:        tags,
		Images:      imgs,
	}

	jsonData, err := json.MarshalIndent(product, "", "    ")
	if err == nil {
		fmt.Println(string(jsonData))
	}

	if fbPoster != nil {
		err = fbPoster.Post(product.Title, product.Price, product.Category, product.Condition, product.Description, product.Tags, product.Images)
		if err != nil {
			cliLog("POST", "31", "Failed to post to Facebook: %v", err)
		}
	}

	sched, err := cron.ParseStandard(postingSchedule)
	if err == nil {
		nextRun := sched.Next(time.Now())
		cliLog("SCHEDULE", "36", "Next posting at %s", nextRun.Format("15:04"))
	} else {
		cliLog("SCHEDULE", "36", "Next posting in 12 hours")
	}
}

func main() {
	log.SetOutput(io.MultiWriter(os.Stderr, frontendLogs))
	apiKey := loadEnv()
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	aiClient := apis.NewOpenAI(apiKey)

	fbPoster := apis.NewFacebookPoster(
		os.Getenv("FB_C_USER"),
		os.Getenv("FB_DATR"),
		os.Getenv("FB_FR"),
		os.Getenv("FB_PRESENCE"),
		os.Getenv("FB_SB"),
		os.Getenv("FB_WD"),
		os.Getenv("FB_XS"),
		os.Getenv("HEADLESS") != "false",
	)

	baseDescription := `monggo yang mau cari aki untuk mobil, motor, maupun sepeda listrik

tersedia beberapa merk, jenis 

- baru, bergaransi resmi
- antar pasang gratis

bisa tukar tambah dengan aki mati, aki soak, aki apapun

Alamat : Siswanto Aki 
Kanggotan 21, Pleret, Bantul

tanya-tanya dulu / konsultasi monggo
hub wa 081354007400`

	tags := []string{
		"aspira",
		"aspira-hybrid-ns40z",
		"aki",
		"aki mobil",
		"aki motor",
		"aki sepeda listrik",
		"servis aki",
		"siswantoaki",
		"akimobiljogja",
	}

	if os.Getenv("RUN_ONCE") == "true" {
		runPosting(aiClient, fbPoster, baseDescription, tags)
		return
	}

	startUI(aiClient, fbPoster, baseDescription, tags)

	c := cron.New()
	c.AddFunc(postingSchedule, func() {
		delay := time.Duration(rand.Intn(16)) * time.Minute
		cliLog("SCHEDULE", "36", "Scheduled run triggered, waiting jitter %s", delay)
		time.Sleep(delay)
		runPosting(aiClient, fbPoster, baseDescription, tags)
	})

	c.Start()

	sched, err := cron.ParseStandard(postingSchedule)
	if err == nil {
		nextRun := sched.Next(time.Now())
		cliLog("SCHEDULE", "36", "Next posting at %s", nextRun.Format("15:04"))
	}

	showCommands()
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			switch strings.ToUpper(strings.TrimSpace(scanner.Text())) {
			case "R":
				cliLog("COMMAND", "32", "Running post immediately...")
				runPosting(aiClient, fbPoster, baseDescription, tags)
				showCommands()
			case "T":
				cliLog("COMMAND", "33", "Running dry-run test...")
				oldDryRun := os.Getenv("DRY_RUN")
				_ = os.Setenv("DRY_RUN", "true")
				runPosting(aiClient, fbPoster, baseDescription, tags)
				if oldDryRun == "" {
					_ = os.Unsetenv("DRY_RUN")
				} else {
					_ = os.Setenv("DRY_RUN", oldDryRun)
				}
				showCommands()
			default:
				showCommands()
			}
		}
	}()

	select {} // block forever
}
