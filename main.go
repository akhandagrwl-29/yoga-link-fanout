package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// Resend email structure
type ResendEmail struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text,omitempty"`
}

// Resend response structure
type ResendResponse struct {
	ID string `json:"id"`
}

type ResendError struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

func main() {
	url := os.Getenv("BASE_URL")

	fmt.Printf("BASE_URL: %s\n", url[:5])
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making GET request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Println("---")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		os.Exit(1)
	}

	// Extract YouTube URL from the HTML response
	result := extractYouTubeURL(string(body))
	fmt.Println("\n--- YouTube URL Extraction ---")
	fmt.Printf("Extracted URL: %s\n", result.ExtractedURL)
	fmt.Printf("Video ID: %s\n", result.VideoID)
	fmt.Printf("Success: %t\n", result.Success)
	fmt.Printf("HTML Length: %d\n", result.Debug.HTMLLength)
	fmt.Printf("Found Watch Endpoint: %t\n", result.Debug.FoundWatchEndpoint)
	fmt.Printf("Found YtUrl: %t\n", result.Debug.FoundYtUrl)

	recipientsString := os.Getenv("EMAIL_RECIPIENTS")
	recipients := strings.Split(recipientsString, ",")

	// err = sendEmail(result.ExtractedURL, recipients)
	if err != nil {
		fmt.Printf("error while sending email: %+v", err)
	}
	// fmt.Printf("Email sent to : %+v\n", recipients)
	logsFile := "logs.txt"

	// Load IST timezone
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		fmt.Printf("Error loading IST timezone: %v\n", err)
		return
	}

	// Get current IST time in human-readable format
	currentTime := time.Now().In(loc).Format("2006-01-02 15:04:05")

	// Open file in append mode, create if not exists
	f, err := os.OpenFile(logsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer f.Close()

	// Append the timestamp with a newline
	if _, err := f.WriteString(currentTime + "\n"); err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	fmt.Printf("Appended last processed time (IST): %s\n", currentTime)
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

func sendEmail(youtubeURL string, recipient []string) error {
	// Get Resend API key from environment
	apiKey := os.Getenv("API_KEY")
	// fmt.Printf("API_KEY: %s\n", apiKey[:5])
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY environment variable is not set")
	}

	senderEmail := os.Getenv("EMAIL_USER")
	// fmt.Printf("EMAIL_USER: %s\n", senderEmail[:5])

	senderName := os.Getenv("EMAIL_SENDER_NAME")
	// fmt.Printf("EMAIL_SENDER_NAME: %s\n", senderName[:5])
	if senderEmail == "" {
		return fmt.Errorf("EMAIL_USER environment variable is not set")
	}

	// Special URL for Sunday
	if time.Now().Weekday() == 0 {
		youtubeURL = "https://me.habuild.in/sunday"
	}

	format := map[int]string{
		0: "Surya Namaskar & Breathing", // Sunday
		1: "Light Yoga & Breathing",     // Monday
		2: "Lower Body",                 // Tuesday
		3: "Upper Body",                 // Wednesday
		4: "Core & Laughter",            // Thursday
		5: "Mobility & Flexibility",     // Friday
		6: "Stamina & Meditation",       // Saturday
	}

	colors := map[int]string{
		0: "#EF7722", // Sunday -
		1: "#3D8D7A", // Monday -
		2: "#09122C", // Tuesday -
		3: "#000957", // Wednesday -
		4: "#7F55B1", // Thursday -
		5: "#DC143C", // Friday
		6: "#3E1E68", // Saturday -
	}
	themeColor := colors[int(time.Now().Weekday())]

	bgColors := map[int]string{
		0: "#0BA6DF", // Sunday -
		1: "#A3D1C6", // Monday -
		2: "#E17564", // Tuesday -
		3: "#FFEB00", // Wednesday -
		4: "#FFE1E0", // Thursday -
		5: "#FDEBD0", // Friday -
		6: "#FFACAC", // Saturday -
	}
	bgColour := bgColors[int(time.Now().Weekday())]

	email := ResendEmail{
		From:    fmt.Sprintf("%s <%s>", senderName, senderEmail),
		To:      recipient,
		Subject: fmt.Sprintf("%s YOGA Link", time.Now().Weekday()),
	}

	email.HTML = fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<title>Daily Yoga Session</title>
	</head>
	<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
		<div style="text-align: center; margin-bottom: 30px;">
			<h1 style="color: %[2]s; margin-bottom: 10px;">🧘‍♀️ Daily Yoga Session</h1>
			<p style="color: #666; font-size: 14px;">%s</p>
		</div>
	
		<div style="background-color: #f9f9f9; padding: 20px; border-radius: 8px; margin: 20px 0;">
			<p style="margin-top: 0;">Hi there!</p>
			<p>Please find your today's YOGA link.</p>
	
			<div style="background-color: %[5]s; padding: 15px; border-radius: 6px; margin: 15px 0;">
				<p style="margin: 0; font-weight: bold; color: %[2]s;">📅 Available Time Slots:</p>
				<p style="margin: 5px 0 0 0;">
					<strong>Morning:</strong> 6:30 AM, 7:30 AM, 8:30 AM<br>
					<strong>Evening:</strong> 5:00 PM, 6:00 PM, 7:00 PM
				</p>
			</div>
		</div>
	
		<div style="background-color: %[2]s; color: white; padding: 20px; border-radius: 8px; text-align: center; margin: 20px 0;">
			<h3 style="margin: 0 0 10px 0;">Today's Format: %s</h3>
			<a href="%s" style="display: inline-block; background-color: white; color: %[2]s; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold; margin-top: 10px;">
				🔗 Join Yoga Session
			</a>
		</div>
	
		<div style="text-align: center; margin-top: 30px; padding-top: 20px; border-top: 1px solid #eee;">
			<p style="margin-bottom: 5px;">Best regards,</p>
			<p style="font-weight: bold; color: %[2]s; font-size: 18px; margin: 0;">Akhand</p>
			<p style="font-size: 12px; color: #666; margin-top: 15px;">
				Stay healthy and mindful! 🙏<br>
				This is your daily yoga reminder.
			</p>
		</div>
	</body>
	</html>
	`, time.Now().Format("Monday, January 2, 2006"), themeColor, format[int(time.Now().Weekday())], youtubeURL, bgColour)

	email.Text = fmt.Sprintf(`Daily Yoga Session - %s

Hi there!

Please find your today's YOGA link.

Available Time Slots:
Morning: 6:30 AM, 7:30 AM, 8:30 AM
Evening: 5:00 PM, 6:00 PM, 7:00 PM

Today's Format: %s

Link: %s

Best regards,
Akhand

Stay healthy and mindful! 🙏
This is your daily yoga reminder.`, time.Now().Format("Monday, January 2, 2006"), format[int(time.Now().Weekday())], youtubeURL)

	// Convert to JSON
	jsonData, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode == 200 {
		var resendResp ResendResponse
		if err := json.NewDecoder(resp.Body).Decode(&resendResp); err != nil {
			log.Printf("Email sent to %s, but couldn't parse response", recipient)
		} else {
			log.Printf("Email sent successfully")
		}
		return nil
	} else {
		var resendErr ResendError
		if err := json.NewDecoder(resp.Body).Decode(&resendErr); err != nil {
			return fmt.Errorf("Resend returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("Resend error (%d): %s", resp.StatusCode, resendErr.Message)
	}
}
