# Mentis: Workflow-Aware Semantic Cache for AI Agents
**Product Requirements Document (PRD)**

**Author:** Anunay Aatipamula  
**Date:** 2025-07-08  
**Version:** 2.0

---

## 1. Executive Summary

AI agents today waste significant computational resources by repeatedly performing identical or semantically similar work across sessions. Traditional caching solutions like Redis provide exact-key matching but fail to capture the semantic relationships and multi-step workflow patterns inherent in agent systems.

**Mentis** introduces a **workflow-aware semantic cache** that understands both the content and the process of AI agent operations. By caching at the workflow step level and using semantic similarity for retrieval, Mentis transforms how agents reuse computational work.

### Key Innovation
Unlike traditional caches that store only final results, Mentis caches **intermediate workflow steps** (scrape → embed → reason → answer) and enables **semantic reuse** across different but related queries.

---

## 2. Problem Statement

### Current State: The Agent Efficiency Crisis

AI agents operating in production environments exhibit several critical inefficiencies:

**P1: Semantic Cache Misses**
- Query: "What are the specs of RTX 4090?"
- Follow-up: "RTX 4090 specifications please"
- **Problem**: Traditional caches miss on paraphrased queries
- **Impact**: 100% cache miss rate for semantically identical requests

**P2: Workflow Step Redundancy**
- Agent A: Scrapes Apple M3 specs → Embeds content → Analyzes performance
- Agent B: Needs to compare M3 vs M4 → Re-scrapes M3, re-embeds, re-analyzes
- **Problem**: No reuse of expensive intermediate computations
- **Impact**: 2-10x computational overhead for related workflows

**P3: Cost-Blind Eviction**
- Expensive 50K-token GPT-4 reasoning chain evicted before simple web scrape
- **Problem**: Traditional LRU/LFU ignore computational cost
- **Impact**: Evicting high-value cached work first

**P4: Provenance Staleness**
- Source content changes but cached results remain stale
- **Problem**: No invalidation strategy based on source freshness
- **Impact**: Agents serving outdated information

---

## 3. Game-Changing Scenarios

### Scenario 1: Research Assistant Revolution
**Before Mentis:**
```
User 1: "Analyze Tesla's Q3 earnings"
Agent: Scrape (2s) → Clean (1s) → Embed (3s) → Analyze (15s) = 21s

User 2: "What were Tesla's Q3 results?"  
Agent: Scrape (2s) → Clean (1s) → Embed (3s) → Analyze (15s) = 21s
```

**With Mentis:**
```
User 1: "Analyze Tesla's Q3 earnings"
Agent: Scrape (2s) → Clean (1s) → Embed (3s) → Analyze (15s) = 21s
[Mentis caches: RAW(scrape), DERIVED(clean+embed), REASONING(analysis)]

User 2: "What were Tesla's Q3 results?"
Agent: Semantic lookup (0.1s) → Cache hit on REASONING = 0.1s
```
**Result: 210x speed improvement, 99.5% cost reduction**

### Scenario 2: Multi-Agent Collaboration
**Before Mentis:**
```
Agent A: Research "sustainable energy trends 2024"
  - Scrapes 15 sources (30s)
  - Embeds all content (20s) 
  - Extracts key insights (45s)

Agent B: Research "renewable energy market 2024"
  - Scrapes 12 sources (24s) [8 overlap with Agent A]
  - Embeds all content (16s)
  - Extracts insights (35s)
```

**With Mentis:**
```
Agent A: Research "sustainable energy trends 2024"
  - Scrapes 15 sources (30s)
  - Embeds all content (20s)
  - Extracts insights (45s)
[Caches workflow steps with semantic tags]

Agent B: Research "renewable energy market 2024"  
  - Semantic lookup finds 8 cached sources (0.5s)
  - Scrapes 4 new sources (8s)
  - Embeds new content only (5s)
  - Combines with cached insights (10s)
```
**Result: Agent B completes in 23.5s instead of 75s (3.2x faster)**

