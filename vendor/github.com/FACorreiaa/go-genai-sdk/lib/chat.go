package genai_sdk

import (
	"context"
	"flag"
	"fmt"
	"iter"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genai"
)

var model = flag.String("model", "gemini-2.0-flash", "the model name, e.g. gemini-2.0-flash")

type LLMChatClient struct {
	client    *genai.Client
	model     string
	ModelName string
}

func NewLLMChatClient(ctx context.Context, apiKey string) (*LLMChatClient, error) {
	ctx, span := otel.Tracer("GenerativeAI").Start(ctx, "NewAIClient")
	defer span.End()

	if apiKey == "" {
		err := fmt.Errorf("GEMINI_API_KEY environment variable is not set")
		span.RecordError(err)
		span.SetStatus(codes.Error, "API key not set")
		log.Fatal(err)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create Gemini client")
		return nil, err
	}

	span.SetStatus(codes.Ok, "AI client created successfully")
	return &LLMChatClient{
		client:    client,
		model:     *model,
		ModelName: *model,
	}, nil
}

type ChatSession struct {
	chat *genai.Chat
}

func (ai *LLMChatClient) GenerateContent(ctx context.Context, prompt, apiKey string, config *genai.GenerateContentConfig) (string, error) {
	ctx, span := otel.Tracer("GenerativeAI").Start(ctx, "GenerateContent", trace.WithAttributes(
		attribute.String("prompt.length", fmt.Sprintf("%d", len(prompt))),
		attribute.String("model", *model),
	))
	defer span.End()

	newClient, err := NewLLMChatClient(ctx, apiKey)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create client")
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	result, err := newClient.client.Models.GenerateContent(ctx, *model, genai.Text(prompt), config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to generate content")
		log.Fatal(err)
	}

	responseText := result.Text()
	span.SetAttributes(attribute.Int("response.length", len(responseText)))
	span.SetStatus(codes.Ok, "Content generated successfully")
	return responseText, nil
}

func (ai *LLMChatClient) GenerateResponse(ctx context.Context, prompt string, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	ctx, span := otel.Tracer("GenerativeAI").Start(ctx, "GenerateResponse", trace.WithAttributes(
		attribute.String("prompt.length", fmt.Sprintf("%d", len(prompt))),
		attribute.String("model", ai.model),
	))
	defer span.End()

	chat, err := ai.client.Chats.Create(ctx, ai.model, config, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create chat")
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	response, err := chat.SendMessage(ctx, genai.Part{Text: prompt})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send message")
		return nil, err
	}

	span.SetStatus(codes.Ok, "Response generated successfully")
	return response, nil
}

func (ai *LLMChatClient) StartChatSession(ctx context.Context, config *genai.GenerateContentConfig) (*ChatSession, error) {
	ctx, span := otel.Tracer("GenerativeAI").Start(ctx, "StartChatSession", trace.WithAttributes(
		attribute.String("model", ai.model),
	))
	defer span.End()

	//config = &genai.GenerateContentConfig{Temperature: genai.Ptr[float32](0.5)}
	chat, err := ai.client.Chats.Create(ctx, ai.model, config, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create chat session")
		return nil, err
	}

	span.SetStatus(codes.Ok, "Chat session created successfully")
	return &ChatSession{chat: chat}, nil
}

func (cs *ChatSession) SendMessage(ctx context.Context, message string) (string, error) {
	ctx, span := otel.Tracer("GenerativeAI").Start(ctx, "SendMessage", trace.WithAttributes(
		attribute.String("message.length", fmt.Sprintf("%d", len(message))),
	))
	defer span.End()

	result, err := cs.chat.SendMessage(ctx, genai.Part{Text: message})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to send message")
		return "", err
	}

	responseText := result.Text()
	span.SetAttributes(attribute.Int("response.length", len(responseText)))
	span.SetStatus(codes.Ok, "Message sent successfully")
	return responseText, nil
}

// GenerateContentStream initiates a streaming content generation process.
func (ai *LLMChatClient) GenerateContentStream(
	ctx context.Context,
	prompt string,
	config *genai.GenerateContentConfig,
) (iter.Seq2[*genai.GenerateContentResponse, error], error) {
	return ai.GenerateContentStreamWithCache(ctx, prompt, config, "")
}

// GenerateContentStreamWithCache initiates a streaming content generation process with cache support
func (ai *LLMChatClient) GenerateContentStreamWithCache(
	ctx context.Context,
	prompt string,
	config *genai.GenerateContentConfig,
	cacheKey string,
) (iter.Seq2[*genai.GenerateContentResponse, error], error) {
	ctx, span := otel.Tracer("GenerativeAI").Start(ctx, "GenerateContentStreamWithCache", trace.WithAttributes(
		attribute.String("prompt.length", fmt.Sprintf("%d", len(prompt))),
		attribute.String("model", ai.model),
		attribute.String("cache.key", cacheKey),
		attribute.Bool("cache.enabled", cacheKey != ""),
	))
	defer span.End()

	if ai.client == nil {
		err := fmt.Errorf("AIClient's internal genai.Client is not initialized")
		span.RecordError(err)
		span.SetStatus(codes.Error, "Client not initialized for stream")
		return nil, err
	}

	var foundCacheName string
	// Try to get cached content first if cache key is provided
	if cacheKey != "" {
		// Check for existing cache
		page, err := ai.client.Caches.List(ctx, &genai.ListCachedContentsConfig{})
		if err == nil {
			for _, cache := range page.Items {
				if cache.DisplayName == cacheKey {
					foundCacheName = cache.Name
					span.SetAttributes(
						attribute.Bool("cache.hit", true),
						attribute.String("cache.name", cache.Name),
					)
					break
				}
			}
		}
		if foundCacheName == "" {
			span.SetAttributes(attribute.Bool("cache.miss", true))
		}
	}

	// Create configuration potentially with cache reference
	configToUse := config
	if configToUse == nil {
		configToUse = &genai.GenerateContentConfig{}
	}

	// Create a chat session
	// Note: For now we create a regular chat session since the exact API for
	// using cached content in streaming is still being clarified
	chat, err := ai.client.Chats.Create(ctx, ai.model, configToUse, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create chat for stream")
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	// Create the prompt part
	part := genai.Part{Text: prompt}

	// If we found a cache, note it in logs but still proceed with regular streaming
	// This allows the infrastructure to be in place while we perfect the cache usage
	if foundCacheName != "" {
		span.SetAttributes(attribute.String("cache.used", foundCacheName))
	}

	span.SetStatus(codes.Ok, "Content stream initiated")
	return chat.SendMessageStream(ctx, part), nil
}

// Add SendMessageStream to ChatSession
func (cs *ChatSession) SendMessageStream(ctx context.Context, message string) iter.Seq2[*genai.GenerateContentResponse, error] {
	return cs.chat.SendMessageStream(ctx, genai.Part{Text: message})
}
