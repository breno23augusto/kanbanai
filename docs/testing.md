# KanbanAI — Testes

## 1. Estratégia

- **Unit Tests**: todos os use cases, services e adapters com mocks via `testify/mock`.
- **Integration Tests**: repositories contra SQLite in-memory.
- **E2E**: fluxo completo criação → done com harness mockado.

---

## 2. Mocks com Testify

```go
// internal/domain/repository/mock/task_repository_mock.go
type MockTaskRepository struct {
    mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *entity.Task) error {
    args := m.Called(ctx, task)
    return args.Error(0)
}

func (m *MockTaskRepository) Find(ctx context.Context, id string) (*entity.Task, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*entity.Task), args.Error(1)
}

// ... demais métodos
```

### Convenção de Nomenclatura de Mocks

Mocks seguem o padrão: `Mock` + nome da interface.

| Interface             | Mock                        |
|-----------------------|-----------------------------|
| `TaskRepository`      | `MockTaskRepository`       |
| `Dispatcher`          | `MockDispatcher`            |
| `HarnessPort`         | `MockHarnessPort`           |
| `PhaseOutputRepository`| `MockPhaseOutputRepository`|

---

## 3. Exemplo de Teste de Use Case

```go
// internal/application/usecase/create_task_test.go
func TestCreateTask_Execute_Success(t *testing.T) {
    mockRepo := new(mocks.MockTaskRepository)
    mockDispatcher := new(mocks.MockDispatcher)

    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Task")).Return(nil)
    mockDispatcher.On("Publish", mock.AnythingOfType("event.Event")).Return()

    uc := usecase.NewCreateTask(mockRepo, mockDispatcher)

    input := dto.CreateTaskInput{
        Title:       "Implementar login",
        Description: "OAuth2 com Google",
    }

    result, err := uc.Execute(context.Background(), input)

    assert.NoError(t, err)
    assert.NotEmpty(t, result.ID)
    assert.Equal(t, entity.PhasePlanning, result.CurrentPhase)
    mockRepo.AssertExpectations(t)
    mockDispatcher.AssertExpectations(t)
}
```

---

## 4. Estrutura de Arquivos de Teste

Cada arquivo de produção tem um arquivo de teste correspondente com sufixo `_test.go`:

| Produção                    | Teste                              |
|-----------------------------|------------------------------------|
| `create_task.go`            | `create_task_test.go`              |
| `task_handler.go`           | `task_handler_test.go`             |
| `task_repository_sqlite.go` | `task_repository_sqlite_test.go`   |
| `dispatcher_memory.go`      | `dispatcher_memory_test.go`        |
| `broker.go`                 | `broker_test.go`                   |

---

## 5. Comando para Rodar Testes

```bash
go test ./... -v -race -cover
```
