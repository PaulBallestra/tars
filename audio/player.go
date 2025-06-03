package audio

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time" // Nécessaire pour les timeouts potentiels

	"github.com/ebitengine/oto/v3"
	// Pour TTSSampleRate, TTSChannels
)

// audioChanReader adapte notre channel de PCM en io.Reader pour oto/v3
type audioChanReader struct {
	pcmChan chan []byte
	mu      sync.Mutex
	buffer  []byte // Buffer interne pour les données non lues
	closed  bool   // Indique si le channel pcmChan a été fermé
	reading bool   // Pour éviter les lectures concurrentes sur le player
}

func newAudioChanReader(pcmChan chan []byte) *audioChanReader {
	return &audioChanReader{
		pcmChan: pcmChan,
	}
}

// Read implémente io.Reader
func (acr *audioChanReader) Read(p []byte) (n int, err error) {
	acr.mu.Lock()
	defer acr.mu.Unlock()

	// Si on a des données en buffer, on les sert d'abord
	if len(acr.buffer) > 0 {
		n = copy(p, acr.buffer)
		acr.buffer = acr.buffer[n:]
		if len(acr.buffer) == 0 && acr.closed { // Si buffer vide et channel fermé, c'est la fin
			return n, io.EOF
		}
		return n, nil
	}

	// Si le channel a été signalé comme fermé et le buffer est vide, c'est EOF
	if acr.closed {
		return 0, io.EOF
	}

	// Lire depuis le channel
	// On utilise un select pour ne pas bloquer indéfiniment si le chan est vide
	// et permettre une interruption propre plus tard si nécessaire.
	select {
	case data, ok := <-acr.pcmChan:
		if !ok { // Channel fermé
			acr.closed = true
			// log.Println("TARS AudioPlayer (chanReader): pcmChan fermé.")
			return 0, io.EOF
		}
		if len(data) == 0 && acr.closed { // Message vide alors qu'on est en train de fermer
			return 0, io.EOF
		}
		if len(data) == 0 { // Juste un message vide, on attend le suivant
			return 0, nil // Pas d'erreur, juste 0 bytes lus
		}

		// log.Printf("TARS AudioPlayer (chanReader): Reçu %d bytes du channel", len(data))
		n = copy(p, data)
		if n < len(data) { // S'il reste des données non copiées, on les bufferise
			acr.buffer = append(acr.buffer, data[n:]...)
		}
		return n, nil
	case <-time.After(100 * time.Millisecond): // Timeout pour ne pas bloquer indéfiniment
		// log.Println("TARS AudioPlayer (chanReader): Timeout lecture channel")
		return 0, nil // Pas d'erreur, mais 0 bytes lus, Read sera rappelé
	}
}

// Close signale que plus aucune donnée ne viendra sur pcmChan.
// Ceci est important pour que Read retourne io.EOF après avoir vidé le buffer.
// Note: ce n'est pas io.Closer, c'est une méthode custom pour notre reader.
func (acr *audioChanReader) SignalClose() {
	acr.mu.Lock()
	defer acr.mu.Unlock()
	acr.closed = true
	// log.Println("TARS AudioPlayer (chanReader): SignalClose appelé.")
}

type AudioPlayer struct {
	otoCtx       *oto.Context
	player       oto.Player // L'interface Player de oto/v3
	chanReader   *audioChanReader
	audioPCMChan chan []byte // Le channel d'origine pour recevoir le PCM
	playingLock  sync.Mutex
	isPlaying    bool
	playerWg     sync.WaitGroup // Pour attendre que le player.Play() se termine proprement
}

