# KanbanAI — Arquitetura

## 1. Visão Geral

**KanbanAI** é uma aplicação em Go que orquestra um fluxo Kanban automatizado por inteligência artificial. O sistema conecta-se a harnesses de IA (Claude, Pi, Hermes etc.) via **MCP (Model Context Protocol)** para executar cada fase do fluxo de forma autônoma.

O usuário cria uma task pelo dashboard; o sistema persiste no banco SQLite, dispara eventos via **Observer Pattern**, notifica o frontend em tempo real via **SSE (Server-Sent Events)** e coordena a execução de cada raia do Kanban através de um harness de IA configurável.

---

## 2. Raias do Kanban

O fluxo segue a ordem estrita abaixo. Cada raia representa uma fase da execução:

| Ordem | Raia           | Descrição                                                        |
|-------|----------------|------------------------------------------------------------------|
| 1     | **Planning**    | Planejamento da task: escopo, subtasks, critérios de aceite      |
| 2     | **Todo**        | Backlog refinado e pronto para execução                          |
| 3     | **Doing**       | Implementação ativa pela harness                                 |
| 4     | **Validating**  | Revisão de código e validação de requisitos                      |
| 5     | **Testing**     | Criação e execução de testes automatizados e manuais             |
| 6     | **Done**        | Task concluída e entregue                                        |

Cada raia pode ser configurada com um **modelo de IA diferente** via variáveis de ambiente.

---

## 3. Stack Tecnológica

| Camada           | Tecnologia                              | Versão / Detalhes          |
|------------------|-----------------------------------------|----------------------------|
| Linguagem        | Go                                      | 1.26                       |
| HTTP Framework   | gin-gonic/gin                           | latest                     |
| MCP SDK          | modelcontextprotocol/go-sdk             | latest                     |
| CLI              | spf13/cobra                             | latest                     |
| Configuração     | spf13/viper                             | latest                     |
| Banco de Dados   | SQLite (via mattn/go-sqlite3)           | latest                     |
| Testes / Mocks   | stretchr/testify                        | latest                     |
| Frontend         | React + Material UI                     | React 18+, MUI 5+          |
| Comunicação RT   | SSE (Server-Sent Events)                | nativo                     |

---

## 4. Princípios e Padrões

### 4.1 Princípios SOLID

- **S** — Single Responsibility: cada arquivo e struct tem um único propósito.
- **O** — Open/Closed: extensão via interfaces, não modificação de structs concretas.
- **L** — Liskov Substitution: qualquer implementação de interface pode substituir outra sem quebra.
- **I** — Interface Segregation: interfaces pequenas e focadas (ex: `TaskCreator`, `TaskFinder`).
- **D** — Dependency Inversion: domínio define interfaces, infraestrutura implementa.

### 4.2 Clean Code

- Nomes descritivos e consistentes em inglês.
- Funções pequenas (máximo ~30 linhas).
- Sem comentários óbvios — o código deve ser autoexplicativo.
- Tratamento de erros explícito e contextualizado com `fmt.Errorf("context: %w", err)`.

### 4.3 Arquitetura Hexagonal (Ports & Adapters)

```
┌──────────────────────────────────────────────────────────┐
│                     ADAPTERS (IN)                        │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐ │
│  │ HTTP (Gin) │  │  CLI (Cobra)│  │  MCP Server (SDK) │ │
│  └─────┬──────┘  └─────┬──────┘  └────────┬───────────┘ │
│        │               │                   │             │
│  ══════╪═══════════════╪═══════════════════╪══════════   │
│        │         PORTS (IN)                │             │
│        │    ┌──────────────────┐           │             │
│        └───►│   Use Cases      │◄──────────┘             │
│             │  (Application)   │                         │
│             └────────┬─────────┘                         │
│                      │                                   │
│  ════════════════════╪═══════════════════════════════    │
│               PORTS (OUT)                                │
│             ┌────────┴─────────┐                         │
│             │    Domain        │                         │
│             │  (Entities +     │                         │
│             │   Interfaces)    │                         │
│             └────────┬─────────┘                         │
│                      │                                   │
│  ════════════════════╪═══════════════════════════════    │
│              ADAPTERS (OUT)                              │
│  ┌────────────┐  ┌────────────┐  ┌──────────────────┐   │
│  │  SQLite    │  │  Harness   │  │  Event Emitter   │   │
│  │  Repos     │  │  Client    │  │  (Observer)      │   │
│  └────────────┘  └────────────┘  └──────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### 4.4 Injeção de Dependências (DI In-Memory)

Um container simples em memória gerencia todas as dependências:

```go
// internal/di/container.go
type Container struct {
    mu       sync.RWMutex
    services map[string]any
}

