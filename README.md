# Lab03 - Sistema de Leilões com Fechamento Automático

## Descrição

Sistema de leilões online desenvolvido em Go que implementa:
- Criação e gerenciamento de leilões
- Sistema de lances (bids) com processamento em batch
- **Fechamento automático de leilões baseado em tempo (Nova Funcionalidade!)**

## 🆕 Nova Funcionalidade: Fechamento Automático de Leilões

O sistema agora fecha automaticamente os leilões após um período configurável, utilizando goroutines do Go para processamento assíncrono e concorrente.

### Características Principais

- ⏰ **Fechamento Baseado em Tempo**: Leilões fecham automaticamente após `AUCTION_INTERVAL`
- 🔄 **Processamento Assíncrono**: Goroutine dedicada verifica e fecha leilões expirados
- 🔒 **Thread-Safe**: Usa Mutex e operações atômicas do MongoDB
- 📊 **Batch Processing**: Fecha múltiplos leilões de uma vez
- ⚙️ **Configurável**: Tempo ajustável via variável de ambiente

### Documentação Detalhada

- [Documentação Técnica da Implementação](AUCTION_AUTO_CLOSE.md)

## Requisitos

- Go 1.25.4 ou superior
- MongoDB 7.0 ou superior
- Docker e Docker Compose (opcional)

## Configuração

### Variáveis de Ambiente

Configure as seguintes variáveis no arquivo `cmd/auction/.env`:

```bash
# Duração do leilão (formatos aceitos: 30s, 5m, 1h, etc)
AUCTION_INTERVAL=20s

# Configuração de Batch de Lances
BATCH_INSERT_INTERVAL=20s
MAX_BATCH_SIZE=4

# MongoDB
MONGODB_URL=mongodb://admin:admin@mongodb:27017/auctions?authSource=admin
MONGODB_DB=auctions
```

## Instalação e Execução

### Com Docker Compose (Recomendado)

```bash
# Iniciar todos os serviços
docker-compose up -d

# Verificar logs
docker-compose logs -f app
```

### Manual

```bash
# Instalar dependências
go mod download

# Executar aplicação
go run cmd/auction/main.go
```

## API Endpoints

### Leilões

#### Criar Leilão
```bash
POST /auction
Content-Type: application/json

{
  "product_name": "iPhone 15 Pro",
  "category": "Eletrônicos",
  "description": "iPhone 15 Pro 256GB em excelente estado",
  "condition": 0
}
```

#### Listar Leilões
```bash
# Leilões ativos
GET /auction?status=0

# Leilões concluídos
GET /auction?status=1

# Filtrar por categoria
GET /auction?category=Eletrônicos

# Buscar por nome do produto
GET /auction?product_name=iPhone
```

#### Buscar Leilão por ID
```bash
GET /auction/:id
```

### Lances

#### Criar Lance
```bash
POST /bid
Content-Type: application/json

{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "auction_id": "auction-id-here",
  "amount": 1500.00
}
```

#### Buscar Lance Vencedor
```bash
GET /bid/:auction_id/winning
```

### Executar em Modo Desenvolvimento

```bash
# Ou executar diretamente
go run cmd/auction/main.go
```

## Troubleshooting

### Leilões não estão fechando

```bash
# Verificar se a goroutine está rodando
docker-compose logs app | grep "Auto-close"

# Verificar variável de ambiente
docker exec <container> env | grep AUCTION_INTERVAL
```

### Erro de Conexão com MongoDB

```bash
# Verificar se o MongoDB está rodando
docker ps | grep mongo

# Reiniciar MongoDB
docker-compose restart mongodb
```