// NewAudioPlayer initialise le contexte oto et le player.
// sampleRate, channels, bitDepth sont pour l'audio à jouer (ex: sortie TTS)
func NewAudioPlayer(
	audioPCMInChan chan []byte,
	sampleRate int,
	channels int,
	// bitDepth est souvent en bits (ex: 16), oto/v3 utilise un Format.
	// Pour PCM 16-bit Little Endian (standard):
) (*AudioPlayer, error) {
	op := &oto.NewContextOptions{}
	op.SampleRate = sampleRate
	op.ChannelCount = channels
	// Le format le plus courant pour le PCM 16-bit est SignedInt16LE
	op.Format = oto.FormatSignedInt16LE // Correspond à PCM 16-bit little-endian

	otoCtx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return nil, fmt.Errorf("TARS AudioPlayer: Erreur création contexte oto: %w", err)
	}
	log.Println("TARS AudioPlayer: Contexte Oto créé. En attente de disponibilité...")
	<-readyChan // Attendre que le système audio soit prêt
	log.Println("TARS AudioPlayer: Système audio Oto prêt.")

	chanReader := newAudioChanReader(audioPCMInChan)

	// Le player prend un io.Reader. Notre chanReader l'implémente.
	// Le player créé ici est prêt à être joué, mais ne démarre pas automatiquement.
	playerInstance := otoCtx.NewPlayer(chanReader)

	return &AudioPlayer{
		otoCtx:       otoCtx,
		player:       *playerInstance,
		chanReader:   chanReader,
		audioPCMChan: audioPCMInChan, // On le garde pour le fermer proprement
	}, nil
}

// Start met en route la lecture continue depuis le channel sur le player.
// Cette méthode doit être appelée dans une goroutine si elle ne doit pas bloquer.
// Ou elle peut être bloquante et main.go gère la goroutine.
// Pour ce POC, on va la rendre bloquante et main.go lancera dans une goroutine dédiée.
func (ap *AudioPlayer) StartPlaybackLoop() {
	ap.playingLock.Lock()
	if ap.isPlaying {
		ap.playingLock.Unlock()
		log.Println("TARS AudioPlayer: Playback loop déjà démarrée.")
		return
	}
	ap.isPlaying = true
	ap.playingLock.Unlock()

	log.Println("TARS AudioPlayer: Démarrage de la boucle de lecture du player.")
	ap.playerWg.Add(1) // Incrémenter avant de démarrer la goroutine de player.Play()

	// player.Play() est bloquant jusqu'à ce que le reader retourne io.EOF ou une erreur.
	// Il doit donc tourner dans sa propre goroutine si l'on veut pouvoir faire autre chose.
	// Cependant, le player.Play() s'arrête dès que Read() retourne io.EOF une fois.
	// Si on veut une lecture continue depuis le channel sans recréer le player,
	// notre chanReader ne doit JAMAIS retourner EOF tant que l'application tourne,
	// sauf si le AudioPlayer est explicitement arrêté/fermé.

	// Le comportement attendu est que player.Play() tourne en boucle et lit de chanReader
	// tant que chanReader a des données. Quand chanReader bloque (attente sur channel),
	// player.Play() bloque aussi.
	ap.player.Play() // Cette fonction est bloquante et ne retourne pas d'erreur.
	// Elle se termine quand le reader (ap.chanReader) renvoie io.EOF.
	log.Println("TARS AudioPlayer: player.Play() terminé (normalement dû à EOF du reader).")

	ap.playingLock.Lock()
	ap.isPlaying = false
	ap.playingLock.Unlock()
	ap.playerWg.Done() // Décrémenter quand player.Play() est terminé
	log.Println("TARS AudioPlayer: Boucle de lecture du player terminée.")
}

// SendData est une méthode pratique pour envoyer des données au channel interne,
// mais la struct est initialisée avec le channel, donc c'est le TTSProcessor
// qui enverra directement sur ce channel. Cette méthode n'est donc plus nécessaire.
/*
func (ap *AudioPlayer) SendData(pcmData []byte) {
    if len(pcmData) == 0 {
        return
    }
    // log.Printf("TARS AudioPlayer: Envoi de %d bytes au channel du player.", len(pcmData))
    ap.audioPCMChan <- pcmData
}
*/

