# Mentis Agent SDK Integration Guide
**Semantic Caching for AI Agent Frameworks**

This guide demonstrates how to integrate Mentis with popular agent development frameworks and SDKs, transforming your AI agents with intelligent semantic caching capabilities.

---

## 🚀 Quick Start

Mentis provides a **REST API** that seamlessly integrates with any agent framework. The key benefits:

- **🔍 Semantic Similarity**: Cache hits on paraphrased queries (`"reset password"` matches `"password reset help"`)
- **🔄 Workflow Awareness**: Cache individual steps, not just final results
- **🤝 Multi-Agent Sharing**: Agents learn from each other's work
- **⚡ Performance**: 60-90% faster response times, 40-70% cost reduction

```bash
# Start Mentis server
docker-compose up -d
go run cmd/server/main.go

# Mentis is now available at http://localhost:8080
```

---

## 🦜 LangChain Integration

### Custom Cache Backend

Replace LangChain's default cache with Mentis for semantic similarity:

```python
import requests
import hashlib
from langchain.cache import BaseCache
from langchain.schema import Generation
from typing import Any, Dict, List, Optional

class MentisCache(BaseCache):
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
    
    def lookup(self, prompt: str, llm_string: str) -> Optional[List[Generation]]:
        """Semantic lookup for cached LLM responses"""
        response = requests.get(f"{self.base_url}/v1/lookup", params={
            "q": prompt,
            "min_score": 0.85,
            "top_k": 1
        })
        
        if response.status_code == 200:
            results = response.json().get("results", [])
            if results:
                cached_content = results[0]["artifact"]["content"]
                return [Generation(text=cached_content)]
        return None
    
    def update(self, prompt: str, llm_string: str, return_val: List[Generation]) -> None:
        """Cache new LLM responses with semantic indexing"""
        content = return_val[0].text if return_val else ""
        
        requests.post(f"{self.base_url}/v1/cache/publish", json={
            "objects": [{
                "type": "ANSWER",
                "content": content.encode(),
                "metadata": {
                    "prompt": prompt,
                    "llm_string": llm_string,
                    "response_length": len(content)
                }
            }]
        })

# Usage with LangChain
from langchain.llms import OpenAI
from langchain import globals

# Set Mentis as the global cache
globals.set_llm_cache(MentisCache())

# Now all LLM calls will use semantic caching
llm = OpenAI()
result1 = llm("What is machine learning?")       # Cache miss
result2 = llm("Explain machine learning to me")  # Cache hit! (semantic similarity)
```

### Tool Integration

Use Mentis as a LangChain tool for explicit caching:

```python
from langchain.tools import BaseTool
from langchain.agents import initialize_agent, Tool
import json

class MentisCacheTool(BaseTool):
    name = "semantic_cache"
    description = "Cache and retrieve information using semantic similarity"
    
    def _run(self, action: str, query: str = "", content: str = "") -> str:
        if action == "lookup":
            response = requests.get(f"http://localhost:8080/v1/lookup", params={
                "q": query,
                "min_score": 0.8,
                "top_k": 3
            })
            if response.status_code == 200:
                results = response.json().get("results", [])
                return json.dumps([r["artifact"] for r in results])
            return "No cached results found"
        
        elif action == "store":
            requests.post(f"http://localhost:8080/v1/cache/publish", json={
                "objects": [{
                    "type": "DERIVED",
                    "content": content.encode(),
                    "metadata": {"query": query, "tool": "langchain"}
                }]
            })
            return "Content cached successfully"

# Use in an agent
tools = [MentisCacheTool()]
agent = initialize_agent(tools, llm, agent="zero-shot-react-description")

# Agent can now cache and retrieve information semantically
result = agent.run("Cache this analysis: The market shows bullish trends...")
cached = agent.run("Find information about market analysis")  # Retrieves cached content
```

### Chain Result Caching

