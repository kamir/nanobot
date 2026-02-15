# Implementation Plan: Text-to-Speech (TTS) & Audio Acknowledgement

## Objective
Enable GoMikroBot to respond to incoming messages with a polite, spoken acknowledgement. This enhances the "Coworker" persona, making interactions feel more personal and confirming understanding before executing complex tasks.

## User Request
> "Respond to the message using a simple Text and an audio track telling the caller what we understood and politely telling that we handle the question."

## Architecture

### 1. TTS Provider Interface
We will extend the `LLMProvider` interface (or create a dedicated `TTSProvider`) to include a `Speak` method.

```go
type TTSRequest struct {
    Text  string
    Voice string // e.g., "alloy", "echo", "shimmer"
    Speed float64
}

type TTSResponse struct {
    AudioData []byte // Raw audio bytes (MP3/Opus)
    Format    string // "mp3", "opus"
}

type LLMProvider interface {
    // ... existing Chat/Transcribe ...
    Speak(ctx context.Context, req *TTSRequest) (*TTSResponse, error)
}
```

### 2. Provider Implementation
We will implement this primarily using **OpenAI's TTS API** (`tts-1` model) for high-quality, human-like vocals.
*   **Backup Option**: Mac OS `say` command (Local, robotic but free/private). We can implement a `LocalTTSProvider` if desired.

### 3. WhatsApp Integration (`internal/channels/whatsapp.go`)
We will add a workflow step in the `eventHandler`:
1.  **Understand**: Summarize the incoming intent (using a fast LLM call or regex).
2.  **Ack Text**: Send a text reply: *"Message received. I am checking [Topic] for you..."*
3.  **Ack Audio**:
    *   Send the text to `Speak()`.
    *   Convert MP3 to OGG (Opus) using `ffmpeg` (required for WhatsApp Voice Notes to play correctly with waveforms).
    *   Upload and send the Voice Note.
4.  **Execute**: Proceed with the actual agent tool execution.

## Configuration
Update `config.json` to include TTS settings:

```json
"tts": {
  "enabled": true,
  "provider": "openai", // or "local"
  "voice": "nova",
  "ack_template": "I have received your request regarding %s. Working on it now."
}
```

## Dependencies
*   `ffmpeg`: Already installed on the system (verified).
*   `whatsmeow`: Supports media upload.
*   `provider`: Needs update.

## Execution Steps
1.  **Update Interface**: Add `Speak` to `provider.go`.
2.  **Implement OpenAI TTS**: Add logic to `openai.go`.
3.  **Update Config**: Add TTS section.
4.  **Implement Logic in WhatsApp**: Update `whatsapp.go` to handle the flow.
