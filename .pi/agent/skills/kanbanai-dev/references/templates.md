# Code Templates

## Use Case Template

```go
// internal/application/usecase/my_new_use_case.go
package usecase

type MyNewUseCase struct {
    taskRepo   repository.TaskRepository
    dispatcher event.Dispatcher
}

func NewMyNewUseCase(repo repository.TaskRepository, disp event.Dispatcher) *MyNewUseCase {
    return &MyNewUseCase{taskRepo: repo, dispatcher: disp}
}

func (uc *MyNewUseCase) Execute(ctx context.Context, input dto.MyInput) (*dto.MyOutput, error) {
    // 1. Validate input
    // 2. Operate on domain entities
    // 3. Persist via repository
    // 4. Publish events via dispatcher
    // 5. Return DTO
}
```

## Repository Implementation Template

```go
// internal/adapter/out/persistence/repository/my_repository_sqlite.go
package repository

type MyRepositorySQLite struct {
    db *sql.DB
}

func NewMyRepositorySQLite(db *sql.DB) *MyRepositorySQLite {
    return &MyRepositorySQLite{db: db}
}

func (r *MyRepositorySQLite) Create(ctx context.Context, entity *entity.MyEntity) error {
    // Use prepared statements
    // Handle optimistic locking if applicable
}
```

## HTTP Handler Template

```go
// internal/adapter/in/http/handler/my_handler.go
package handler

type MyHandler struct {
    myUseCase *usecase.MyNewUseCase
}

func NewMyHandler(container *di.Container) *MyHandler {
    return &MyHandler{
        myUseCase: container.MustResolve("myNewUseCase").(*usecase.MyNewUseCase),
    }
}

func (h *MyHandler) Handle(c *gin.Context) {
    // Parse request
    // Call use case
    // Return standardized response
}
```

## Bootstrap Registration Template

In `internal/adapter/bootstrap/bootstrap.go`:

```go
// Register implementation
myRepo := repository.NewMyRepositorySQLite(db)
container.Register("myRepo", myRepo)

// Register use case
myUseCase := usecase.NewMyNewUseCase(myRepo, dispatcher)
container.Register("myNewUseCase", myUseCase)

// Wire events if needed
dispatcher.Subscribe(event.SomeEvent, func(evt event.Event) {
    // ...
})
```

## Test Template

```go
// internal/application/usecase/my_new_use_case_test.go
package usecase_test

func TestMyNewUseCase_Execute_Success(t *testing.T) {
    mockRepo := new(mocks.MockTaskRepository)
    mockDispatcher := new(mocks.MockDispatcher)

    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Task")).Return(nil)
    mockDispatcher.On("Publish", mock.AnythingOfType("event.Event")).Return()

    uc := usecase.NewMyNewUseCase(mockRepo, mockDispatcher)
    result, err := uc.Execute(context.Background(), input)

    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
    mockDispatcher.AssertExpectations(t)
}
```
