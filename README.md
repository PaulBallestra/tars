# TARS - Advanced Voice-Controlled AI Assistant

TARS is an ambitious project aimed at building a highly responsive conversational AI assistant in Go, capable of executing complex actions. The core objective is to simulate and optimize an interactive audio pipeline: 

**User Speaks** → **Voice Activity Detection (VAD)** → **Targeted Audio Capture** → **Speech-to-Text (STT)** → **Large Language Model (LLM)** → [If Function Call: **Route to Action** → **Execute Action** → **Return to LLM for Synthesis**] → **Text-to-Speech (TTS)** → **Bot Speaks** (via system audio output), all while handling interruptions and targeting minimal latency for a seamless user experience, prioritizing local processing whenever possible.

## Table of Contents

- [Detailed Main Objective](#detailed-main-objective)
- [Key Features](#key-features)
- [Detailed Architecture & Data Flow](#detailed-architecture--data-flow)
- [Project Structure](#project-structure)
- [Technologies Used & Planned](#technologies-used--planned)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Current Status & Known Issues](#current-status--known-issues)
- [Roadmap / Future Features](#roadmap--future-features)
- [Action Handling (Function Calling)](#action-handling-function-calling)
- [Planned Discord Integration](#planned-discord-integration)

## Detailed Main Objective

The core of TARS is to create a continuous, low-latency voice interaction loop:

- **Continuous & Intelligent Listening**: The system actively listens to the user via the microphone.
- **Voice Activity Detection (VAD)**: Precisely identifies when the user starts and stops speaking to process only relevant audio segments.
- **Efficient Capture**: Records only the detected speech.
- **Fast Transcription (STT)**: Converts speech to text with minimal latency.
- **Comprehension & Decision (LLM)**: The text is sent to an LLM (e.g., OpenAI GPT) to understand intent. The LLM can:
  - Respond directly.
  - Request clarification.
  - Invoke a "function" or "tool" to perform a specific task.
- **Function Calling & Routing**: If an action is required, the request is routed to the appropriate action module.
- **Action Execution**: The specific tool/action is executed (e.g., creating a Discord event, web search, controlling a smart home device).
- **Response Synthesis (LLM)**: The action's result is sent back to the LLM to formulate a coherent response for the user.
- **Voice Response (TTS)**: The textual response is converted to speech.
- **Seamless Interaction**: The bot responds vocally to the user.
- **Interruption Handling**: The user can interrupt the bot at any time (e.g., during speech or long actions).

Emphasis is placed on minimizing latency at each step and, where feasible, local processing for privacy and responsiveness (though some components, like advanced LLMs, typically rely on cloud services).

## Key Features

- **Real-Time Audio Capture**: Using PortAudio.
- **Voice Activity Detection (VAD)**: Targeting go-webrtcvad for precise detection.
- **Speech-to-Text (STT)**: OpenAI Whisper (cloud) or future local solutions (e.g., Vosk, Whisper.cpp).
- **Natural Language Processing (LLM)**: OpenAI GPT (cloud) or future local solutions (e.g., Llama.cpp, Ollama).
- **Function Calling**: Allows the LLM to request execution of predefined Go code ("tools").
- **Action Routing**: A system (actions/router.go) to direct function call requests to the appropriate executors.
- **Modular Action Executors**: (actions/executors/) Each tool has its own execution code.
- **Text-to-Speech (TTS)**: OpenAI TTS (cloud) or future local solutions (e.g., Piper, CoquiTTS).
- **Interruption Handling**: Allows users to speak over the bot.
- **(Planned) Discord Integration**: Ability to interact with Discord servers (create channels, events, post messages) via specific tools.
- **(Planned) Wake Word Detection**: To activate TARS vocally.

## Detailed Architecture & Data Flow

```plaintext
┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐
│   User Speaks     │---->│   Microphone   ├----->│   Audio I/O    │
└───────────────────┘ └───────────────────┘ │   (PortAudio)    │
                                    └─────────┬─────────┘
                                              │ Audio PCM Stream
                                              ▼
                                    ┌───────────────────┐
                                    │   VAD Processor   │
                                    │  (go-webrtcvad)   │
                                    └─────────┬─────────┘
                                              │ Speech Segments
                                              ▼
                                    ┌───────────────────┐
                                    │ Audio Accumulator │
                                    │ & Formatter (WAV) │
                                    └─────────┬─────────┘
                                              │ Audio Data (WAV)
                                              ▼
                                    ┌───────────────────┐
                                    │   STT Processor   │
                                    │ (OpenAI Whisper)  │
                                    └─────────┬─────────┘
                                              │ User Text
                                              ▼
                                    ┌───────────────────┐
                                    │   LLM Processor   │
                                    │   (OpenAI GPT)    │
                                    │ - Intent Recogn.  │
                                    │ - Function Call?   │
                                    └─────────┬─────────┘
                                              │ Decision
┌─────────────────────────────────────────────────────────┼──────────────────────────────────────────────────────────┐
│ LLM Decides Action                                      │ LLM Decides Direct Response                       │
▼                                                       ▼                                                  │
┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐ ┌────────────────────────────────────────┐ │
│      Router       │---->│ Executor (Tool A) │--┐ │ Executor (Tool B) │--┐ │ ... (Other tools/actions)      │ │
│(actions/router.go)│ │(ex: Discord Chan) │ └─>│ (ex: Web Search)  │ └─>│ (ex: Discord Event, Home Auto) │ │
└───────────────────┘ └───────────────────┘ └───────────────────┘ └────────────────────────────────────────┘ │
                │                         │ Action Result                                            │
                └────────────────────────────────────────────────────┐───────────────────────────────────────────────────────────────┘
                                              │
                                              ▼
                                    ┌───────────────────┐
                                    │   LLM Processor   │
                                    │ (Response Synthesis│
                                    │ with action result │
                                    │   if needed)      │
                                    └─────────┬─────────┘
                                              │ Bot Text Response
                                              ▼
                                    ┌───────────────────┐
                                    │   TTS Processor   │
                                    │ (OpenAI TTS /     │
                                    │ Local Solution)    │
                                    └─────────┬─────────┘
                                              │ Bot Speech Audio
                                              ▼
                                    ┌───────────────────┐ ┌───────────────────┐
                                    │   Audio Output    │----->│   Speakers     │
                                    │   (PortAudio)     │ └───────────────────┘
                                    └───────────────────┘
```

**Interruption Handling**: A parallel process checks for user speech to interrupt the bot's TTS or long actions.

## Project Structure

```plaintext
tars/
├── actions/                    # Manages "function calls" / tools
│   ├── router.go               # Routes function calls to executors
│   └── executors/              # Modules for specific actions/tools
│       ├── discord_channel_creator.go (example)
│       ├── discord_event_creator.go (example)
│       └── template_executor.go (for new tools)
├── audio/                      # Modules for audio capture, VAD, and output
│   ├── capturer.go
│   ├── player.go               # For playing TTS
│   └── vad_processor.go
├── stt/                        # Speech-to-Text modules
│   └── openai_stt.go           # (and/or stt_local.go)
├── llm/                        # Large Language Model interaction modules
│   └── openai_llm.go           # (and/or llm_local.go)
├── tts/                        # Text-to-Speech modules
│   └── openai_tts.go           # (and/or tts_local.go)
├── config/                     # Global project configuration
│   └── config.go
├── utils/                      # Shared utility functions
│   └── audio_conversion.go     # (e.g., for PCM <> WAV conversion)
├── main.go                     # Application entry point
├── go.mod
├── go.sum
└── README.md
```

## Technologies Used & Planned

- **Language**: Go
- **Audio I/O**: `github.com/gordonklaus/portaudio`
- **VAD (target)**: `github.com/maxhawkins/go-webrtcvad`
- **STT**:
  - Initial: OpenAI Whisper API (via `github.com/sashabaranov/go-openai`)
  - Future: Exploration of local solutions (Vosk, Whisper.cpp via CGo or Go bindings)
- **LLM**:
  - Initial: OpenAI GPT API (via `github.com/sashabaranov/go-openai`)
  - Future: Exploration of local solutions (Llama.cpp, Ollama with Go bindings)
- **TTS**:
  - Initial: OpenAI TTS API (via `github.com/sashabaranov/go-openai`)
  - Future: Exploration of local solutions (Piper, CoquiTTS via CGo or Go bindings, gTTS)
- **Audio Encoding**: `github.com/go-audio/wav`, `github.com/go-audio/audio`
- **(Potential for Discord)**: `github.com/bwmarrin/discordgo`

## Prerequisites

- **Go**: Version 1.21+ (ideally 1.22+ for certain optimizations).
- **PortAudio Library**: Installed on your system (refer to installation instructions).
- **OpenAI Account and API Key**: Required for initial STT, LLM, and TTS modules.
- **(Optional, for WebRTC VAD)**: A C/C++ compiler (gcc/clang) as go-webrtcvad is a C wrapper.

## Installation

1. **Clone the repository**:
   ```bash
   git clone [TARS_REPOSITORY_URL]
   cd tars
   ```

2. **Ensure system prerequisites are installed**.

3. **Configure your OpenAI API key** (see Configuration section).

4. **Download Go dependencies**:
   ```bash
   go mod tidy
   ```

   *(If issues with go-webrtcvad persist, a specific version or fork may be required.)*

## Configuration

The main configuration is in `config/config.go`. API keys (OpenAI, Discord bot token if applicable) must be stored securely. **DO NOT COMMIT API KEYS TO A PUBLIC REPOSITORY**. Use environment variables or a secrets management system.

Example environment variables:
```bash
export OPENAI_API_KEY="sk-yourkey"
export DISCORD_BOT_TOKEN="yourtoken" # (If/when Discord is implemented)
```

The code in `config.go` should read these variables (e.g., `os.Getenv("OPENAI_API_KEY")`).

Other configurable settings:
- Input/output audio devices.
- STT/LLM/TTS models.
- VAD parameters (aggressiveness, timeouts).
- Interruption handling thresholds.

## Usage

```bash
go run main.go
```

The goal is natural voice interaction. If a wake word is implemented:
- Say the wake word (e.g., "Hey TARS").
- The system starts actively listening.
- Speak your request.
- TARS processes and responds.

Without a wake word (or in push-to-talk mode for debugging):
- An interaction (e.g., key press) may be required to start the listening session.

## Current Status & Known Issues

- **Basic Audio Pipeline (Manual)**: Capture -> STT -> LLM -> Console Text Response is functional with manual recording triggers.
- **VAD with go-webrtcvad**: Blocked due to dependency issues. Workarounds (simulated VAD, fixed recording) are used for developing other modules. Resolving this is a high priority for the target pipeline.
- **TTS**: Initial implementation with OpenAI TTS in progress/planned.
- **Function Calling & Routing**: Architecture defined (actions/router.go, actions/executors/), implementation in progress.
- **Interruption Handling**: Conceptual, not implemented.
- **Discord Integration**: Planned, not started.
- **Latency Optimization**: Ongoing work. Each component (local vs. cloud, library choices) impacts latency.

## Roadmap / Future Features

- **Priority #1**: Resolve and integrate go-webrtcvad or find a robust Go-native VAD alternative.
- **Implement the full continuous audio pipeline**: VAD -> STT -> LLM -> TTS with silence handling.
- **Implement interruption handling (barge-in)**.
- **Finalize Function Calling architecture**:
  - actions/router.go
  - Example executors in actions/executors/
  - LLM <-> Router <-> Executor <-> LLM communication.
- **Integrate OpenAI TTS for vocal responses**.
- **Implement Discord tools**:
  - Channel creation.
  - Event creation.
  - Other relevant Discord actions.
- **Wake Word Detection**: Use libraries like Porcupine, Picovoice, or explore open-source solutions.
- **Explore/Integrate local STT/LLM/TTS models** to reduce cloud dependency and latency.
- **Advanced conversational context management**.
- **Configuration interface** (CLI or simple GUI) for audio devices, API keys, etc.
- **Unit and integration tests**.
- **Packaging/Distribution** (if the project matures).

## Action Handling (Function Calling)

The LLM (e.g., OpenAI GPT) can identify when a user requests an action beyond simple conversation, generating a "function call" structure describing the action and its parameters.

- **LLM**: Identifies an action intent and generates a request (e.g., `{ "name": "createDiscordChannel", "arguments": { "server_id": "123", "channel_name": "new-channel" } }`).
- **tars/llm/openai_llm.go**: Detects the function call request.
- **tars/actions/router.go**:
  - Receives the request from the LLM.
  - Analyzes the function name.
  - Calls the corresponding executor (e.g., a function in `tars/actions/executors/discord_channel_creator.go`).
- **tars/actions/executors/specific_file.go**:
  - Contains logic to execute the actual action (e.g., calling the Discord API).
  - Returns the action result (success, failure, data) to the router.
- **Router**: Sends the result back to `openai_llm.go`.
- **openai_llm.go**: Forwards the result to the OpenAI LLM to formulate a final user response based on the action's outcome.

This approach keeps action-specific code organized while allowing the LLM flexibility to decide when and how to use actions.

## Planned Discord Integration

TARS will function as a Discord bot, implemented as a set of "tools" the LLM can invoke. Example Discord tools:

- **create_discord_channel**: Creates a new text or voice channel.
- **create_discord_event**: Schedules a new event on the server.
- **send_discord_message**: Sends a message to a specific channel.
- **get_discord_server_info**: Retrieves server information.

These tools will be Go modules in `actions/executors/` using a library like `github.com/bwmarrin/discordgo` to interact with the Discord API. A Discord bot token will be required in the configuration.