Cache intermediate results in complex chains:

```python
from langchain.chains import LLMChain
from langchain.prompts import PromptTemplate

class CachedChain(LLMChain):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.cache_url = "http://localhost:8080"
    
    def _call(self, inputs: Dict[str, Any]) -> Dict[str, str]:
        # Check cache first
        cache_key = f"{self.prompt.template}:{str(inputs)}"
        
        response = requests.get(f"{self.cache_url}/v1/lookup", params={
            "q": cache_key,
            "min_score": 0.95,
            "top_k": 1
        })
        
        if response.status_code == 200:
            results = response.json().get("results", [])
            if results:
                return {"text": results[0]["artifact"]["content"]}
        
        # Cache miss - execute chain
        result = super()._call(inputs)
        
        # Cache the result
        requests.post(f"{self.cache_url}/v1/cache/publish", json={
            "objects": [{
                "type": "ANSWER",
                "content": result["text"].encode(),
                "metadata": {
                    "chain_type": self.__class__.__name__,
                    "inputs": inputs
                }
            }]
        })
        
        return result

# Usage
prompt = PromptTemplate(template="Analyze this data: {data}", input_variables=["data"])
cached_chain = CachedChain(llm=llm, prompt=prompt)

result1 = cached_chain.run("Q3 sales data shows 15% growth")  # Cache miss
result2 = cached_chain.run("Q3 sales increased by 15%")       # Cache hit!
```

---

## 🕸️ LangGraph Integration

### State Caching in Graphs

Cache expensive state transitions in LangGraph workflows:

```python
from langgraph.graph import StateGraph
from typing import TypedDict
import requests

class AgentState(TypedDict):
    messages: list
    current_step: str
    cached_results: dict

def cache_state_transition(state: AgentState, step_name: str, result: Any) -> AgentState:
    """Cache intermediate graph state"""
    
    # Store in Mentis
    requests.post("http://localhost:8080/v1/workflow/steps", json={
        "session_id": f"graph-{id(state)}",
        "step_type": step_name,
        "input": str(state.get("messages", [])),
        "metadata": {
            "graph_type": "agent_workflow",
            "step_name": step_name
        }
    })
    
    # Update state
    state["cached_results"][step_name] = result
    state["current_step"] = step_name
    return state

def lookup_cached_step(state: AgentState, step_name: str) -> Optional[Any]:
    """Check if step result is cached"""
    
    response = requests.post("http://localhost:8080/v1/workflow/steps/lookup", json={
        "session_id": f"graph-{id(state)}",
        "step_type": step_name,
        "input": str(state.get("messages", [])),
        "top_k": 1
    })
    
    if response.status_code == 200:
        results = response.json().get("results", [])
        if results and results[0]["score"] > 0.9:
            return results[0]["artifact"]["content"]
    return None

# Define graph nodes with caching
def research_node(state: AgentState) -> AgentState:
    # Check cache first
    cached = lookup_cached_step(state, "research")
    if cached:
        state["messages"].append(f"[CACHED] {cached}")
        return state
    
    # Perform research (expensive operation)
    research_result = perform_web_research(state["messages"][-1])
    
    # Cache the result
    return cache_state_transition(state, "research", research_result)

def analysis_node(state: AgentState) -> AgentState:
    cached = lookup_cached_step(state, "analysis")
    if cached:
        state["messages"].append(f"[CACHED] {cached}")
        return state
    
    # Perform analysis
    analysis_result = analyze_data(state["cached_results"].get("research"))
    
    return cache_state_transition(state, "analysis", analysis_result)

# Build the graph
workflow = StateGraph(AgentState)
workflow.add_node("research", research_node)
workflow.add_node("analysis", analysis_node)
workflow.add_edge("research", "analysis")

app = workflow.compile()

# Usage - subsequent runs with similar inputs will hit cache
result1 = app.invoke({"messages": ["Research AI trends"], "current_step": "", "cached_results": {}})
result2 = app.invoke({"messages": ["Study AI developments"], "current_step": "", "cached_results": {}})  # Cache hit!
```

