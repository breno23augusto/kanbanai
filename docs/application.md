# KanbanAI — Camada de Aplicação

## 1. Fluxo Principal

```
┌──────────┐      POST /api/tasks       ┌──────────────┐
│ Frontend │ ───────────────────────────►│ HTTP Handler │
│ (React)  │                             └──────┬───────┘
│          │◄─── SSE /api/events ────────────┐  │
└──────────┘                                 │  │
                                             │  ▼
                                    ┌────────┴──────────┐
                                    │  CreateTaskUseCase │
                                    └────────┬──────────┘
                                             │
                              ┌──────────────┼──────────────┐
                              ▼              ▼              ▼
                    ┌──────────────┐ ┌──────────────┐ ┌───────────┐
                    │ TaskRepo     │ │ EventDispatch│ │ SSE Hub   │
                    │ (SQLite)     │ │ (Observer)   │ │ (Broker)  │
                    └──────────────┘ └──────┬───────┘ └───────────┘
                                            │
                                            ▼
                                   ┌─────────────────┐
                                   │ PhaseOrchestrator│
                                   └────────┬────────┘
                                            │
                                            ▼
                                   ┌─────────────────┐
                                   │ HarnessAdapter   │
                                   │ (CLI dispatch)   │
                                   └────────┬────────┘
                                            │
                              ┌─────────────┼─────────────┐
                              ▼             ▼             ▼
                         ┌────────┐   ┌──────────┐  ┌──────────┐
                         │ Claude │   │   Pi     │  │ Hermes   │
                         └────────┘   └──────────┘  └──────────┘
```

### 1.1 Passo a Passo

1. **Usuário cria task** no dashboard (React).
2. **POST `/api/tasks`** é enviado ao servidor Go.
3. **`CreateTaskUseCase`** valida e persiste no SQLite via `TaskRepository`.
4. **Evento `task.created`** é disparado pelo `EventDispatcher`.
5. O **SSE Hub** recebe o evento e envia para todos os clientes conectados.
6. O **`PhaseOrchestrator`** (subscriber de `task.created`) inicia o fluxo.
7. Evento **`phase.planning.started`** é disparado.
8. O **`HarnessAdapter`** monta o prompt com instruções MCP e dispara o harness.
9. O **harness executa** a fase, usando o MCP para reportar progresso ao servidor.
10. Via MCP, o harness chama tools como `update_task_phase`, `report_progress`, `complete_phase`.
11. Ao chamar `complete_phase`, evento **`phase.planning.completed`** é disparado.
12. O **`PhaseOrchestrator`** reage e inicia a próxima raia (Todo).
13. **Repete** até chegar em Done.
14. Evento **`phase.done.reached`** é disparado.

---

## 2. Use Cases

Cada use case é um arquivo dedicado com uma struct e um método `Execute`:

```go
// internal/application/usecase/create_task.go
type CreateTask struct {
    taskRepo   repository.TaskRepository
    dispatcher event.Dispatcher
}

func NewCreateTask(repo repository.TaskRepository, disp event.Dispatcher) *CreateTask

func (uc *CreateTask) Execute(ctx context.Context, input dto.CreateTaskInput) (*dto.TaskOutput, error) {
    // 1. Validar input
    // 2. Criar entidade Task
    // 3. Persistir via taskRepo.Create()
    // 4. Disparar evento task.created
    // 5. Retornar DTO de saída
}
```

### 2.1 Lista de Use Cases

| Use Case                | Arquivo                      | Responsabilidade                           |
|-------------------------|------------------------------|--------------------------------------------|
| `CreateTask`            | `create_task.go`             | Criar task e disparar fluxo                |
| `UpdateTask`            | `update_task.go`             | Atualizar dados da task                    |
| `DeleteTask`            | `delete_task.go`             | Remover task                               |
| `GetTask`               | `get_task.go`                | Buscar task por ID                         |
| `ListTasks`             | `list_tasks.go`              | Listar tasks com filtros                   |
| `AdvancePhase`          | `advance_phase.go`           | Concluir fase atual (persiste PhaseOutput, dispara phase.*.completed). **Não** inicia a próxima fase — isso é feito pelo PhaseOrchestrator como subscriber do evento. |
| `ReportPhaseProgress`   | `report_phase_progress.go`   | Registrar progresso de uma fase            |
| `SavePhaseOutput`       | `save_phase_output.go`       | Salvar artefatos/outputs de uma fase       |

---

## 3. DTOs

