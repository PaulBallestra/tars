package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/sashabaranov/go-openai"
)

// Définition des outils (si vous voulez les utiliser via l'API function calling plus tard)
// Pour l'instant, on ne les utilise pas dans GetResponse pour garder le flux simple.
var tools = []openai.Tool{
	// {
	//  Type: openai.ToolTypeFunction,
	//  Function: &openai.FunctionDefinition{
	//      Name:        "getCurrentWeather",
	//      Description: "Get the current weather in a given location",
	//      Parameters: map[string]interface{}{
	//          "type": "object",
	//          "properties": map[string]interface{}{
	//              "location": map[string]interface{}{
	//                  "type":        "string",
	//                  "description": "The city and state, e.g. San Francisco, CA",
	//              },
	//              "unit": map[string]interface{}{
	//                  "type": "string",
	//                  "enum": []string{"celsius", "fahrenheit"},
	//              },
	//          },
	//          "required": []string{"location"},
	//      },
	//  },
	// },
}

type LLMResponse struct {
	Content   string
	ToolCalls []openai.ToolCall
	Error     error
}

type LLMProcessor struct {
	client     *openai.Client
	outputChan chan LLMResponse
}

func NewLLMProcessor(client *openai.Client, outputChan chan LLMResponse) *LLMProcessor {
	return &LLMProcessor{
		client:     client,
		outputChan: outputChan,
	}
}

func (lp *LLMProcessor) GetResponse(ctx context.Context, messages []openai.ChatCompletionMessage, availableTools []openai.Tool) {
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo, // Ou gpt-4, gpt-4-turbo-preview etc.
		Messages: messages,
		// Stream:   true, // On ne gère pas le streaming pour le moment pour l'Étape 0 simple
		// Tools: availableTools, // Activer si vous voulez des tool calls
	}

	log.Println("LLM: Envoi de la requête à OpenAI...")
	resp, err := lp.client.CreateChatCompletion(ctx, req)

	if err != nil {
		lp.outputChan <- LLMResponse{Error: fmt.Errorf("erreur ChatCompletion: %w", err)}
		return
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" && len(resp.Choices[0].Message.ToolCalls) == 0 {
		lp.outputChan <- LLMResponse{Error: errors.New("réponse LLM vide")}
		return
	}

	choice := resp.Choices[0]
	if len(choice.Message.ToolCalls) > 0 {
		log.Printf("LLM: Reçu des ToolCalls: %+v", choice.Message.ToolCalls)
		lp.outputChan <- LLMResponse{ToolCalls: choice.Message.ToolCalls}
	} else {
		log.Printf("LLM: Réponse reçue: %s", choice.Message.Content)
		lp.outputChan <- LLMResponse{Content: choice.Message.Content}
	}
}

// GetResponseStream (PLUS COMPLEXE, pour plus tard si besoin de latence encore plus faible pour la réponse)
func (lp *LLMProcessor) GetResponseStream(ctx context.Context, messages []openai.ChatCompletionMessage) {
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT3Dot5Turbo,
		Messages: messages,
		Stream:   true,
	}

	stream, err := lp.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		lp.outputChan <- LLMResponse{Error: fmt.Errorf("erreur ChatCompletionStream: %w", err)}
		return
	}
	defer stream.Close()

	var fullResponse string
	log.Println("LLM Stream: ")
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			lp.outputChan <- LLMResponse{Error: fmt.Errorf("erreur réception stream: %w", err)}
			return
		}
		content := response.Choices[0].Delta.Content
		fullResponse += content
		// Pour une interaction ultra-rapide, on pourrait envoyer des bouts de phrase au TTS ici.
		// fmt.Print(content) // Ou envoyer vers un channel pour le TTS
	}
	// fmt.Println()
	lp.outputChan <- LLMResponse{Content: fullResponse}
}