### Cross-Graph Knowledge Sharing

Share cached results between different graph instances:

```python
class SharedGraphCache:
    def __init__(self):
        self.base_url = "http://localhost:8080"
    
    def get_shared_knowledge(self, topic: str, graph_type: str) -> List[Dict]:
        """Get cached knowledge from other graph instances"""
        response = requests.get(f"{self.base_url}/v1/lookup", params={
            "q": topic,
            "min_score": 0.8,
            "top_k": 5
        }, headers={"X-Graph-Type": graph_type})
        
        if response.status_code == 200:
            return response.json().get("results", [])
        return []
    
    def share_knowledge(self, topic: str, result: str, graph_type: str):
        """Share knowledge with other graph instances"""
        requests.post(f"{self.base_url}/v1/cache/publish", json={
            "objects": [{
                "type": "DERIVED",
                "content": result.encode(),
                "metadata": {
                    "topic": topic,
                    "graph_type": graph_type,
                    "shared": True
                }
            }]
        })

# Usage in multiple graph instances
shared_cache = SharedGraphCache()

# Graph A learns about market analysis
def market_analysis_node(state: AgentState) -> AgentState:
    # Check shared knowledge first
    shared_results = shared_cache.get_shared_knowledge("market analysis", "research_graph")
    
    if shared_results:
        # Leverage knowledge from other graphs
        state["messages"].append(f"Building on shared knowledge: {shared_results[0]['artifact']['content']}")
    
    # Perform new analysis
    new_analysis = perform_market_analysis()
    
    # Share back to the collective
    shared_cache.share_knowledge("market analysis", new_analysis, "research_graph")
    
    return state
```

---

## 🤖 Google Agent Development Kit (ADK)

### Agent Action Caching

Cache expensive agent actions in Google's ADK:

```python
import google.generativeai as genai
from google.ai.generativelanguage import FunctionCall
import requests
import json

class CachedADKAgent:
    def __init__(self, model_name: str = "gemini-pro"):
        self.model = genai.GenerativeModel(model_name)
        self.cache_url = "http://localhost:8080"
        self.session_id = f"adk-agent-{uuid.uuid4()}"
    
    def cached_function_call(self, function_call: FunctionCall) -> str:
        """Cache expensive function calls"""
        
        # Create cache key from function call
        cache_key = f"{function_call.name}:{json.dumps(function_call.args)}"
        
        # Check cache
        response = requests.get(f"{self.cache_url}/v1/lookup", params={
            "q": cache_key,
            "min_score": 0.95,
            "top_k": 1
        })
        
        if response.status_code == 200:
            results = response.json().get("results", [])
            if results:
                return results[0]["artifact"]["content"]
        
        # Cache miss - execute function
        result = self.execute_function(function_call)
        
        # Cache the result
        requests.post(f"{self.cache_url}/v1/cache/publish", json={
            "objects": [{
                "type": "ANSWER",
                "content": result.encode(),
                "metadata": {
                    "function_name": function_call.name,
                    "args": dict(function_call.args),
                    "agent_type": "adk"
                }
            }]
        })
        
        return result
    
    def execute_function(self, function_call: FunctionCall) -> str:
        """Execute the actual function (expensive operation)"""
        if function_call.name == "web_search":
            return perform_web_search(function_call.args["query"])
        elif function_call.name == "data_analysis":
            return analyze_dataset(function_call.args["data"])
        # Add more functions...
    
    def run_agent_loop(self, user_input: str) -> str:
        """Main agent execution loop with caching"""
        
        # Start workflow session
        requests.post(f"{self.cache_url}/v1/workflow/sessions", json={
            "goal": f"Process user request: {user_input}",
            "context": {"agent_type": "adk", "user_input": user_input}
        })
        
        # Process with model
        response = self.model.generate_content(user_input)
        
        # Handle function calls
        if response.candidates[0].content.parts:
            for part in response.candidates[0].content.parts:
                if hasattr(part, 'function_call'):
                    # Use cached function call
                    result = self.cached_function_call(part.function_call)
                    
                    # Continue conversation with result
                    follow_up = self.model.generate_content(
                        f"Function {part.function_call.name} returned: {result}. Please provide final response."
                    )
                    return follow_up.text
        
        return response.text

# Usage
agent = CachedADKAgent()

result1 = agent.run_agent_loop("Research machine learning trends")  # Cache miss
result2 = agent.run_agent_loop("Find info on ML developments")      # Cache hit!
```

