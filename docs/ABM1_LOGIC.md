# ABM1 Logic (Answering Machine Mode 1)

This document describes the logic for the "Smart Answering Machine" (ABM1) feature of GoMikroBot.

## Classification Levels

The system classifies every incoming request into one of three categories and responds accordingly via Text and Audio (TTS).

### 1. Emergency (Notfall)
*   **Trigger**: The user message implies an urgent situation, danger, or critical system failure requiring immediate attention.
*   **Response**:
    *   Mirror the understanding of the emergency.
    *   Explicitly ask for confirmation: "Is this really an emergency?"
    *   *Action (Internal)*: Upon confirmation (in follow-up), mark the session as `PRIORITY_EMERGENCY` and alert human operators immediately.
*   **Sample Response (DE)**: "Ich habe verstanden, dass es sich um einen Notfall handelt: [Zusammenfassung]. Bitte bestätige kurz mit 'Ja', wenn dies ein kritischer Notfall ist. Wir melden uns dann umgehend."

### 2. General Assistance (Assistenz)
*   **Trigger**: General questions, research tasks, coding help, or vague requests.
*   **Response**:
    *   Mirror the understood intent.
    *   If vague, ask for precision.
    *   Promise response as soon as possible.
*   **Sample Response (DE)**: "Ich habe verstanden, dass du Unterstützung bei [Thema] benötigst. Wir kümmern uns so schnell wie möglich darum."

### 3. Appointment Request (Terminanfrage)
*   **Trigger**: Requests for meetings, calls, or scheduling specific times/dates.
*   **Response**:
    *   Acknowledge the proposed times.
    *   State that we will check availability and confirm or send counter-proposals.
*   **Sample Response (DE)**: "Danke für die Terminvorschläge. Ich prüfe die Verfügbarkeit und wir bestätigen den Termin oder senden einen Gegenvorschlag."

## Implementation Details

*   **Intent Detection**: Uses a fast LLM call (e.g., GPT-4o-mini or similar) to classify the incoming text before generating the TTS response.
*   **Language Support**: The response logic adheres to the detected language (DE/EN).
