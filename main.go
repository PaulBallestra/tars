// main.go (ou autre)
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time" // Pour le goroutine principale d'exemple

	"tars/audio"
	"tars/config"

	// "tars/llm"
	// "tars/stt"

	"github.com/gordonklaus/portaudio"
)

// Helper pour convertir PCM []int16 (mono) en []byte pour le VAD
func pcm16MonoToBytes(pcm []int16) []byte {
	buf := make([]byte, len(pcm)*2) // BytesPerSample = 2
	for i, s := range pcm {
		buf[i*2] = byte(s & 0xFF)          // Little-endian LSB
		buf[i*2+1] = byte((s >> 8) & 0xFF) // Little-endian MSB
	}
	return buf
}

func main() {
	log.Println("Démarrage de TARS...")

	// --- Initialisation PortAudio (UNE FOIS) ---
	if err := portaudio.Initialize(); err != nil {
		log.Fatalf("TARS: Erreur initialisation PortAudio: %v", err)
	}
	defer portaudio.Terminate() // Assurer la terminaison à la fin de main
	log.Println("TARS: PortAudio initialisé.")
	// --- Fin Initialisation PortAudio ---

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Gestion des signaux d'arrêt (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("TARS: Signal d'arrêt reçu, nettoyage...")
		cancel()
	}()

	// --- Canaux de communication ---
	audioFromCaptureChan := make(chan []int16, 50) // Buffer pour les frames capturées (int16)
	//vadDetectChan := make(chan bool)             // true si speech, false si silence (ou pour fin de parole)
	speechStopChan := make(chan bool, 1) // Pour signaler la *fin* de la parole (bufferisé)
	// textFromSTTChan := make(chan string)
	// llmResponseChan := make(chan string)
	// audioPCMForPlayerChan := make(chan []byte)

	// --- Modules ---
	// 1. AudioCapturer
	// Utilise VADFrameDurationMs de config car le VAD traitera ces frames.
	// Channels DOIT être 1 pour go-webrtcvad.
	capturer, err := audio.NewAudioCapturer(
		config.SampleRate,
		config.Channels, // DOIT être 1 pour VAD actuel
		config.VADFrameDurationMs,
		audioFromCaptureChan,
	)

	if err != nil {
		log.Fatalf("TARS: Erreur création AudioCapturer: %v", err)
	}
	go capturer.Start(ctx) // Démarre la capture dans une goroutine

	// 2. VADProcessor
	/*
		vadProcessor, err := audio.NewVADProcessor(speechStopChan)
		// speechStopChan est pour indiquer la fin de parole
		if err != nil {
			log.Fatalf("TARS: Erreur création VADProcessor: %v", err)
		}
		defer vadProcessor.Close() // Pas d'action pour go-webrtcvad mais bonne pratique
	*/

	// Boucle principale de traitement de l'audio capturé pour VAD
	/*
		go func() {
			for {
				select {
				case <-ctx.Done():
					log.Println("TARS VAD Loop: Contexte annulé, arrêt.")
					return
				case pcmFrame16, ok := <-audioFromCaptureChan:
					if !ok {
						log.Println("TARS VAD Loop: Canal de capture fermé.")
						return // Le canal de capture est fermé, la goroutine doit s'arrêter
					}

					// Assurer que la frame est mono. Si config.Channels > 1, il faudrait une conversion ici.
					// Pour l'instant, on suppose config.Channels = 1.
					if len(pcmFrame16) == 0 {
						// log.Println("TARS VAD Loop: Frame PCM16 vide reçue, ignorée.")
						continue
					}

					// Convertir []int16 en []byte pour le VAD
					byteFrame := pcm16MonoToBytes(pcmFrame16)

					// Passer au VAD
					// Le VAD a besoin de connaître le SampleRate avec lequel les données ont été capturées
					isSpeaking, err := vadProcessor.Process(byteFrame, config.SampleRate)
					if err != nil {
						log.Printf("TARS VAD Loop: Erreur traitement VAD: %v", err)
						// Décider quoi faire en cas d'erreur VAD, peut-être continuer
						continue
					}

					if isSpeaking {
						// log.Printf("TARS VAD Loop: Parole détectée.")
						// Ici, vous accumuleriez `byteFrame` pour STT jusqu'à ce que `speechStopChan` reçoive un signal
					}
					// La logique de "fin de parole" (envoyer sur speechStopChan) est gérée DANS vadProcessor.Process()
				}
			}
		}()
	*/

	// Boucle de gestion de la fin de parole (exemple)
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("TARS Speech Handler: Contexte annulé, arrêt.")
				return
			case <-speechStopChan:
				log.Println("TARS Speech Handler: FIN DE PAROLE DÉTECTÉE (par VAD). C'est ici qu'on lancerait STT.")
				// Logique pour prendre l'audio accumulé et l'envoyer au STT
			}
		}
	}()

	log.Println("TARS est initialisé et à l'écoute. Appuyez sur Ctrl+C pour quitter.")
	// Garder le programme principal en vie jusqu'à ce que le contexte soit annulé
	<-ctx.Done()

	log.Println("TARS: Programme principal en cours d'arrêt...")
	// Attendre un peu pour que les goroutines aient une chance de se nettoyer si nécessaire,
	// bien que ctx.Done() devrait le gérer.
	time.Sleep(1 * time.Second)
	log.Println("TARS: Arrêt terminé.")
}
