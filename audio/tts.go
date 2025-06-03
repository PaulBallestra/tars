package audio

import (
	"context"
	"io"
	"log"

	// Pour décoder si OpenAI TTS renvoie du MP3
	"github.com/sashabaranov/go-openai"
)

type TTSProcessor struct {
	client     *openai.Client
	outputChan chan []byte // Chan de bytes PCM
}

func NewTTSProcessor(client *openai.Client, outputChan chan []byte) *TTSProcessor {
	return &TTSProcessor{
		client:     client,
		outputChan: outputChan,
	}
}

func (tp *TTSProcessor) Process(ctx context.Context, text string) {
	if text == "" {
		log.Println("TTS: Texte vide, rien à synthétiser.")
		return
	}

	req := openai.CreateSpeechRequest{
		Model:          openai.TTSModel1, // ou TTSModel1HD
		Input:          text,
		Voice:          openai.VoiceAlloy,              // ou Echo, Fable, Onyx, Nova, Shimmer
		ResponseFormat: openai.SpeechResponseFormatPcm, // Pour obtenir directement du PCM
		// Speed: 1.0, // Facteur de vitesse
		// Si ResponseFormat était Mp3, on aurait besoin de le décoder:
		// ResponseFormat: openai.SpeechResponseFormatMp3,
	}

	log.Printf("TTS: Demande de synthèse vocale pour: \"%s\"", text)
	audioStream, err := tp.client.CreateSpeech(ctx, req)
	if err != nil {
		log.Printf("Erreur création speech OpenAI: %v", err)
		return
	}
	defer audioStream.Close()

	audioBytes, err := io.ReadAll(audioStream)
	if err != nil {
		log.Printf("Erreur lecture stream audio OpenAI: %v", err)
		return
	}

	// Si on demande du PCM: openai.SpeechResponseFormatPcm
	// L'API OpenAI TTS retourne du PCM à 24kHz, 16-bit, mono
	// On suppose que le AudioPlayer est configuré pour ce format.
	// Si ce n'était pas le cas (ex: MP3), il faudrait décoder :
	/*
	   if req.ResponseFormat == openai.SpeechResponseFormatMp3 {
	       mp3Decoder, err := mp3.NewDecoder(bytes.NewReader(audioBytes))
	       if err != nil {
	           log.Printf("Erreur création décodeur MP3: %v", err)
	           return
	       }
	       // Le sample rate du décodeur mp3 dépend du fichier. OpenAI TTS mp3 est à 24kHz.
	       // Pour oto, nous aurions besoin de connaître ce sample rate.
	       // mp3Decoder.SampleRate() vous le donnerait.

	       pcmData, err := io.ReadAll(mp3Decoder)
	       if err != nil {
	           log.Printf("Erreur décodage MP3 en PCM: %v", err)
	           return
	       }
	       audioBytes = pcmData // Remplacer par les données PCM décodées
	        // Mettre à jour le sampleRate si nécessaire pour le Player.
	   }
	*/

	log.Printf("TTS: audio PCM reçu (%d bytes). Envoi au player.", len(audioBytes))
	tp.outputChan <- audioBytes
}
