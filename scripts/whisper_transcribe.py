import sys
import torch
import librosa
from transformers import AutoModelForSpeechSeq2Seq, AutoProcessor, pipeline

def main():
    if len(sys.argv) < 2:
        print("Usage: whisper_transcribe.py path/to/audio.m4a", file=sys.stderr)
        sys.exit(1)

    audio_file = sys.argv[1]

    # Device / dtype
    device = "cuda:0" if torch.cuda.is_available() else "cpu"
    torch_dtype = torch.float16 if torch.cuda.is_available() else torch.float32

    print(f"Using device: {device}", file=sys.stderr)
    print(f"Using dtype: {torch_dtype}", file=sys.stderr)
    print(f"CUDA available: {torch.cuda.is_available()}", file=sys.stderr)
    if torch.cuda.is_available():
        print(f"CUDA device: {torch.cuda.get_device_name(0)}", file=sys.stderr)

    model_id = "openai/whisper-large-v3"

    print("\nLoading model...", file=sys.stderr)
    model = AutoModelForSpeechSeq2Seq.from_pretrained(
        model_id,
        torch_dtype=torch_dtype,
        low_cpu_mem_usage=True,
        use_safetensors=True,
    )
    model.to(device)

    print("Loading processor...", file=sys.stderr)
    processor = AutoProcessor.from_pretrained(model_id)

    print("Creating pipeline...", file=sys.stderr)
    pipe = pipeline(
        "automatic-speech-recognition",
        model=model,
        tokenizer=processor.tokenizer,
        feature_extractor=processor.feature_extractor,
        torch_dtype=torch_dtype,
        device=device,
    )

    print(f"\nTranscribing {audio_file}...", file=sys.stderr)
    print("Loading audio file...", file=sys.stderr)

    audio_array, sampling_rate = librosa.load(audio_file, sr=16000)
    print(f"Audio loaded: {len(audio_array) / sampling_rate:.2f} seconds", file=sys.stderr)

    result = pipe(
        {"array": audio_array, "sampling_rate": sampling_rate},
        generate_kwargs={
            "language": "english",
            "task": "transcribe",
        },
        return_timestamps=True,
    )

    # IMPORTANT:
    # Only print the final text to stdout so Go can read it cleanly
    print(result["text"])

    # If you want timestamps, print them to stderr or write them to a file
    print("\nTimestamps:", file=sys.stderr)
    for chunk in result.get("chunks", []):
        print(
            f"[{chunk['timestamp'][0]:.2f}s - {chunk['timestamp'][1]:.2f}s]: {chunk['text']}",
            file=sys.stderr,
        )


if __name__ == "__main__":
    main()
