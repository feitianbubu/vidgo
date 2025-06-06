package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/feitianbubu/vidgo"
)

func main() {
	config := &vidgo.ProviderConfig{
		BaseURL: "https://api.kuaishou.com",
		APIKey:  "your_access_key,your_secret_key",
		Timeout: 60 * time.Second,
	}

	client, err := vidgo.NewClient(vidgo.ProviderKling, config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Printf("Provider: %s\n", client.GetProviderName())
	fmt.Printf("Supported models: %v\n", client.GetSupportedModels())

	req := &vidgo.GenerationRequest{
		Prompt:       "Birds flying at sunrise in the mountains, animated scene",
		Duration:     5.0,
		Width:        512,
		Height:       512,
		FPS:          30,
		Model:        "kling-v2-master",
		QualityLevel: vidgo.QualityLevelStandard,
	}

	ctx := context.Background()

	fmt.Println("Creating video generation task...")
	resp, err := client.CreateGeneration(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create generation: %v", err)
	}

	fmt.Printf("Task created, ID: %s\n", resp.TaskID)

	fmt.Println("Waiting for video generation to complete...")
	result, err := client.WaitForCompletion(ctx, resp.TaskID, 10*time.Second)
	if err != nil {
		log.Fatalf("Failed to wait for completion: %v", err)
	}

	switch result.Status {
	case vidgo.TaskStatusSucceeded:
		fmt.Printf("Video generation succeeded!\n")
		fmt.Printf("Video URL: %s\n", result.URL)
		fmt.Printf("Format: %s\n", result.Format)
		if result.Metadata != nil {
			fmt.Printf("Duration: %.1f seconds\n", result.Metadata.Duration)
			fmt.Printf("FPS: %d\n", result.Metadata.FPS)
		}
	case vidgo.TaskStatusFailed:
		fmt.Printf("Video generation failed: %s\n", result.Error.Message)
	default:
		fmt.Printf("Unknown status: %s\n", result.Status)
	}

	fmt.Println("\n--- Image-to-Video Example ---")
	imgReq := &vidgo.GenerationRequest{
		Image:    "https://example.com/sample.jpg",
		Prompt:   "Animate this image with natural motion effects",
		Duration: 5.0,
		Width:    512,
		Height:   512,
		Model:    "kling-v2-master",
	}

	imgResp, err := client.CreateGeneration(ctx, imgReq)
	if err != nil {
		log.Printf("Failed to create image-to-video task: %v", err)
	} else {
		fmt.Printf("Image-to-video task created, ID: %s\n", imgResp.TaskID)
	}
}
