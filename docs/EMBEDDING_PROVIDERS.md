# Embedding Providers Configuration

Mentis supports multiple embedding providers for semantic similarity search. Configure via environment variables:

## Providers

### Mock Provider (Default)
```env
EMBEDDING_PROVIDER=mock
```
- Uses deterministic hash-based embeddings
- Good for development/testing
- No external API calls required

### OpenAI
```env
EMBEDDING_PROVIDER=openai
OPENAI_API_KEY=sk-your-api-key
OPENAI_MODEL=text-embedding-3-small
```
**Supported Models:**
- `text-embedding-3-small` (1536 dimensions)
- `text-embedding-3-large` (3072 dimensions) 
- `text-embedding-ada-002` (1536 dimensions)

### Google Gemini
```env
EMBEDDING_PROVIDER=gemini
GEMINI_API_KEY=your-api-key
GEMINI_MODEL=text-embedding-004
```
**Supported Models:**
- `text-embedding-004` (768 dimensions)
- `embedding-001` (768 dimensions)

### OpenAI-Compatible APIs
```env
EMBEDDING_PROVIDER=openai_compatible
EMBEDDING_BASE_URL=http://localhost:11434/v1
EMBEDDING_API_KEY=optional-key
EMBEDDING_MODEL=nomic-embed-text
```

**Common Use Cases:**
- **Ollama**: `http://localhost:11434/v1` with models like `nomic-embed-text`
- **Azure OpenAI**: `https://your-resource.openai.azure.com/openai/deployments/your-deployment/` 
- **Local servers**: Any OpenAI-compatible embedding API

## Model Dimensions

The system automatically configures Qdrant collections based on the embedding provider:
- OpenAI: 1536 or 3072 dimensions
- Gemini: 768 dimensions  
- OpenAI-Compatible: Varies by model (384-1536)

## Fallback Behavior

If embedding generation fails, the system logs the error but continues operation with the mock provider as fallback.