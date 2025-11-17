package activities

import (
    "bytes"
    "context"
    "fmt"
    "os/exec"
    "strings"
)

func RunWhisper(ctx context.Context, filePath string) (string, error) {
    // Call the Python script; it must be available in the container PATH
    cmd := exec.CommandContext(ctx, "python3", "scripts/whisper_transcribe.py", filePath)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("whisper_transcribe failed: %v, stderr: %s", err, stderr.String())
    }

    text := strings.TrimSpace(stdout.String())
    return text, nil
}