### Multi-Agent Coordination with ADK

Coordinate multiple ADK agents using shared Mentis cache:

```python
class ADKAgentCluster:
    def __init__(self, agent_configs: List[Dict]):
        self.agents = {}
        self.cache_url = "http://localhost:8080"
        
        for config in agent_configs:
            self.agents[config["name"]] = CachedADKAgent(config["model"])
    
    def coordinate_task(self, task: str, required_agents: List[str]) -> Dict[str, str]:
        """Coordinate task across multiple agents with shared caching"""
        
        results = {}
        
        for agent_name in required_agents:
            if agent_name not in self.agents:
                continue
            
            # Check if another agent already processed similar task
            shared_response = requests.get(f"{self.cache_url}/v1/lookup", params={
                "q": task,
                "min_score": 0.85,
                "top_k": 1
            }, headers={"X-Agent-Type": "adk"})
            
            if shared_response.status_code == 200:
                shared_results = shared_response.json().get("results", [])
                if shared_results:
                    results[agent_name] = f"[SHARED] {shared_results[0]['artifact']['content']}"
                    continue
            
            # No shared result - agent processes task
            result = self.agents[agent_name].run_agent_loop(task)
            results[agent_name] = result
            
            # Share result for other agents
            requests.post(f"{self.cache_url}/v1/cache/publish", json={
                "objects": [{
                    "type": "ANSWER",
                    "content": result.encode(),
                    "metadata": {
                        "task": task,
                        "agent_name": agent_name,
                        "agent_type": "adk",
                        "shared": True
                    }
                }]
            })
        
        return results

# Usage
cluster = ADKAgentCluster([
    {"name": "researcher", "model": "gemini-pro"},
    {"name": "analyst", "model": "gemini-pro"},
    {"name": "writer", "model": "gemini-pro"}
])

# First agent processes task, others benefit from cache
results = cluster.coordinate_task(
    "Analyze the impact of AI on healthcare", 
    ["researcher", "analyst", "writer"]
)
```

---

## 🏗️ Generic Agent Architecture Patterns

### Workflow-Aware Agent Framework

Based on patterns from USER_SCENARIOS.md, here's a generic framework:

