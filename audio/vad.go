// Fichier: audio/vad.go (APRÈS - Option 1: Pas de VAD)
package audio

import "fmt" // Plus besoin de go-webrtcvad pour l'instant

const (
	SampleRate    = 16000
	FrameDuration = 30 // ms (ou une autre durée pour vos chunks)
	Channels      = 1
	BitDepth      = 16
)

type VAD struct {
	// Plus besoin de vadInstance
}

// NewVAD ne fait plus grand chose, mais on garde la signature pour compatibilité
func NewVAD(aggressiveness int) (*VAD, error) {
	fmt.Println("WARN: VAD désactivé pour le test. Tous les chunks audio seront considérés comme vocaux.")
	return &VAD{}, nil
}

// Process retourne toujours true, ou vous pouvez la modifier pour qu'elle ne soit plus appelée
func (v *VAD) Process(pcm []int16) (bool, error) {
	return true, nil // Simule que la voix est toujours présente
}

func (v *VAD) Close() {
	// Rien à faire
}