```go
// internal/application/dto/create_task_input.go
type CreateTaskInput struct {
    Title       string
    Description string
    Priority    int
}

// internal/application/dto/task_output.go
type TaskOutput struct {
    ID           string
    Title        string
    Description  string
    CurrentPhase entity.Phase
    Status       entity.Status
    Priority     int
    Version      int
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// internal/application/dto/task_filter.go
type TaskFilter struct {
    Phase  *entity.Phase
    Status *entity.Status
    Limit  int
    Offset int
}

// internal/application/dto/phase_progress.go
type PhaseProgress struct {
    TaskID   string
    Phase    entity.Phase
    Message  string
}
```

---

## 4. PhaseOrchestrator (Serviço de Aplicação)

O `PhaseOrchestrator` é o serviço central que coordena a execução das fases. Ele **não** é um use case — é um serviço de aplicação que reage a eventos e gerencia o ciclo de vida dos processos harness.

```go
// internal/application/service/phase_orchestrator.go
type PhaseOrchestrator struct {
    taskRepo        repository.TaskRepository
    phaseOutputRepo repository.PhaseOutputRepository
    harnessAdapter  port.HarnessPort
    promptBuilder   *PromptBuilder
    dispatcher      event.Dispatcher
    processRegistry map[string]*exec.Cmd  // taskID → harness process
    mu              sync.RWMutex
}
```

### 4.1 StartFlow — Início do Fluxo

Iniciado quando o evento `task.created` é disparado. Começa sempre pela fase `Planning`:

```go
func (o *PhaseOrchestrator) StartFlow(ctx context.Context, task *entity.Task) error {
    o.dispatcher.Publish(event.Event{
        Type: event.PhasePlanningStarted, TaskID: task.ID,
        Payload: map[string]any{"phase": entity.PhasePlanning},
    })
    prompt, err := o.promptBuilder.Build(entity.PhasePlanning, task)
    if err != nil {
        return fmt.Errorf("prompt build: %w", err)
    }
    if err := o.harnessAdapter.Dispatch(ctx, task, entity.PhasePlanning, prompt); err != nil {
        return fmt.Errorf("harness dispatch: %w", err)
    }
    return nil
}
```

### 4.2 AdvancePhase — Avançar para Próxima Fase

Chamado pelos subscribers de eventos `phase.*.completed`. **Não** é o mesmo que o use case `AdvancePhase`:

```go
func (o *PhaseOrchestrator) AdvancePhase(ctx context.Context, taskID string) error {
    task, err := o.taskRepo.Find(ctx, taskID)
    if err != nil {
        return fmt.Errorf("find task: %w", err)
    }
    nextPhase, hasNext := task.CurrentPhase.Next()
    if !hasNext || task.CurrentPhase.IsTerminal() {
        task.Status = entity.StatusCompleted
        task.UpdatedAt = time.Now()
        if err := o.taskRepo.Update(ctx, task); err != nil {
            return fmt.Errorf("update task: %w", err)
        }
        o.dispatcher.Publish(event.Event{
            Type: event.PhaseDoneReached, TaskID: task.ID,
        })
        return nil
    }
    task.CurrentPhase = nextPhase
    task.Status = entity.StatusPending
    task.UpdatedAt = time.Now()
    if err := o.taskRepo.Update(ctx, task); err != nil {
        return fmt.Errorf("update task: %w", err)
    }
    o.dispatcher.Publish(event.Event{
        Type: event.LaneTransitionCompleted, TaskID: task.ID,
        Payload: map[string]any{"from": task.CurrentPhase, "to": nextPhase},
    })
    return o.dispatchPhase(ctx, task, nextPhase)
}
```

### 4.3 dispatchPhase (método interno)

```go
func (o *PhaseOrchestrator) dispatchPhase(ctx context.Context, task *entity.Task, phase entity.Phase) error {
    task.Status = entity.StatusInProgress
    task.UpdatedAt = time.Now()
    if err := o.taskRepo.Update(ctx, task); err != nil {
        return fmt.Errorf("update task status: %w", err)
    }
    o.dispatcher.Publish(event.Event{
        Type: event.PhaseEvent(phase, "started"), TaskID: task.ID,
        Payload: map[string]any{"phase": phase},
    })
    prompt, err := o.promptBuilder.Build(phase, task)
    if err != nil {
        return fmt.Errorf("prompt build: %w", err)
    }
    return o.harnessAdapter.Dispatch(ctx, task, phase, prompt)
}
```

### 4.4 KillProcess — Interrupção de Harness

Usado quando uma task é deletada durante execução:

