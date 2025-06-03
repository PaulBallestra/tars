package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"log"
	"tars/config" // Pour SampleRate, Channels

	"github.com/sashabaranov/go-openai"
)

type STTProcessor struct {
	client     *openai.Client
	outputChan chan string
}

func NewSTTProcessor(client *openai.Client, outputChan chan string) *STTProcessor {
	return &STTProcessor{
		client:     client,
		outputChan: outputChan,
	}
}

// createWavInMemory prend des données PCM brutes et les enveloppe dans un header WAV.
// Les données PCM doivent être en 16-bit little-endian.
func createWavInMemory(pcmData []byte, sampleRate, channels, bitDepth int) (*bytes.Reader, error) {
	buf := new(bytes.Buffer)
	// RIFF header
	buf.WriteString("RIFF")
	// Placeholder pour la taille du chunk (taille totale du fichier - 8 bytes)
	// Sera rempli plus tard
	binary.Write(buf, binary.LittleEndian, uint32(0))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16)) // Taille du sous-chunk fmt (16 pour PCM)
	binary.Write(buf, binary.LittleEndian, uint16(1))  // Format audio (1 pour PCM)
	binary.Write(buf, binary.LittleEndian, uint16(channels))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	byteRate := sampleRate * channels * (bitDepth / 8)
	binary.Write(buf, binary.LittleEndian, uint32(byteRate))
	blockAlign := channels * (bitDepth / 8)
	binary.Write(buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(buf, binary.LittleEndian, uint16(bitDepth))

	// data chunk
	buf.WriteString("data")
	// Placeholder pour la taille du sous-chunk data (taille des données PCM)
	// Sera rempli plus tard
	binary.Write(buf, binary.LittleEndian, uint32(0))

	// Écrire les données PCM réelles
	pcmLen, err := buf.Write(pcmData)
	if err != nil {
		return nil, err
	}

	// Revenir en arrière et remplir les tailles
	finalBytes := buf.Bytes()
	binary.LittleEndian.PutUint32(finalBytes[4:], uint32(len(finalBytes)-8))
	binary.LittleEndian.PutUint32(finalBytes[40:], uint32(pcmLen))

	return bytes.NewReader(finalBytes), nil
}

func (sp *STTProcessor) Process(ctx context.Context, pcmData []byte) {
	if len(pcmData) == 0 {
		log.Println("STT: Aucune donnée PCM à traiter.")
		return
	}

	// Créer un fichier WAV en mémoire
	wavReader, err := createWavInMemory(pcmData, config.SampleRate, config.Channels, config.BitDepth)
	if err != nil {
		log.Printf("Erreur création WAV en mémoire: %v", err)
		return
	}

	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: "recording.wav", // Nom de fichier pour l'API, pas un vrai fichier ici
		Reader:   wavReader,       // Utilisation de l'io.Reader pour les données en mémoire
		// Language: "fr", // Facultatif: Spécifier la langue
	}

	log.Println("STT: Envoi de l'audio à OpenAI Whisper...")
	resp, err := sp.client.CreateTranscription(ctx, req)
	if err != nil {
		log.Printf("Erreur transcription OpenAI: %v", err)
		// TODO: Gérer les erreurs API (limites de taux, etc.)
		// Si erreur de type *openai.APIError, vous pouvez vérifier resp.Error.HTTPStatusCode
		return
	}

	log.Printf("STT: Texte reçu: %s", resp.Text)
	sp.outputChan <- resp.Text
}
