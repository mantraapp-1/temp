package workflows

import (
    "time"

    "go.temporal.io/sdk/temporal"
    "go.temporal.io/sdk/workflow"

    "github.com/mantraapp-1/temp/internal/temporal/activities"
)

type TranscriptionInput struct {
    FilePath string
}

func TranscriptionWorkflow(ctx workflow.Context, in TranscriptionInput) (string, error) {
    opts := workflow.ActivityOptions{
        StartToCloseTimeout: time.Hour,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second * 2,
            BackoffCoefficient: 2.0,
            MaximumAttempts:    5,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, opts)

    var text string
    err := workflow.ExecuteActivity(ctx, activities.RunWhisper, in.FilePath).Get(ctx, &text)
    if err != nil {
        return "", err
    }
    return text, nil
}
