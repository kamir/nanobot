# 🚀 GoMikroBot Usage Guide

This document explains how to interact with GoMikroBot and how information flows through the system during a typical session.

## 🏁 Getting Started

### 1. Initialize
Run the onboard command to create the default config and workspace directory structure:
```bash
./gomikrobot onboard
```

### 2. Configuration
The main configuration file is located at `~/.gomikrobot/config.json`.
You can also use environment variables:
- `OPENAI_API_KEY`: Your primary API key.
- `MIKROBOT_AGENTS_MODEL`: Default is `gpt-4o`.

## 📡 Interaction Modes

### CLI Mode (Direct)
Use this for quick interactions or debugging logic:
```bash
./gomikrobot agent -m "Calculate the hash of main.go"
```

### Gateway Mode (Daemon)
Use this to start the persistent bot that listens on channels like WhatsApp:
```bash
./gomikrobot gateway
```
*Note: On first run, it will print a QR code in the terminal for WhatsApp pairing.*

---

## 🌊 Logic Flow

1. **Input**: A message arrives (e.g., from WhatsApp).
2. **Identification**: The system identifies the `SessionID` (e.g., `whatsapp:user_number`).
3. **Drafting**: The **Context Builder** loads the "Soul" (AGENTS.md, etc.) and the recent history from the Session Manager.
4. **Processing**: The **LLM** decides if it needs to act.
5. **Action**: If a tool is called (e.g., `read_file`), GoMikroBot executes it locally and sends the result back to the LLM.
6. **Final Output**: Once the LLM is satisfied, the final response is published to the **Message Bus**.
7. **Delivery**: The respective channel (WhatsApp) picks up the message and sends it to you.

## 🛠️ Managing the Soul
You can change the bot's behavior without recompiling:
- Edit `~/.gomikrobot/workspace/SOUL.md` to change its personality.
- Edit `~/.gomikrobot/workspace/USER.md` to tell it more about yourself.
- The bot will pick up these changes in the very next message.
