package main
// at top of main.go
import "github.com/mantraapp-1/temp/internal/temporal/activities"

// …

func registerActivities(w worker.Worker) {
    w.RegisterActivity(activities.RunWhisper)
}
