# KanbanAI — Convenções de Nomenclatura

Todas as convenções abaixo são **obrigatórias** para qualquer código adicionado ao projeto.

| Elemento             | Convenção                          | Exemplo                               |
|----------------------|------------------------------------|---------------------------------------|
| Package              | lowercase singular                 | `usecase`, `handler`, `entity`        |
| Interface            | PascalCase, noun                   | `TaskRepository`, `Dispatcher`        |
| Struct               | PascalCase, noun                   | `CreateTask`, `SSEBroker`             |
| Método               | PascalCase, verb+noun              | `Execute`, `FindByFilters`            |
| Arquivo              | snake_case                         | `create_task.go`, `task_handler.go`   |
| Teste                | `<filename>_test.go`               | `create_task_test.go`                 |
| Constante            | PascalCase com prefixo do tipo     | `PhasePlanning`, `StatusActive`       |
| Variável de ambiente | UPPER_SNAKE_CASE com prefixo       | `KANBANAI_SERVER_PORT`                |
| Evento               | dot.notation lowercase             | `task.created`, `phase.planning.started` |
| DTO                  | PascalCase + Input/Output/Filter   | `CreateTaskInput`, `TaskOutput`       |
| Repository impl      | Interface + Storage suffix         | `TaskRepositorySQLite`                |
| Query impl           | Interface + Storage suffix         | `TaskWithPhasesQuerySQLite`           |
| Mock                 | Mock + Interface name              | `MockTaskRepository`                  |

---

## 2. Regras de Código (Clean Code)

- Nomes descritivos e consistentes em **inglês**.
- Funções pequenas (máximo ~30 linhas).
- Sem comentários óbvios — o código deve ser autoexplicativo.
- Tratamento de erros explícito e contextualizado com `fmt.Errorf("context: %w", err)`.
- Um arquivo por struct/use case (Single Responsibility).

---

## 3. Estrutura de Arquivos por Camada

| Camada          | Diretório                              | Exemplo de Arquivo          |
|-----------------|----------------------------------------|-----------------------------|
| Entidade        | `internal/domain/entity/`              | `task.go`                   |
| Interface Repo  | `internal/domain/repository/`          | `task_repository.go`        |
| Interface Query | `internal/domain/query/`               | `task_with_phases_query.go` |
| Porta de Saída  | `internal/domain/port/`                | `harness_port.go`           |
| Evento          | `internal/domain/event/`               | `types.go`, `dispatcher.go` |
| Use Case        | `internal/application/usecase/`        | `create_task.go`            |
| DTO             | `internal/application/dto/`            | `create_task_input.go`      |
| Serviço         | `internal/application/service/`        | `phase_orchestrator.go`     |
| Adapter IN      | `internal/adapter/in/<tipo>/`          | `task_handler.go`           |
| Adapter OUT     | `internal/adapter/out/<tipo>/`         | `task_repository_sqlite.go` |
| Config          | `config/`                              | `config.go`                 |
| CLI             | `internal/adapter/in/cli/`             | `serve.go`                  |
| MCP             | `internal/adapter/in/mcp/`             | `tool_complete_phase.go`    |
| Bootstrap       | `internal/adapter/bootstrap/`          | `bootstrap.go`              |
| DI Container    | `internal/di/`                         | `container.go`              |
| PKG (utils)     | `pkg/<nome>/`                           | `uid/generator.go`          |

---

## 4. Padrão de Eventos

Eventos seguem o formato `categoria.ação[.detalhe]`:

- `task.created`, `task.updated`, `task.deleted`, `task.status_changed`
- `lane.transition.started`, `lane.transition.completed`, `lane.transition.failed`
- `phase.<fase>.started`, `phase.<fase>.progress`, `phase.<fase>.retry`, `phase.<fase>.completed`, `phase.<fase>.failed`
- `harness.command.dispatched`, `harness.output.received`, `harness.error.occurred`
- `system.health.check`, `system.error`
- `sse.client.connected`, `sse.client.disconnected`

---

## 5. Padrão de Registro no DI Container

Nomes de registro no container seguem camelCase descritivo:

| Nome                    | Tipo                              |
|-------------------------|-----------------------------------|
| `taskRepo`              | `repository.TaskRepository`       |
| `eventLogRepo`          | `repository.TaskEventLogRepository`|
| `phaseOutputRepo`       | `repository.PhaseOutputRepository`|
| `dispatcher`            | `event.Dispatcher`                |
| `sseBroker`             | `*sse.Broker`                     |
| `harnessAdapter`        | `port.HarnessPort`                |
| `createTaskUseCase`     | `*usecase.CreateTask`             |
| `advancePhaseUseCase`   | `*usecase.AdvancePhase`           |
| `orchestrator`          | `*service.PhaseOrchestrator`      |
