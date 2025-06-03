package config

import "os"

var (
	OpenAIAPIKey       string
	SampleRate         = 16000 // Pour la CAPTURE micro (Whisper)
	Channels           = 1     // Pour la CAPTURE micro
	BitDepth           = 16    // Pour la CAPTURE micro (PCM 16-bit)
	VADFrameDurationMs = 20    // ms, pour VAD (avec 16kHz, donne 320 samples / 640 bytes)
	VADSilenceFrames   = 25    // Nombre de frames silence avant de considérer fin de parole (25 * 20ms = 500ms)
	VADSpeechFrames    = 3     // Nombre de frames de parole avant de commencer à enregistrer (3 * 20ms = 60ms)
	VADAggressiveness  = 2     // 0 (least aggressive) à 3 (most aggressive)

	// Note: Le TTS OpenAI (PCM) sort à 24kHz, 1 canal, 16-bit.
	// Le AudioPlayer doit être configuré avec ces valeurs.
	TTSSampleRate = 24000
	TTSChannels   = 1
)

func LoadConfig() {
	OpenAIAPIKey = os.Getenv("OPENAI_API_KEY")
	if OpenAIAPIKey == "" {
		// Pour tests locaux, on peut le hardcoder mais ce n'est pas recommandé
		// OpenAIAPIKey = "votre_cle_api_ici"
		// if OpenAIAPIKey == "votre_cle_api_ici" {
		//  log.Println("ATTENTION: Clé API OpenAI hardcodée dans config.go")
		// } else {
		panic("La variable d'environnement OPENAI_API_KEY n'est pas définie.")
		// }
	}
}