### Scenario 3: Dynamic Source Invalidation
**Traditional Cache:**
```
12:00 PM: Agent caches Apple M3 performance data
2:00 PM: Apple updates M3 benchmarks on their website
2:05 PM: User asks about M3 performance
Result: Agent serves stale 2-hour-old data
```

**With Mentis:**
```
12:00 PM: Agent caches Apple M3 data with source provenance
2:00 PM: Mentis detects ETag change on Apple's website
2:01 PM: Marks all M3-related artifacts as STALE
2:05 PM: User asks about M3 performance  
Result: Agent automatically re-scrapes fresh data
```

---

## 4. Target Personas

### Primary: AI Agent Developer "Alex"
- **Role**: Senior Engineer at AI startup
- **Builds**: Customer support agents, research assistants
- **Pain Points**: 
  - Agent response times too slow for production
  - High OpenAI API costs from repeated similar queries
  - Complex caching logic reduces development velocity
- **Success Metrics**: <2s response time, 50% cost reduction

### Secondary: Platform Engineer "Taylor"  
- **Role**: Infrastructure Lead at enterprise AI platform
- **Manages**: 1000+ concurrent agents across multiple tenants
- **Pain Points**:
  - Unpredictable scaling costs
  - Cache warming strategies are manual
  - No visibility into agent computational efficiency
- **Success Metrics**: Predictable costs, <300ms p99 latency

### Tertiary: AI Researcher "Sam"
- **Role**: Research Scientist experimenting with agent architectures
- **Builds**: Novel multi-agent systems, tool-using agents
- **Pain Points**:
  - Expensive experimentation cycles
  - Difficulty reproducing agent behaviors
  - Need for workflow step introspection
- **Success Metrics**: Faster iteration, reproducible experiments

---

## 5. Functional Requirements

| ID | Requirement | Priority | Acceptance Criteria |
|----|-------------|----------|---------------------|
| **F-01** | Semantic artifact lookup with configurable similarity threshold | Must | Return artifacts with cosine similarity ≥ 0.85 in <150ms p95 |
| **F-02** | Multi-granular workflow step caching (RAW→DERIVED→REASONING→ANSWER) | Must | Support 4-tier artifact hierarchy with dependency tracking |
| **F-03** | Multi-provider embedding support (OpenAI, Gemini, local models) | Must | Configurable via environment, fallback to mock |
| **F-04** | Content deduplication via cryptographic hashing | Must | Prevent storage of duplicate content across all artifact types |
| **F-05** | Workflow session management and step tracking | Must | Track multi-step agent workflows with session context |
| **F-06** | RESTful API for cache operations | Must | Publish, lookup, invalidate operations via HTTP |
| **F-07** | Provenance-based invalidation | Should | Detect source changes via ETag/Last-Modified headers |
| **F-08** | Cost-aware eviction policies | Should | Evict based on (size × computational_cost) / reuse_frequency |
| **F-09** | Real-time metrics and observability | Should | Prometheus metrics, structured logging |
| **F-10** | Multi-tenant isolation | Could | Namespace-based data separation |

---

## 6. Non-Functional Requirements

### Performance
- **Latency**: Semantic lookup p95 < 150ms, p99 < 300ms
- **Throughput**: 1K RPS sustained, 5K RPS burst
- **Embedding Generation**: Support for batch operations

### Scalability  
- **Artifacts**: 10M+ artifacts per deployment
- **Vectors**: 1B+ vector points in Qdrant
- **Concurrent Workflows**: 10K+ active sessions

### Reliability
- **Availability**: 99.9% monthly uptime
- **Consistency**: Read-after-write within 1 second
- **Durability**: Zero data loss with PostgreSQL persistence

### Security
- **Encryption**: AES-256 at rest, TLS 1.3 in transit
- **Authentication**: API key based access control
- **Privacy**: PII detection and scrubbing capabilities

---

## 7. Success Metrics & KPIs