// Interrupt arrête la sortie audio actuelle en vidant le buffer du reader
// et en signalant au reader de se préparer à se clore si le channel est vide.
// Dans oto/v3, interrompre directement player.Play() est plus complexe si on veut le reprendre.
// Pour l'instant, la "vraie" interruption sera gérée par le flux de données :
// Si le VAD détecte une nouvelle parole utilisateur, le LLM sera (re)sollicité,
// et une nouvelle réponse TTS viendra remplacer / suivre l'ancienne.
// Une interruption plus "brutale" pourrait consister à fermer et recréer le player.
func (ap *AudioPlayer) Interrupt() {
	ap.playingLock.Lock()
	defer ap.playingLock.Unlock()

	log.Println("TARS AudioPlayer: Demande d'interruption.")
	// Vider le buffer interne du chanReader
	ap.chanReader.mu.Lock()
	ap.chanReader.buffer = nil
	// On ne ferme pas pcmChan ici, car il est partagé et peut être réutilisé.
	// On ne signale pas non plus chanReader.SignalClose(), sinon il ne lira plus jamais.
	// L'interruption doit être plus "logique" : ne plus envoyer sur pcmChan, ou envoyer du silence.
	ap.chanReader.mu.Unlock()

	// Pour une interruption plus agressive, on pourrait utiliser player.Pause() s'il était disponible
	// ou fermer le player actuel et en recréer un nouveau.
	// oto/v3 n'a pas de Pause/Resume simple sur le Player si le reader est continu.
	// La solution la plus simple est de simplement arrêter d'envoyer des données au reader.
	// La prochaine parole du bot remplacera l'ancienne si elle arrive sur le même channel.
	log.Println("TARS AudioPlayer: Buffer interne vidé. La lecture s'arrêtera si plus de données ne suivent.")
}

// Close libère les ressources oto.
func (ap *AudioPlayer) Close() {
	log.Println("TARS AudioPlayer: Fermeture...")

	// 1. Signaler au chanReader de se terminer (il retournera EOF après avoir vidé son buffer)
	//    et fermer le channel d'entrée pour débloquer toute goroutine en attente dessus.
	ap.chanReader.SignalClose() // Important pour que Read retourne EOF
	close(ap.audioPCMChan)      // Débloque le Read dans chanReader s'il attendait

	// 2. Attendre que la goroutine de player.Play() se termine.
	//    player.Play() devrait se terminer car chanReader retournera EOF.
	log.Println("TARS AudioPlayer: En attente de la fin de player.Play()...")
	ap.playerWg.Wait() // Attendre que la boucle de lecture (player.Play()) soit vraiment finie.

	// 3. Fermer le player (pas explicitement requis si le context est fermé, mais bonne pratique)
	if ap.player != (oto.Player{}) {
		err := ap.player.Close()
		if err != nil {
			log.Printf("TARS AudioPlayer: Erreur à la fermeture du player oto: %v", err)
		} else {
			log.Println("TARS AudioPlayer: Player oto fermé.")
		}
	}

	// 4. Fermer le contexte Oto
	if ap.otoCtx != nil {
		log.Println("TARS AudioPlayer: Le contexte Oto se fermera lorsque les players seront fermés et qu'il ne sera plus référencé.")
	}
	log.Println("TARS AudioPlayer: Fermeture terminée.")
}

// IsPlaying retourne l'état actuel du player.
func (ap *AudioPlayer) IsPlaying() bool {
	ap.playingLock.Lock()
	defer ap.playingLock.Unlock()
	// isPlaying est mis à jour par StartPlaybackLoop et Play().
	// On peut aussi vérifier si le player oto est actif:
	// return ap.isPlaying && ap.player != nil && ap.player.IsPlaying() // IsPlaying() sur le player oto v3 n'existe pas directement
	// On se fie à notre propre flag isPlaying, géré par StartPlaybackLoop
	return ap.isPlaying
}