```python
from abc import ABC, abstractmethod
from typing import Dict, List, Any, Optional
import requests
import uuid

class CachedAgentStep(ABC):
    def __init__(self, step_name: str, cache_url: str = "http://localhost:8080"):
        self.step_name = step_name
        self.cache_url = cache_url
    
    @abstractmethod
    def execute(self, input_data: Any) -> Any:
        """Implement the actual step logic"""
        pass
    
    def run(self, input_data: Any, session_id: str) -> Any:
        """Execute step with caching"""
        
        # Check cache first
        response = requests.post(f"{self.cache_url}/v1/workflow/steps/lookup", json={
            "session_id": session_id,
            "step_type": self.step_name,
            "input": str(input_data),
            "top_k": 1
        })
        
        if response.status_code == 200:
            results = response.json().get("results", [])
            if results and results[0]["score"] > 0.9:
                return results[0]["artifact"]["content"]
        
        # Cache miss - execute step
        result = self.execute(input_data)
        
        # Cache result
        requests.post(f"{self.cache_url}/v1/workflow/steps", json={
            "session_id": session_id,
            "step_type": self.step_name,
            "input": str(input_data),
            "metadata": {
                "step_class": self.__class__.__name__,
                "execution_time": time.time()
            }
        })
        
        return result

class WebScrapingStep(CachedAgentStep):
    def execute(self, url: str) -> str:
        # Expensive web scraping operation
        return scrape_website(url)

class DataAnalysisStep(CachedAgentStep):
    def execute(self, data: str) -> Dict:
        # Expensive data analysis
        return analyze_data(data)

class SummaryStep(CachedAgentStep):
    def execute(self, analysis: Dict) -> str:
        # Generate summary using LLM
        return generate_summary(analysis)

class CachedWorkflowAgent:
    def __init__(self, steps: List[CachedAgentStep]):
        self.steps = steps
        self.session_id = str(uuid.uuid4())
    
    def run_workflow(self, initial_input: Any) -> Any:
        """Execute workflow with step-level caching"""
        
        current_input = initial_input
        
        for step in self.steps:
            current_input = step.run(current_input, self.session_id)
        
        return current_input

# Usage - Research Agent Workflow
research_agent = CachedWorkflowAgent([
    WebScrapingStep("scrape"),
    DataAnalysisStep("analyze"),
    SummaryStep("summarize")
])

result = research_agent.run_workflow("https://example.com/research-paper")
```

### Multi-Agent System with Shared Intelligence

```python
class AgentCluster:
    def __init__(self, cache_url: str = "http://localhost:8080"):
        self.cache_url = cache_url
        self.agents = {}
    
    def register_agent(self, agent_id: str, agent: CachedWorkflowAgent):
        """Register agent in cluster"""
        self.agents[agent_id] = agent
    
    def broadcast_knowledge(self, knowledge_type: str, content: str, source_agent: str):
        """Share knowledge across all agents in cluster"""
        requests.post(f"{self.cache_url}/v1/cache/publish", json={
            "objects": [{
                "type": "DERIVED",
                "content": content.encode(),
                "metadata": {
                    "knowledge_type": knowledge_type,
                    "source_agent": source_agent,
                    "cluster_shared": True,
                    "timestamp": time.time()
                }
            }]
        })
    
    def get_cluster_knowledge(self, query: str) -> List[Dict]:
        """Get relevant knowledge from cluster"""
        response = requests.get(f"{self.cache_url}/v1/lookup", params={
            "q": query,
            "min_score": 0.8,
            "top_k": 5
        }, headers={"X-Cluster-Shared": "true"})
        
        if response.status_code == 200:
            return response.json().get("results", [])
        return []
    
    def coordinate_task(self, task: str, agent_assignments: Dict[str, str]) -> Dict[str, Any]:
        """Coordinate task across multiple agents"""
        results = {}
        
        for agent_id, subtask in agent_assignments.items():
            if agent_id not in self.agents:
                continue
            
            # Check if any agent in cluster already solved similar task
            cluster_knowledge = self.get_cluster_knowledge(subtask)
            
            if cluster_knowledge:
                results[agent_id] = f"[CLUSTER] {cluster_knowledge[0]['artifact']['content']}"
                continue
            
            # Agent executes task
            result = self.agents[agent_id].run_workflow(subtask)
            results[agent_id] = result
            
            # Share knowledge with cluster
            self.broadcast_knowledge(f"task_result_{subtask[:20]}", str(result), agent_id)
        
        return results

# Usage - Research Cluster
cluster = AgentCluster()

# Different types of research agents
cluster.register_agent("market_researcher", CachedWorkflowAgent([...]))
cluster.register_agent("tech_researcher", CachedWorkflowAgent([...]))
cluster.register_agent("competitive_researcher", CachedWorkflowAgent([...]))

# Coordinate research across agents
results = cluster.coordinate_task("AI market analysis", {
    "market_researcher": "Analyze AI market size and growth",
    "tech_researcher": "Research latest AI technologies",
    "competitive_researcher": "Study key AI companies"
})
```

