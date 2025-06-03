// audio/capture.go
package audio

import (
	"context"
	"log"
	"time"

	// IMPORTANT: Pour utiliser les constantes partagées
	"github.com/gordonklaus/portaudio"
)

type AudioCapturer struct {
	// sampleRate et channels sont déjà là, c'est bien
	sampleRate      int
	channels        int
	frameDurationMs int // Cette valeur sera config.VADFrameDurationMs
	outputChan      chan<- []int16
	stream          *portaudio.Stream
	// isInitialized   bool // Non requis si Initialize/Terminate sont gérés en dehors
}

// NewAudioCapturer: portaudio.Initialize() doit être appelé avant dans main.go
func NewAudioCapturer(sampleRate, channels, frameDurationMs int, outputChan chan<- []int16) (*AudioCapturer, error) {
	if channels != 1 {
		// go-webrtcvad nécessite du mono. On pourrait ajouter une conversion ici si nécessaire.
		log.Printf("TARS AudioCapturer: AVERTISSEMENT - Le VAD attend de l'audio MONO. La capture est configurée pour %d canaux. Assurez-vous que c'est intentionnel ou que la conversion est gérée.", channels)
	}
	if frameDurationMs != 10 && frameDurationMs != 20 && frameDurationMs != 30 {
		log.Printf("TARS AudioCapturer: AVERTISSEMENT - VADFrameDurationMs (%dms) n'est pas une valeur VAD standard (10, 20, 30). Cela causera des erreurs dans le VADProcessor.", frameDurationMs)
	}

	return &AudioCapturer{
		sampleRate:      sampleRate,
		channels:        channels,
		frameDurationMs: frameDurationMs,
		outputChan:      outputChan,
	}, nil
}

func (ac *AudioCapturer) Start(ctx context.Context) {
	// portaudio.Terminate() doit être appelé dans main.go via defer

	framesPerBuffer := int(float64(ac.sampleRate) * float64(ac.frameDurationMs) / 1000.0)
	// buffer pour portaudio, taille totale (samples * cannaux)
	// portaudioCallbackBuffer := make([]int16, framesPerBuffer*ac.channels) // PAS UTILISÉ DIRECTEMENT DANS OpenDefaultStream avec callback

	var err error
	ac.stream, err = portaudio.OpenDefaultStream(
		ac.channels, // input channels
		0,           // output channels (mic only)
		float64(ac.sampleRate),
		framesPerBuffer, // frames per buffer (samples per channel per callback)
		func(in []int16) { // `in` aura une taille de `framesPerBuffer * ac.channels`
			// Si channels > 1, il faudrait ici extraire le premier canal ou moyenner pour obtenir du mono.
			// Pour l'instant, on assume channels = 1, donc len(in) == framesPerBuffer.
			// Si config.Channels était > 1, il faudrait une conversion :
			// monoFrame := make([]int16, framesPerBuffer)
			// for i := 0; i < framesPerBuffer; i++ {
			// 	monoFrame[i] = in[i*ac.channels] // Prendre le premier canal
			// }

			frameCopy := make([]int16, len(in)) // Si mono, len(in) == framesPerBuffer
			copy(frameCopy, in)

			select {
			case ac.outputChan <- frameCopy:
			case <-time.After(15 * time.Millisecond): // Timeout généreux basé sur frameDuration
				log.Println("TARS AudioCapturer: Timeout envoi frame audio vers VAD channel")
			case <-ctx.Done():
				return // Contexte annulé
			}
		},
	)
	if err != nil {
		log.Printf("TARS AudioCapturer: Erreur ouverture flux PortAudio: %v. Essayez de spécifier un périphérique.", err)
		// Lister les périphériques: devices, _ := portaudio.Devices()...
		close(ac.outputChan) // Fermer pour signaler l'échec
		return
	}
	// defer ac.stream.Close() // Sera fait après Stop()

	err = ac.stream.Start()
	if err != nil {
		log.Printf("TARS AudioCapturer: Erreur démarrage flux PortAudio: %v", err)
		ac.stream.Close() // Fermer si Start échoue
		close(ac.outputChan)
		return
	}
	log.Println("TARS AudioCapturer: Capture audio démarrée...")

	<-ctx.Done() // Attendre le signal d'arrêt

	log.Println("TARS AudioCapturer: Arrêt capture audio...")
	if err := ac.stream.Stop(); err != nil {
		log.Printf("TARS AudioCapturer: Erreur arrêt flux PortAudio: %v", err)
	}
	if err := ac.stream.Close(); err != nil {
		log.Printf("TARS AudioCapturer: Erreur fermeture flux PortAudio: %v", err)
	}
	close(ac.outputChan) // Important de fermer le canal quand la capture est finie
	log.Println("TARS AudioCapturer: Capture audio terminée et canal de sortie fermé.")
}
