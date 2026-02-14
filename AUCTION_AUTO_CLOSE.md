# Sistema de Fechamento Automático de Leilões

## Visão Geral

Esta implementação adiciona funcionalidade de fechamento automático de leilões baseado em tempo, utilizando goroutines do Go para processar leilões expirados de forma assíncrona e concorrente.

## Arquitetura da Solução

### 1. Cálculo do Tempo do Leilão (`getAuctionDuration()`)

A função `getAuctionDuration()` calcula a duração do leilão baseada na variável de ambiente `AUCTION_INTERVAL`:

```go
func getAuctionDuration() time.Duration {
    auctionInterval := os.Getenv("AUCTION_INTERVAL")
    duration, err := time.ParseDuration(auctionInterval)
    if err != nil {
        logger.Error("Error parsing AUCTION_INTERVAL, using default 5 minutes", err)
        return 5 * time.Minute
    }
    return duration
}
```

**Características:**
- Lê a variável de ambiente `AUCTION_INTERVAL` (ex: "20s", "5m", "1h")
- Se não configurada ou inválida, usa valor padrão de 5 minutos
- Suporta múltiplos formatos: segundos (s), minutos (m), horas (h)

### 2. Goroutine de Fechamento Automático (`startAutoCloseRoutine()`)

Esta goroutine é iniciada automaticamente quando o `AuctionRepository` é criado:

```go
func (ar *AuctionRepository) startAutoCloseRoutine(ctx context.Context) {
    go func() {
        checkInterval := ar.auctionInterval / 2
        if checkInterval < 10*time.Second {
            checkInterval = 10 * time.Second
        }
        
        ticker := time.NewTicker(checkInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                logger.Info("Auto-close auction routine stopped")
                return
            case <-ticker.C:
                ar.closeExpiredAuctions(context.Background())
            }
        }
    }()
}
```

**Características:**
- Executa em uma goroutine separada (não bloqueia a aplicação)
- Verifica leilões expirados a cada metade do `AUCTION_INTERVAL` (mínimo de 10 segundos)
- Utiliza `time.Ticker` para execução periódica
- Respeita o contexto, permitindo cancelamento gracioso
- Logs informativos sobre início e parada da rotina

### 3. Fechamento de Leilões Expirados (`closeExpiredAuctions()`)

Esta função busca e fecha todos os leilões que ultrapassaram o tempo limite:

```go
func (ar *AuctionRepository) closeExpiredAuctions(ctx context.Context) {
    ar.mu.Lock()
    defer ar.mu.Unlock()

    expirationTime := time.Now().Add(-ar.auctionInterval).Unix()

    filter := bson.M{
        "status":    auction_entity.Active,
        "timestamp": bson.M{"$lte": expirationTime},
    }

    update := bson.M{
        "$set": bson.M{
            "status": auction_entity.Completed,
        },
    }

    result, err := ar.Collection.UpdateMany(ctx, filter, update)
    // ... tratamento de erro e log
}
```

**Características:**
- **Thread-safe**: Usa `sync.Mutex` para evitar race conditions
- **Batch processing**: Atualiza múltiplos leilões em uma única operação
- **Eficiente**: Usa `UpdateMany` do MongoDB para performance
- Calcula tempo de expiração: `now - AUCTION_INTERVAL`
- Busca apenas leilões `Active` que expiraram
- Atualiza status para `Completed`
- Log informativo quando leilões são fechados

## Sincronização e Concorrência

A solução implementa várias estratégias para lidar com concorrência:

### 1. Mutex para Proteção de Dados
```go
type AuctionRepository struct {
    Collection      *mongo.Collection
    auctionInterval time.Duration
    mu              sync.Mutex  // Protege operações de atualização
}
```

### 2. Operações Atômicas no MongoDB
- Utiliza `UpdateMany` que é uma operação atômica no MongoDB
- Garante consistência mesmo com múltiplas instâncias da aplicação

### 3. Integração com Sistema de Bids
O sistema de bids já valida se o leilão está fechado:
```go
// Trecho de internal/infra/database/bid/create_bid.go
if auctionStatus == auction_entity.Completed || now.After(auctionEndTime) {
    return  // Não permite lance em leilão fechado/expirado
}
```

