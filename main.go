package main

import (
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"time"
)

func main() {
	// URL to make GET request to

	url := os.Getenv("BASE_URL")

	// Make the GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making GET request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Print status information
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Println("---")

	// Read and print the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Response Body:")
	fmt.Println(string(body))

	// Extract YouTube URL from the HTML response
	result := extractYouTubeURL(string(body))
	fmt.Println("\n--- YouTube URL Extraction ---")
	fmt.Printf("Extracted URL: %s\n", result.ExtractedURL)
	fmt.Printf("Video ID: %s\n", result.VideoID)
	fmt.Printf("Success: %t\n", result.Success)
	fmt.Printf("HTML Length: %d\n", result.Debug.HTMLLength)
	fmt.Printf("Found Watch Endpoint: %t\n", result.Debug.FoundWatchEndpoint)
	fmt.Printf("Found YtUrl: %t\n", result.Debug.FoundYtUrl)
	recipients := []string{
		"coding.akhandagarwal6174@gmail.com",
		"akhand.yogaeveryday@gmail.com",
		// "krishnapriya24698@gmail.com",
		// "kkpmzp2000@gmail.com",
		// "badriprasad7571@gmail.com",
		// "murali.kummitha@gmail.com",
	}
	for _, email := range recipients {
		err = sendEmail(result.ExtractedURL, result.VideoID, email)
		if err != nil {
			fmt.Printf("error while sending email: %+v", err)
		}
		fmt.Printf("Email sent to : %s\n", email)
	}
}

type ExtractionResult struct {
	ExtractedURL string
	VideoID      string
	Success      bool
	Debug        DebugInfo
}

type DebugInfo struct {
	HTMLLength         int
	FoundWatchEndpoint bool
	FoundYtUrl         bool
}

func extractYouTubeURL(html string) ExtractionResult {
	var videoURL string
	var videoID string

	// Pattern 1: Look for "watchEndpoint":{"videoId":"VIDEO_ID"}
	watchEndpointRegex := regexp.MustCompile(`"watchEndpoint":\s*{\s*"videoId"\s*:\s*"([a-zA-Z0-9_-]+)"`)
	watchEndpointMatch := watchEndpointRegex.FindStringSubmatch(html)

	if len(watchEndpointMatch) > 1 {
		videoID = watchEndpointMatch[1]
		videoURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	} else {
		// Fallback 1: Look for window['ytUrl'] pattern
		ytUrlRegex := regexp.MustCompile(`window\['ytUrl'\]\s*=\s*'\\?\/watch\?v\\?x3d([a-zA-Z0-9_-]+)'`)
		ytUrlMatch := ytUrlRegex.FindStringSubmatch(html)

		if len(ytUrlMatch) > 1 {
			videoID = ytUrlMatch[1]
			videoURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
		} else {
			// Fallback 2: Look for any videoId pattern
			videoIdRegex := regexp.MustCompile(`"videoId"\s*:\s*"([a-zA-Z0-9_-]+)"`)
			videoIdMatch := videoIdRegex.FindStringSubmatch(html)

			if len(videoIdMatch) > 1 {
				videoID = videoIdMatch[1]
				videoURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
			}
		}
	}

	// Set default values if nothing found
	if videoURL == "" {
		videoURL = "No YouTube URL found"
	}
	if videoID == "" {
		videoID = "No video ID found"
	}

	// Check for debug patterns
	ytUrlPattern := regexp.MustCompile(`window\['ytUrl'\]`)
	foundYtUrl := ytUrlPattern.MatchString(html)

	return ExtractionResult{
		ExtractedURL: videoURL,
		VideoID:      videoID,
		Success:      videoURL != "No YouTube URL found",
		Debug: DebugInfo{
			HTMLLength:         len(html),
			FoundWatchEndpoint: len(watchEndpointMatch) > 1,
			FoundYtUrl:         foundYtUrl,
		},
	}
}

func sendEmail(youtubeURL, videoID string, recipient string) error {
	// Email configuration - you'll need to set these environment variables
	smtpServer := os.Getenv("SMTP_SERVER") // e.g., "smtp.gmail.com"
	smtpPort := os.Getenv("SMTP_PORT")     // e.g., "587"

	if time.Now().Weekday() == 0 {
		youtubeURL = "https://me.habuild.in/sunday"
	}

	format := map[int]string{
		0: "Surya Namaskar & Breathing",
		1: "Light Yoga & Breathing",
		2: "Lower Body",
		3: "Upper Body",
		4: "Core & Laughter",
		5: "Mobility & Flexibility",
		6: "Stamina & Meditation",
	}

	// Default values if environment variables are not set
	if smtpServer == "" {
		smtpServer = "smtp.gmail.com"
	}
	if smtpPort == "" {
		smtpPort = "587"
	}

	senderPassword := os.Getenv("EMAIL_PASSWORD")
	senderEmail := "akhand.yogaeveryday@gmail.com"

	// Email content
	subject := fmt.Sprintf("%s YOGA Link <> Akhand", time.Now().Weekday())
	body := fmt.Sprintf(`Hi,

Please find your today's YOGA link.
Choose from time slots: 6:30 AM, 7:30 AM, and 8:30 AM, as well as 5:00 PM, 6:00 PM, and 7:00 PM.

Today's Format: %s

Link: %s

Best regards,
Akhand`, format[int(time.Now().Weekday())], youtubeURL)

	// Compose message
	message := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", recipient, subject, body))

	// SMTP authentication
	auth := smtp.PlainAuth("", senderEmail, senderPassword, smtpServer)

	// Send email
	err := smtp.SendMail(
		smtpServer+":"+smtpPort,
		auth,
		senderEmail,
		[]string{recipient},
		message,
	)

	return err
}
