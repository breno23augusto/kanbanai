# KanbanAI — Regras de Negócio e Ciclo de Vida

## 1. Controle de Concorrência (Optimistic Locking)

Para evitar race conditions de múltiplas requisições simultâneas (harnesses ou usuários editando a mesma task):

1. A tabela `tasks` possui a coluna `version INTEGER`.
2. Toda query de atualização verifica se a versão enviada bate com a versão do banco:
   ```sql
   UPDATE tasks 
   SET title = ?, description = ?, current_phase = ?, status = ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
   WHERE id = ? AND version = ?;
   ```
3. Se `rows affected = 0`, o repositório retorna `ErrConcurrentModification`.
4. Os Use Cases capturam este erro e efetuam retries automáticos (até 3 tentativas) recarregando a entidade e tentando novamente.

---

## 2. Regras de Fluxo e Ciclo de Vida do Agente (MCP & Orquestração)

### 2.1 Associação e Validação de Task no Servidor MCP

- Sempre que o `HarnessAdapter` spawna um processo filho do harness, ele injeta a variável de ambiente `KANBANAI_TASK_ID`.
- O servidor MCP, ao receber requisições de ferramentas (`report_progress`, `update_task_output`, `complete_phase`), **valida** se o parâmetro `task_id` fornecido corresponde exatamente ao `KANBANAI_TASK_ID` que o processo foi autorizado a manipular. Caso contrário, retorna erro de segurança.

### 2.2 Máquina de Estados do Kanban

O ciclo de estados internos segue estritamente:

1. **Transição de Raia**: Quando a fase avança (ex: Planning → Todo), `current_phase` é atualizada e `status` é setado para `pending`.
2. **Sinal de Início**: Assim que o harness é iniciado ou executa a primeira chamada MCP, `status` vai para `in_progress`.
3. **Sucesso**: Quando `complete_phase` é executada com sucesso, `status = completed`. O orquestrador reage e avança a fase resetando para `status = pending` na nova fase (ou finaliza em `completed` se for `Done`).

### 2.3 Transições Válidas de Status

```
StatusPending    -> StatusInProgress, StatusCancelled
StatusInProgress -> StatusCompleted, StatusFailed, StatusCancelled
StatusCompleted  -> StatusPending (próxima fase) ou terminal
StatusFailed e StatusCancelled são terminais (só saem via intervenção manual)
```

### 2.4 Monitoramento de Encerramento e Falha do Processo

- A goroutine `monitorProcess` mapeia o ciclo de vida do comando CLI do harness.
- Se o processo terminar com exit code ≠ 0 ou estourar `TimeoutSec`, a goroutine delega a falha ao orchestrator.
- O orchestrator tenta retry seguro (backoff linear: `2 * tentativa` segundos).
- Se `tentativas > MaxRetries`, `status = failed` e a execução é bloqueada, necessitando intervenção manual (botão "Restart Phase" no frontend).
- Se a task for deletada durante execução, o `PhaseOrchestrator` envia `SIGKILL` ao processo CLI associado.

---

## 3. Distinção Crítica: Use Case `AdvancePhase` vs `PhaseOrchestrator.AdvancePhase`

| Método | Quem chama | O que faz |
|--------|-----------|-----------|
| **Use Case `AdvancePhase`** | MCP tool `complete_phase` | **Persiste a conclusão da fase**: salva `PhaseOutput`, atualiza `status=completed`, dispara `phase.<phase>.completed`. **Não** inicia a próxima fase. |
| **`PhaseOrchestrator.AdvancePhase`** | Event subscribers (`phase.*.completed`) | **Inicia a próxima fase**: atualiza `current_phase`, reseta `status=pending`, dispara harness para a nova fase. |

Este design em dois passos evita loop infinito: a tool MCP conclui a fase → evento disparado → orchestrator reage e inicia a próxima. O use case **nunca** chama o orchestrator diretamente, e o orchestrator **nunca** chama o use case.

---

## 4. Política de Retry

| Parâmetro     | Valor                          |
|---------------|--------------------------------|
| Backoff       | Linear: `2 * tentativa` segundos |
| Max Retries   | Configurável por fase (default: 3) |
| Timeout       | Configurável por fase (default: 600s, Doing/Testing: 900s) |
| Falha Terminal| `status = failed`, evento `phase.<fase>.failed` |

---

## 5. Regras de Segurança MCP

- Cada processo harness recebe um `KANBANAI_TASK_ID` único.
- Toda tool MCP valida que o `task_id` do argumento bate com o `KANBANAI_TASK_ID` do processo.
- Tentativas de acessar tasks não autorizadas resultam em erro imediato.
