# KanbanAI — Frontend (React + MUI)

## 1. Componentes Principais

| Componente           | Responsabilidade                                           |
|----------------------|------------------------------------------------------------|
| `KanbanBoard`        | Grid com as 6 raias, drag indicator visual                 |
| `KanbanLane`         | Coluna individual com header, contagem e cards             |
| `TaskCard`           | Card com título, status, fase atual e indicador de progresso |
| `CreateTaskDialog`   | Dialog MUI para criar nova task                            |
| `TaskDetailDrawer`   | Drawer lateral com detalhes, outputs, subtasks e timeline    |
| `MarkdownView`        | Renderiza o output da fase como markdown GFM (headings, tabelas, code, listas de tarefas) |
| `EventTimeline`      | Timeline vertical dos eventos da task                      |
| `PhaseProgress`      | Stepper MUI mostrando progresso entre fases                |

---

## 2. Estrutura de Diretórios do Frontend

```
frontend/
├── package.json
├── src/
│   ├── App.tsx
│   ├── main.tsx
│   ├── theme/
│   │   └── theme.ts                     # Tema MUI customizado
│   ├── hooks/
│   │   ├── useSSE.ts                    # Hook para SSE
│   │   └── useTasks.ts                  # Hook para tasks
│   ├── services/
│   │   └── api.ts                       # Client HTTP
│   ├── components/
│   │   ├── KanbanBoard.tsx              # Board principal
│   │   ├── KanbanLane.tsx               # Raia individual
│   │   ├── TaskCard.tsx                 # Card da task
│   │   ├── CreateTaskDialog.tsx         # Modal de criação
│   │   ├── TaskDetailDrawer.tsx         # Drawer de detalhes
│   │   ├── MarkdownView.tsx             # Renderizador markdown (react-markdown + remark-gfm) dos outputs de fase, com checkboxes de subtask
│   │   ├── EventTimeline.tsx            # Timeline de eventos
│   │   └── PhaseProgress.tsx            # Progresso da fase
│   ├── pages/
│   │   └── Dashboard.tsx                # Página principal
│   └── types/
│       ├── task.ts                      # Tipos de task
│       └── event.ts                     # Tipos de evento
└── public/
    └── index.html
```

---

## 3. Hook SSE

```typescript
// frontend/src/hooks/useSSE.ts
export function useSSE(url: string) {
    const [events, setEvents] = useState<SSEEvent[]>([]);

    useEffect(() => {
        const source = new EventSource(url);

        source.addEventListener('task.created', (e) => { /* ... */ });
        source.addEventListener('phase.planning.started', (e) => { /* ... */ });
        // ... demais eventos

        return () => source.close();
    }, [url]);

    return { events };
}
```

---

## 4. State Management

O frontend React é gerido por uma arquitetura leve, utilizando Vite e **React Context API**:

- **TaskContext**: Mantém a lista atualizada de tasks e expõe funções de alteração de estado (`createTask`, `updateTask`).
- **useSSE**: Escuta mensagens do endpoint `/api/v1/events` e despacha mutações diretamente para o `TaskContext` de acordo com o tipo de evento:
  - `task.status_changed` → move card para nova coluna
  - `phase.doing.progress` → atualiza barra de progresso no card
  - `task.created` / `task.deleted` → insere/remove card
- **Vite Config**: O backend URL do Go é repassado dinamicamente via `VITE_API_BASE_URL`.

### 4.1 Fluxo de Integração

1. **Carregamento Inicial**: Na montagem da página `Dashboard.tsx`, o hook `useTasks.ts` executa `GET /api/v1/tasks` e popula as 6 raias do `KanbanBoard`.
2. **Conexão Real-Time**: O hook `useSSE.ts` inicia `new EventSource('/api/v1/events')`.
3. **Mutações Reativas**:
   - `task.status_changed` → Contexto React (`TaskContext`) move o card para a nova coluna (`current_phase`) e aplica estilização do status (`pending`, `in_progress`, `failed`, `completed`).
   - `phase.*.progress` → Card exibe barra de progresso linear e log de atividades do agente.
   - `task.created` / `task.deleted` → Card inserido/removido instantaneamente.
   - Desconexão → EventSource reconecta automaticamente e re-lista tasks silenciosamente.
