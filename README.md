# WeiCloth

### 1. Levantar el servidor 

ng serve -o

>Start only the kafka instance

>>Up

```code
docker compose up kafka -d
```
>>Test

```code
go test ./internal/adapters/event_publisher/kafka/ -tags=integration -v
```