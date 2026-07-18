package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
	"math/rand"

	cron "github.com/robfig/cron/v3"
	"marketplace/apis"
)

// Product defines product data structure
type Product struct {
	Title       string   `json:"title"`
	Price       string   `json:"price"`
	Category    string   `json:"category"`
	Condition   string   `json:"condition"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Images      []string `json:"images"`
}

func loadEnv() string {
	return os.Getenv("OPENAI_API_KEY")
}

func randomJitter(duration time.Duration) time.Duration {
	min := -15
	max := 15
	jitter := time.Duration(rand.Intn(max-min+1)+min) * time.Minute
	return duration + jitter
}

func randomImagePicker() string {
	files, err := filepath.Glob("images/*")
	if err != nil || len(files) == 0 {
		return ""
	}
	return files[rand.Intn(len(files))]
}

func encodeToBase64(filePath string) string {
	if filePath == "" {
		return ""
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func main() {
	apiKey := loadEnv()
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	aiClient := apis.NewOpenAI(apiKey)
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

	c := cron.New()
	c.AddFunc("0 7,12,17 * * *", func() {
		durations := []time.Duration{7 * time.Hour, 12 * time.Hour, 17 * time.Hour}
		for _, d := range durations {
			timeToWait := randomJitter(d - time.Now().Sub(time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location())))
			time.Sleep(timeToWait)

			title, err := aiClient.GenerateTitle()
			if err != nil {
				log.Printf("Failed to generate title: %v", err)
				title = "Aki Mobil Baru Bergaransi antar pasang gratis"
			}

			desc, err := aiClient.ParaphraseDescription(baseDescription)
			if err != nil {
				log.Printf("Failed to paraphrase description: %v", err)
				desc = baseDescription
			}

			img := randomImagePicker()
			var imgs []string
			if img != "" {
				imgs = append(imgs, encodeToBase64(img))
			}

			product := Product{
				Title:       title,
				Price:       "525",
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
		}
	})

	c.Start()
	select {} // block forever
}