---

## ⚙️ Configuration & Best Practices

### Environment Setup

```bash
# .env file for agent applications
MENTIS_URL=http://localhost:8080
MENTIS_MIN_SCORE=0.85
MENTIS_MAX_CACHE_AGE=3600
MENTIS_ENABLE_WORKFLOW_TRACKING=true

# Vector provider configuration
VECTOR_PROVIDER=qdrant
QDRANT_HOST=localhost
QDRANT_PORT=6334

# Embedding provider
EMBEDDING_PROVIDER=openai
OPENAI_API_KEY=your-api-key
```

### Cache Key Strategies

```python
class CacheKeyStrategy:
    @staticmethod
    def generate_llm_key(prompt: str, model: str, params: Dict) -> str:
        """Generate cache key for LLM calls"""
        return f"llm:{model}:{hash(prompt)}:{hash(json.dumps(params, sort_keys=True))}"
    
    @staticmethod
    def generate_tool_key(tool_name: str, args: Dict) -> str:
        """Generate cache key for tool calls"""
        return f"tool:{tool_name}:{hash(json.dumps(args, sort_keys=True))}"
    
    @staticmethod
    def generate_workflow_key(workflow_name: str, input_data: Any) -> str:
        """Generate cache key for workflow steps"""
        return f"workflow:{workflow_name}:{hash(str(input_data))}"
```

### Error Handling & Fallbacks

```python
class ResilientCacheClient:
    def __init__(self, cache_url: str, timeout: int = 5):
        self.cache_url = cache_url
        self.timeout = timeout
    
    def safe_lookup(self, query: str, default=None) -> Any:
        """Safe cache lookup with fallback"""
        try:
            response = requests.get(
                f"{self.cache_url}/v1/lookup",
                params={"q": query, "min_score": 0.8},
                timeout=self.timeout
            )
            if response.status_code == 200:
                results = response.json().get("results", [])
                if results:
                    return results[0]["artifact"]["content"]
        except (requests.RequestException, KeyError, IndexError) as e:
            print(f"Cache lookup failed: {e}")
        
        return default
    
    def safe_store(self, content: str, metadata: Dict) -> bool:
        """Safe cache storage with error handling"""
        try:
            response = requests.post(
                f"{self.cache_url}/v1/cache/publish",
                json={
                    "objects": [{
                        "type": "DERIVED",
                        "content": content.encode(),
                        "metadata": metadata
                    }]
                },
                timeout=self.timeout
            )
            return response.status_code == 200
        except requests.RequestException as e:
            print(f"Cache storage failed: {e}")
            return False
```

### Performance Monitoring

```python
import time
from typing import ContextManager

class CachePerformanceMonitor:
    def __init__(self):
        self.stats = {
            "cache_hits": 0,
            "cache_misses": 0,
            "avg_lookup_time": 0,
            "total_lookups": 0
        }
    
    def track_lookup(self, query: str) -> ContextManager:
        """Context manager to track cache lookup performance"""
        
        class LookupTracker:
            def __init__(self, monitor):
                self.monitor = monitor
                self.start_time = None
            
            def __enter__(self):
                self.start_time = time.time()
                return self
            
            def __exit__(self, exc_type, exc_val, exc_tb):
                duration = time.time() - self.start_time
                self.monitor.stats["total_lookups"] += 1
                
                # Update average lookup time
                total_time = (self.monitor.stats["avg_lookup_time"] * 
                            (self.monitor.stats["total_lookups"] - 1) + duration)
                self.monitor.stats["avg_lookup_time"] = total_time / self.monitor.stats["total_lookups"]
        
        return LookupTracker(self)
    
    def record_hit(self):
        self.stats["cache_hits"] += 1
    
    def record_miss(self):
        self.stats["cache_misses"] += 1
    
    def get_hit_rate(self) -> float:
        total = self.stats["cache_hits"] + self.stats["cache_misses"]
        return self.stats["cache_hits"] / total if total > 0 else 0

# Usage
monitor = CachePerformanceMonitor()

with monitor.track_lookup("AI trends"):
    result = cache_client.safe_lookup("AI trends")
    if result:
        monitor.record_hit()
    else:
        monitor.record_miss()

print(f"Cache hit rate: {monitor.get_hit_rate():.2%}")
```

