## Estrutura do Projeto

```text
cep-module5/
├── cmd/
│   └── cep/
│       └── main.go             # Ponto de entrada: faz o "wiring" (liga as peças), pré-aloca as memórias estáticas e inicia as 4 Goroutines.
├── internal/
│   ├── domain/                 # O "coração" conceitual: contratos e estruturas de dados.
│   │   └── models.go           # Onde vamos codificar os structs das Tabelas 3.5.1 (Entrada) e 3.5.2 (Saída).
│   ├── ingest/                 # Estágio 1: Recepção e Buffering (Hot Path).
│   │   ├── receiver.go         # Listener UDP Broadcast e lógica de descarte silencioso.
│   │   └── ringbuffer.go       # Implementação do Buffer Circular de 150 MB.
│   ├── core/                   # Estágio 2: Motor CEP e Inteligência.
│   │   ├── engine.go           # Orquestrador da janela de 60s e regra de disparo (> 30 eventos).
│   │   └── h3map.go            # Tabela Hash pré-alocada (128 MB) para indexação espacial.
│   ├── output/                 # Estágios 3 e 4: Filas de saída assimétricas.
│   │   ├── queues.go           # Filas circulares de 24 MB (Persistência) e 8 MB (Notificação).
│   │   ├── db_writer.go        # Thread de gravação seletiva no Incident Log.
│   │   └── broadcaster.go      # Thread de envio JSON via UDP Broadcast para o Módulo 3.
│   └── config/                 # Parametrização estática do sistema.
│       └── config.go           # Portas UDP, tamanhos de buffer pré-definidos.
├── go.mod                      
└── go.sum
```