### Primary Business Metrics
- **Cost Reduction**: 40% reduction in LLM API costs
- **Latency Improvement**: 60% faster agent response times
- **Hit Rate**: 70% cache hit rate for production workloads

### Technical Metrics  
- **Semantic Hit Rate**: 55% for paraphrased queries
- **Workflow Reuse**: 60% reuse of intermediate artifacts
- **Invalidation Accuracy**: <10 minute lag for source changes

### Developer Experience
- **Integration Time**: <1 hour to integrate via REST API
- **API Reliability**: 99.95% successful request rate
- **Documentation Quality**: <5 support tickets per 100 integrations

---

## 8. Competitive Analysis

| Solution | Strengths | Weaknesses | Differentiation |
|----------|-----------|------------|-----------------|
| **Redis** | Fast, battle-tested | Exact-key only, no semantics | Mentis adds semantic similarity |
| **Pinecone** | Vector search | No workflow context | Mentis understands agent steps |
| **LangChain Cache** | Framework integration | Limited scalability | Mentis is deployment-independent |
| **Custom Solutions** | Domain-specific | High maintenance | Mentis is general-purpose |

**Key Differentiator**: Mentis is the only solution that combines semantic similarity with workflow-step awareness for AI agents.

---

## 9. Technical Approach Preview

### Architecture Principles
1. **Separation of Concerns**: Vector storage (Qdrant) + Metadata (PostgreSQL)
2. **Provider Agnostic**: Support multiple embedding providers
3. **Workflow Aware**: Cache at step granularity, not just final results
4. **Semantic First**: Default to semantic similarity over exact matching

### Core Components
- **Embedding Service**: Multi-provider abstraction (OpenAI/Gemini/Local)
- **Vector Store**: Qdrant for semantic similarity search
- **Metadata Store**: PostgreSQL for relationships and provenance
- **Workflow Engine**: Session and step tracking with dependency graphs

---

## 10. Success Stories (Projected)

### Customer Support Agent Platform
- **Before**: 45-second average response time, $50K/month in API costs
- **After**: 8-second response time, $20K/month costs
- **ROI**: 400% improvement in customer satisfaction, 60% cost savings

### Research Intelligence Platform  
- **Before**: Researchers wait 5+ minutes for multi-source analysis
- **After**: Same analysis in 30 seconds using cached intermediate steps
- **ROI**: 10x researcher productivity, enables real-time research workflows

### Multi-Agent Trading System
- **Before**: Each agent independently analyzes market data
- **After**: Agents share semantic analysis via Mentis cache
- **ROI**: 90% reduction in duplicate API calls, 5x faster decision cycles

---

## 11. Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Vector similarity false positives** | Wrong cache hits | Medium | Configurable similarity thresholds, hybrid scoring |
| **Embedding provider outages** | Service degradation | Low | Multi-provider fallback, mock provider backup |
| **Storage cost explosion** | Operational costs | Medium | Intelligent eviction, compression strategies |
| **Complex workflow dependencies** | Cache consistency | High | Immutable artifacts, versioned dependencies |

---

## 12. Future Roadmap

### Phase 1 (Implemented): Core Semantic Cache
- ✅ Multi-provider embeddings
- ✅ Workflow step tracking  
- ✅ Basic semantic lookup
- ✅ REST API

### Phase 2: Intelligence Layer
- 🔄 Cost-aware eviction
- 🔄 Provenance tracking
- 🔄 Real-time invalidation
- 🔄 Performance analytics

### Phase 3: Enterprise Features
- 📋 Multi-tenant isolation
- 📋 Advanced observability
- 📋 SDK libraries (Python, TypeScript)
- 📋 Compliance tools

### Phase 4: AI-Native Features
- 📋 Automatic workflow optimization
- 📋 Predictive pre-caching
- 📋 Cross-agent learning
- 📋 Federated cache networks

---

**Mentis represents the next evolution of caching for the AI agent era - moving from simple key-value storage to intelligent, workflow-aware semantic systems that understand how agents think and work.**