```go
func (o *PhaseOrchestrator) KillProcess(taskID string) {
    o.mu.RLock()
    cmd, exists := o.processRegistry[taskID]
    o.mu.RUnlock()
    if exists && cmd.Process != nil {
        cmd.Process.Signal(syscall.SIGKILL)
        o.mu.Lock()
        delete(o.processRegistry, taskID)
        o.mu.Unlock()
    }
}
```

### 4.5 Distinção Crítica: `PhaseOrchestrator.AdvancePhase` vs Use Case `AdvancePhase`

| Método | Quem chama | O que faz |
|--------|-----------|-----------|
| **Use Case `AdvancePhase`** (`advance_phase.go`) | MCP tool `complete_phase` | **Persiste a conclusão da fase**: salva `PhaseOutput`, atualiza `status=completed`, dispara `phase.<phase>.completed`. **Não** inicia a próxima fase. |
| **`PhaseOrchestrator.AdvancePhase`** | Event subscribers (`phase.*.completed`) | **Inicia a próxima fase**: atualiza `current_phase`, reseta `status=pending`, dispara harness para a nova fase. |

Este design em dois passos evita loop infinito: a tool MCP conclui a fase → evento disparado → orchestrator reage e inicia a próxima. O use case nunca chama o orchestrator diretamente, e o orchestrator nunca chama o use case.

### 4.6 Retry Handler

```go
func (o *PhaseOrchestrator) HandleRetry(ctx context.Context, taskID string, phase entity.Phase, attempt int, maxRetries int) {
    if attempt > maxRetries {
        task, _ := o.taskRepo.Find(ctx, taskID)
        task.Status = entity.StatusFailed
        task.UpdatedAt = time.Now()
        _ = o.taskRepo.Update(ctx, task)
        o.dispatcher.Publish(event.Event{
            Type: event.PhaseEvent(phase, "failed"), TaskID: taskID,
            Payload: map[string]any{"phase": phase, "attempt": attempt},
        })
        return
    }
    time.Sleep(time.Duration(2*attempt) * time.Second)
    o.dispatcher.Publish(event.Event{
        Type: event.PhaseEvent(phase, "retry"), TaskID: taskID,
        Payload: map[string]any{"phase": phase, "attempt": attempt},
    })
    task, _ := o.taskRepo.Find(ctx, taskID)
    _ = o.dispatchPhase(ctx, task, phase)
}
```

---

## 5. PromptBuilder — Modelos de Prompts por Fase

O `PromptBuilder` gera prompts direcionados e customizados para cada fase com diretrizes rígidas sobre o uso de ferramentas MCP:

- **Planning**:
  > Você é um arquiteto de software. Analise os requisitos da task "{{.Task.Title}}". Identifique e salve as subtasks e critérios de aceite usando `update_task_output`. Reporte o progresso com `report_progress`. Finalize executando `complete_phase`.
- **Todo**:
  > Você é um Product Owner / Tech Lead. Pegue o planejamento gerado na fase anterior e refine as subtasks em histórias de usuário menores e detalhadas. Atualize os outputs e finalize o refinamento.
- **Doing**:
  > Você é um Engenheiro de Software Sênior. Implemente a solução da task no repositório. Produza código limpo e coeso. Salve o relatório da implementação e os arquivos modificados.
- **Validating**:
  > Você é um Quality Assurance / Reviewer. Analise o código produzido na fase anterior. Realize análises estáticas e valide se todos os critérios de aceite foram atendidos.
- **Testing**:
  > Você é um Engenheiro de Testes de Software. Escreva e execute os testes automatizados unitários/integração para cobrir o código implementado na fase Doing.

### Exemplo de Prompt para Planning

```
Você é um agente de planejamento de software. Sua tarefa é planejar a implementação da seguinte task:

**Task**: {{ .Task.Title }}
**Descrição**: {{ .Task.Description }}

Você tem acesso a um servidor MCP para reportar seu progresso e resultados.

## Ferramentas MCP Disponíveis

- `report_progress(task_id, message)` — Reporte progresso parcial
- `update_task_output(task_id, phase, output)` — Salve artefatos da fase
- `complete_phase(task_id, phase, summary)` — Marque a fase como concluída

## Instruções

1. Analise os requisitos da task
2. Defina subtasks necessárias
3. Estabeleça critérios de aceite
4. Use `report_progress` para informar o andamento
5. Use `update_task_output` para salvar o plano
6. Use `complete_phase` para finalizar

MCP Server: {{ .MCPServerURL }}
Task ID: {{ .Task.ID }}
Phase: planning
```