---

## 🔄 Migration Guide

### Retrofitting Existing Agents

#### Step 1: Add Cache Layer Gradually

```python
# Before: Direct LLM call
def old_agent_function(prompt: str) -> str:
    return llm.generate(prompt)

# After: Add caching layer
def cached_agent_function(prompt: str) -> str:
    # Check cache first
    cached_result = cache_client.safe_lookup(prompt)
    if cached_result:
        return cached_result
    
    # Fallback to original function
    result = llm.generate(prompt)
    
    # Cache for future use
    cache_client.safe_store(result, {"prompt": prompt})
    
    return result
```

#### Step 2: Implement Workflow Tracking

```python
# Wrap existing functions with workflow tracking
def add_workflow_tracking(func, step_name: str):
    def wrapper(*args, **kwargs):
        session_id = kwargs.get("session_id", "default")
        
        # Try workflow cache first
        cached = lookup_workflow_step(session_id, step_name, args[0])
        if cached:
            return cached
        
        # Execute original function
        result = func(*args, **kwargs)
        
        # Cache in workflow context
        cache_workflow_step(session_id, step_name, args[0], result)
        
        return result
    return wrapper

# Apply to existing functions
existing_function = add_workflow_tracking(existing_function, "data_processing")
```

#### Step 3: Enable Multi-Agent Sharing

```python
# Add cluster awareness to existing agents
class ClusterAwareAgent:
    def __init__(self, original_agent, cluster_id: str):
        self.original_agent = original_agent
        self.cluster_id = cluster_id
    
    def run(self, task: str) -> str:
        # Check cluster knowledge first
        cluster_results = get_cluster_knowledge(task, self.cluster_id)
        if cluster_results:
            return f"[CLUSTER] {cluster_results[0]['content']}"
        
        # Run original agent
        result = self.original_agent.run(task)
        
        # Share with cluster
        share_cluster_knowledge(task, result, self.cluster_id)
        
        return result
```

### Testing Strategy

```python
import unittest
from unittest.mock import patch, MagicMock

class TestCachedAgent(unittest.TestCase):
    def setUp(self):
        self.cache_client = ResilientCacheClient("http://localhost:8080")
        self.agent = CachedWorkflowAgent([...])
    
    @patch('requests.get')
    def test_cache_hit(self, mock_get):
        # Mock cache hit response
        mock_get.return_value = MagicMock(
            status_code=200,
            json=lambda: {
                "results": [{
                    "artifact": {"content": "cached result"},
                    "score": 0.95
                }]
            }
        )
        
        result = self.cache_client.safe_lookup("test query")
        self.assertEqual(result, "cached result")
    
    @patch('requests.get')
    def test_cache_miss_fallback(self, mock_get):
        # Mock cache miss
        mock_get.return_value = MagicMock(
            status_code=200,
            json=lambda: {"results": []}
        )
        
        result = self.cache_client.safe_lookup("test query", default="fallback")
        self.assertEqual(result, "fallback")
    
    def test_workflow_execution(self):
        # Test that workflow executes correctly with caching
        result = self.agent.run_workflow("test input")
        self.assertIsNotNone(result)

if __name__ == "__main__":
    unittest.main()
```

---

## 🎯 Performance Optimization Tips

