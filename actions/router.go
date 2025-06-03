package actions

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"
)

type ActionRouter struct {
	// Pourrait avoir des clients vers d'autres services plus tard
}

func NewActionRouter() *ActionRouter {
	return &ActionRouter{}
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"` // Contenu JSON du résultat de l'outil
}

// ProcessToolCalls simule l'exécution d'outils et retourne leurs résultats.
func (ar *ActionRouter) ProcessToolCalls(toolCalls []openai.ToolCall) []ToolResult {
	var results []ToolResult

	if len(toolCalls) == 0 {
		return results
	}

	log.Printf("ActionRouter: Reçu %d tool calls à traiter.", len(toolCalls))

	for _, call := range toolCalls {
		if call.Type != openai.ToolTypeFunction {
			log.Printf("ActionRouter: Type d'outil non supporté: %s", call.Type)
			continue
		}

		log.Printf("ActionRouter: Simulation de l'exécution de la fonction '%s' avec les arguments: %s", call.Function.Name, call.Function.Arguments)

		// Simuler une réponse en fonction du nom de l'outil
		var responseData interface{}
		switch call.Function.Name {
		case "getCurrentWeather":
			// Simuler une réponse météo
			// Pour un vrai outil, on lirait call.Function.Arguments
			responseData = map[string]interface{}{
				"location":    "Paris",
				"temperature": "15",
				"unit":        "celsius",
				"description": "Partiellement nuageux",
			}
		case "createDiscordChannel":
			// Simuler la création d'un canal
			responseData = map[string]interface{}{
				"status":      "success",
				"channelName": "simulated-channel-" + call.ID, // Utiliser call.ID pour un peu de variabilité
				"message":     "Canal simulé créé avec succès.",
			}
		default:
			responseData = map[string]interface{}{
				"status":  "error",
				"message": fmt.Sprintf("Outil '%s' non reconnu ou simulation non implémentée.", call.Function.Name),
			}
		}

		// Le LLM attend des résultats sous forme de string JSON
		jsonResult, err := json.Marshal(responseData)
		if err != nil {
			log.Printf("ActionRouter: Erreur de marshalling du résultat JSON pour l'outil %s: %v", call.Function.Name, err)
			results = append(results, ToolResult{
				ToolCallID: call.ID,
				Content:    `{"error": "failed to serialize result"}`,
			})
			continue
		}

		log.Printf("ActionRouter: Résultat pour l'outil '%s' (ID: %s): %s", call.Function.Name, call.ID, string(jsonResult))
		results = append(results, ToolResult{
			ToolCallID: call.ID,
			Content:    string(jsonResult),
		})
	}
	return results
}