## Testes

### Testes Implementados

1. **TestAutoCloseExpiredAuctions**: Testa fechamento de um único leilão expirado
2. **TestAutoCloseMultipleExpiredAuctions**: Testa fechamento de múltiplos leilões
3. **TestGetAuctionDuration**: Testa parsing da variável de ambiente

### Executando os Testes

```bash
# Executar todos os testes
go test ./internal/infra/database/auction/...

# Executar com verbose
go test -v ./internal/infra/database/auction/...

# Executar teste específico
go test -v -run TestAutoCloseExpiredAuctions ./internal/infra/database/auction/...
```

**Importante**: Os testes requerem:
- MongoDB rodando (local ou container)
- Variável de ambiente `MONGODB_URL` configurada ou usa padrão: `mongodb://admin:admin@localhost:27017/auctions_test?authSource=admin`

## Configuração

### Variáveis de Ambiente

No arquivo `.env`:
```bash
AUCTION_INTERVAL=20s        # Duração do leilão
BATCH_INSERT_INTERVAL=20s   # Intervalo para batch de bids
MAX_BATCH_SIZE=4            # Tamanho máximo do batch

MONGODB_URL=mongodb://admin:admin@mongodb:27017/auctions?authSource=admin
MONGODB_DB=auctions
```

### Valores Aceitos para AUCTION_INTERVAL
- Segundos: `10s`, `30s`, `60s`
- Minutos: `1m`, `5m`, `30m`
- Horas: `1h`, `2h`, `24h`
- Combinações: `1h30m`, `2h30m45s`

## Fluxo de Funcionamento

1. **Criação do Repository**
   - `NewAuctionRepository()` é chamado
   - Lê `AUCTION_INTERVAL` das variáveis de ambiente
   - Inicia goroutine de fechamento automático

2. **Execução Periódica**
   - Ticker dispara a cada `AUCTION_INTERVAL / 2` (mínimo 10s)
   - `closeExpiredAuctions()` é chamada

3. **Fechamento de Leilões**
   - Calcula timestamp de expiração
   - Busca leilões ativos expirados
   - Atualiza status para `Completed` em batch
   - Loga quantidade de leilões fechados

4. **Validação em Bids**
   - Sistema de bids já valida status antes de aceitar lance
   - Impede lances em leilões fechados ou expirados

## Melhorias Implementadas

### Comparado com Implementação Manual:
1. **Automático**: Não requer intervenção manual
2. **Eficiente**: Atualiza múltiplos leilões de uma vez
3. **Thread-safe**: Protegido contra race conditions
4. **Configurável**: Tempo ajustável via variável de ambiente
5. **Testável**: Testes automatizados garantem funcionamento
6. **Observável**: Logs informativos para monitoramento

## Monitoramento

A solução inclui logs em pontos-chave:
- Início da goroutine: `"Auto-close auction routine started"`
- Fechamento de leilões: `"Closed expired auctions"`
- Erros: `"Error trying to close expired auctions"`
- Parada da goroutine: `"Auto-close auction routine stopped"`

## Considerações de Produção

1. **Escalabilidade**: A solução funciona com múltiplas instâncias devido ao uso de operações atômicas no MongoDB
2. **Performance**: Verificação periódica minimiza carga no banco
3. **Confiabilidade**: Mutex e operações atômicas garantem consistência
4. **Manutenibilidade**: Código limpo, testável e bem documentado

## Estrutura de Arquivos

```
internal/infra/database/auction/
├── create_auction.go       # Implementação principal
├── create_auction_test.go  # Testes automatizados
└── find_auction.go         # Funções de busca (existente)
```

## Integração com Sistema Existente

A solução integra-se perfeitamente com:
- Sistema de criação de leilões (auction_entity)
- Sistema de lances (bid_entity)
- Validação de leilões na criação de bids
- Infraestrutura de logging existente

## Conclusão

Esta implementação fornece uma solução robusta, testável e eficiente para fechamento automático de leilões, seguindo as melhores práticas do Go para concorrência e mantendo consistência com a arquitetura existente do projeto.