func (c *Container) Register(name string, svc any)
func (c *Container) Resolve(name string) any
func (c *Container) MustResolve(name string) any
```

Todas as dependências são registradas no bootstrap da aplicação e resolvidas via container. Nenhum `new` direto em use cases.

### 4.5 Observer Pattern

O sistema de eventos é o coração da reatividade:

```go
// internal/domain/event/dispatcher.go
type EventType string

type Event struct {
    Type      EventType
    Payload   any
    Timestamp time.Time
    TaskID    string
}

type Handler func(event Event)

type Dispatcher interface {
    Subscribe(eventType EventType, handler Handler)
    SubscribeAll(handler Handler)  // Wildcard: recebe todos os eventos publicados
    Publish(event Event)
}
```

---

## 5. Diagrama de Sequência — Fluxo Completo

```
User        Frontend       HTTP         UseCase      Repository    Dispatcher    SSE         Orchestrator    Harness
 │              │            │              │             │             │          │              │              │
 │──create──►   │            │              │             │             │          │              │              │
 │              │──POST──►   │              │             │             │          │              │              │
 │              │            │──Execute──►  │             │             │          │              │              │
 │              │            │              │──Create──►  │             │          │              │              │
 │              │            │              │             │──persist──► │          │              │              │
 │              │            │              │             │◄──ok────── │          │              │              │
 │              │            │              │──Publish──► │             │          │              │              │
 │              │            │              │             │  task.created          │              │              │
 │              │            │              │             │             │──SSE──►  │              │              │
 │              │◄──SSE─────────────────────────────────────────────── │          │              │              │
 │              │            │              │             │             │──notify──►              │              │
 │              │            │              │             │             │          │──StartFlow──►│              │
 │              │            │              │             │             │          │              │──Dispatch──► │
 │              │            │              │             │             │          │              │              │──run harness
 │              │            │              │             │             │          │              │              │
 │              │            │          ◄────────────────MCP: report_progress─────────────────── │              │
 │              │            │              │             │             │──SSE──►  │              │              │
 │              │◄──SSE─────────────────────────────────────────────── │          │              │              │
 │              │            │          ◄────────────────MCP: complete_phase──────────────────── │              │
 │              │            │              │──Publish──► │             │          │              │              │
 │              │            │              │    phase.planning.completed          │              │              │
 │              │            │              │             │             │──notify──►              │              │
 │              │            │              │             │             │          │──next phase──►              │
 │              │            │              │             │             │          │              │   (repeat)   │
```

---

## 6. Resumo de Decisões Arquiteturais

| Decisão                               | Justificativa                                                       |
|---------------------------------------|---------------------------------------------------------------------|
| Hexagonal Architecture                | Desacoplamento total entre domínio e infraestrutura                 |
| Observer Pattern                      | Reatividade sem acoplamento direto entre componentes                |
| SSE ao invés de WebSocket             | Mais simples, unidirecional (server→client), suficiente para o caso |
| SQLite                                | Lightweight, zero-config, suficiente para o escopo                  |
| Repository + Query separados          | SRP: CRUD simples vs queries complexas com joins                    |
| DI Container in-memory                | Simplicidade sem framework pesado                                   |
| Arquivo por use case                  | SRP + facilita navegação e testes                                   |
| Modelo por raia configurável          | Flexibilidade para usar modelos especializados por fase             |
| MCP para comunicação harness→server   | Protocolo padrão para comunicação com LLMs                          |
| Cobra + Viper                         | Stack padrão Go para CLI + config                                   |
