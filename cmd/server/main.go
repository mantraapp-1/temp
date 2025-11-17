package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "time"

    "go.temporal.io/sdk/client"
    "go.temporal.io/sdk/worker"

    "github.com/mantraapp-1/temp/internal/temporal/activities"
    "github.com/mantraapp-1/temp/internal/temporal/workflows"
)

const (
    defaultTemporalAddress   = "temporal:7233"
    defaultTemporalNamespace = "default"
    defaultTaskQueue         = "TRANSCRIBE_QUEUE"
    uploadDir                = "/app/data/uploads"
)

var temporalClient client.Client

func main() {
    addr := envOr("TEMPORAL_ADDRESS", defaultTemporalAddress)
    ns := envOr("TEMPORAL_NAMESPACE", defaultTemporalNamespace)
    tq := envOr("TEMPORAL_TASK_QUEUE", defaultTaskQueue)

    var err error
    temporalClient, err = client.Dial(client.Options{
        HostPort:  addr,
        Namespace: ns,
    })
    if err != nil {
        log.Fatalf("failed to create Temporal client: %v", err)
    }
    defer temporalClient.Close()

    // Worker
    w := worker.New(temporalClient, tq, worker.Options{})

    // Register workflow(s)
    w.RegisterWorkflow(workflows.TranscriptionWorkflow)

    // Register activity functions
    registerActivities(w)

    go func() {
        log.Println("Starting Temporal worker...")
        if err := w.Run(worker.InterruptCh()); err != nil {
            log.Fatalf("unable to start worker: %v", err)
        }
    }()

    // Ensure upload dir exists
    if err := os.MkdirAll(uploadDir, 0o755); err != nil {
        log.Fatalf("failed to create upload dir: %v", err)
    }

    // HTTP handler
    http.HandleFunc("/transcribe", handleTranscribe)

    port := envOr("HTTP_PORT", "8080")
    log.Printf("HTTP server listening on :%s\n", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatalf("http server failed: %v", err)
    }
}

func registerActivities(w worker.Worker) {
    // Each exported func you want to call from workflows goes here
    w.RegisterActivity(activities.RunWhisper)
}

func envOr(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}

func handleTranscribe(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "use POST /transcribe", http.StatusMethodNotAllowed)
        return
    }

    // Max upload size 100MB; tune as needed
    if err := r.ParseMultipartForm(100 << 20); err != nil {
        http.Error(w, "failed to parse form: "+err.Error(), http.StatusBadRequest)
        return
    }

    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "missing file field: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    savedPath, err := saveUploadedFile(file, header)
    if err != nil {
        http.Error(w, "failed to save file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    log.Printf("Uploaded file saved to %s", savedPath)

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
    defer cancel()

    we, err := temporalClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
        ID:        fmt.Sprintf("transcription-%d", time.Now().UnixNano()),
        TaskQueue: envOr("TEMPORAL_TASK_QUEUE", defaultTaskQueue),
    }, workflows.TranscriptionWorkflow, workflows.TranscriptionInput{
        FilePath: savedPath,
    })
    if err != nil {
        http.Error(w, "failed to start workflow: "+err.Error(), http.StatusInternalServerError)
        return
    }

    var text string
    if err := we.Get(ctx, &text); err != nil {
        http.Error(w, "workflow failed: "+err.Error(), http.StatusInternalServerError)
        return
    }

    resp := map[string]string{"text": text}
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(resp)
}

func saveUploadedFile(file multipart.File, hdr *multipart.FileHeader) (string, error) {
    ext := filepath.Ext(hdr.Filename)
    if ext == "" {
        ext = ".m4a"
    }
    filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
    destPath := filepath.Join(uploadDir, filename)

    out, err := os.Create(destPath)
    if err != nil {
        return "", err
    }
    defer out.Close()

    if _, err := io.Copy(out, file); err != nil {
        return "", err
    }

    return destPath, nil
}