### 1. **Batch Operations**
```python
# Instead of individual cache calls
for item in items:
    cache_client.lookup(item)

# Use batch lookup (if supported)
cache_client.batch_lookup(items)
```

### 2. **Async Operations**
```python
import asyncio
import aiohttp

async def async_cache_lookup(session, query):
    async with session.get(f"{CACHE_URL}/v1/lookup", params={"q": query}) as response:
        return await response.json()

# Use async for better performance
async def process_multiple_queries(queries):
    async with aiohttp.ClientSession() as session:
        tasks = [async_cache_lookup(session, query) for query in queries]
        return await asyncio.gather(*tasks)
```

### 3. **Smart Cache Invalidation**
```python
# Invalidate related cache entries when source data changes
def invalidate_related_cache(source_url: str):
    requests.post(f"{CACHE_URL}/v1/cache/invalidate", json={
        "source_url": source_url
    })

# Use webhook for real-time invalidation
@app.route("/webhook/data-update", methods=["POST"])
def handle_data_update():
    data = request.json
    invalidate_related_cache(data["source_url"])
    return {"status": "ok"}
```

---

## 📊 Monitoring & Analytics

### Dashboard Metrics

Track these key metrics in your monitoring dashboard:

```python
def collect_cache_metrics():
    """Collect metrics for monitoring dashboard"""
    return {
        "cache_hit_rate": monitor.get_hit_rate(),
        "avg_response_time": monitor.stats["avg_lookup_time"],
        "total_cache_size": get_cache_size(),
        "cost_savings": calculate_cost_savings(),
        "agent_efficiency": calculate_agent_efficiency()
    }
```

### Performance Alerts

```python
def setup_performance_alerts():
    """Setup alerts for performance degradation"""
    
    if monitor.get_hit_rate() < 0.7:
        send_alert("Cache hit rate below 70%")
    
    if monitor.stats["avg_lookup_time"] > 0.5:
        send_alert("Cache lookup time above 500ms")
```

---

## 🔮 Advanced Patterns

### Learning Agent Pattern

```python
class LearningAgent:
    """Agent that improves performance over time using cache insights"""
    
    def __init__(self):
        self.cache_client = ResilientCacheClient()
        self.learning_threshold = 0.8
    
    def adaptive_query(self, user_input: str) -> str:
        # Try multiple semantic variations
        variations = generate_query_variations(user_input)
        
        for variation in variations:
            result = self.cache_client.safe_lookup(variation)
            if result:
                # Learn from successful variation
                self.update_query_patterns(user_input, variation)
                return result
        
        # No cache hit - execute and learn
        result = self.execute_query(user_input)
        return result
```

### Federated Agent Network

```python
class FederatedAgentNetwork:
    """Network of agents sharing knowledge across different deployments"""
    
    def __init__(self, nodes: List[str]):
        self.nodes = nodes
        self.local_cache = ResilientCacheClient()
    
    def federated_lookup(self, query: str) -> Optional[str]:
        # Try local cache first
        result = self.local_cache.safe_lookup(query)
        if result:
            return result
        
        # Query federated nodes
        for node in self.nodes:
            try:
                response = requests.get(f"{node}/v1/lookup", params={"q": query})
                if response.status_code == 200:
                    results = response.json().get("results", [])
                    if results:
                        # Cache locally for future use
                        self.local_cache.safe_store(
                            results[0]["artifact"]["content"],
                            {"federated_source": node}
                        )
                        return results[0]["artifact"]["content"]
            except requests.RequestException:
                continue
        
        return None
```

---

**🚀 Ready to supercharge your agents with semantic caching?**

Start with a simple integration pattern and gradually add more sophisticated caching strategies. The examples in this guide provide a solid foundation for building intelligent, efficient AI agent systems.

For more detailed examples and advanced use cases, check out our [User Scenarios](USER_SCENARIOS.md) documentation.