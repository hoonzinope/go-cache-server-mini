# go-cache-server-mini

## 개요
`go-cache-server-mini`는 Gin 기반의 HTTP API를 통해 키-값 쌍을 저장·조회할 수 있는 초경량 인메모리 캐시 서버입니다. 학습용 혹은 간단한 PoC(Service mock) 환경에서 빠르게 상태 저장이 필요할 때 사용할 수 있도록 최소한의 러닝 커브로 설계되었습니다.

## 주요 기능
- 캐시 작성/조회/삭제: `POST /set`, `GET /get`, `DELETE /del`
- 메타 정보: `GET /exists`, `GET /keys`, `GET /ttl`
- 유지 관리: `POST /expire`, `POST /flush`, `GET /ping`
- `json.RawMessage`로 값을 저장해 JSON 문자열·객체·숫자를 그대로 유지
- `sync.Map` 기반 동시성 안전 + 중앙집중 에러(`internal/errors.go`)

## 프로젝트 구조
```
cmd/main.go                 # 서버 엔트리포인트, 시그널 처리 및 graceful shutdown
internal/api/api.go         # Gin 서버 부트스트랩 및 라우터 구성
internal/api/handler/*.go   # HTTP 핸들러 (set/get/del/...)
internal/api/dto/*.go       # 요청/응답 DTO
internal/core/core.go       # Cache 타입, TTL 및 만료 워커
internal/core/cache_interface.go
internal/config/config.go   # YAML 설정 로더
internal/errors.go          # 공용 에러 정의
config.yml                  # 기본 TTL, HTTP 설정
```

## 빠른 시작
1. Go 1.24.5 이상을 설치합니다.
2. 의존성을 받으면서 서버를 실행합니다.
   ```bash
   go run ./cmd
   ```
3. 바이너리가 필요하다면 다음과 같이 빌드합니다.
   ```bash
   go build -o bin/cache-server ./cmd
   ```

## 설정
`config.yml`을 수정하거나 환경 변수를 사용하세요.

```yaml
ttl:
  default: 86400   # 요청에서 TTL을 비우거나 0 이하로 주면 이 값 사용
  max: 604800      # TTL이 이 값보다 크면 자동으로 잘림
http:
  enabled: true
  address: ":8080"
```

YAML 파일 안에 `${PORT}`처럼 환경변수를 넣으면 실행 중 `os.ExpandEnv`로 치환됩니다.

## Graceful shutdown & 오류 전파
- `cmd/main.go`는 `SIGINT`, `SIGTERM`을 기다리다가 시그널이 들어오면 `context.CancelFunc`를 호출해 API 서버와 TTL 만료 워커를 순차적으로 내려줍니다.
- `api.StartAPIServer`가 포트 바인딩 실패 등으로 즉시 종료되면 해당 오류가 메인 루프로 전달되어 프로세스가 `os.Exit(1)`로 종료됩니다. 덕분에 “포트 이미 사용 중” 같은 문제가 발생해도 고루틴이 조용히 종료되어 서비스가 멈춘 채 남아있지 않습니다.

## API 사용 예시
```bash
# 값 저장
curl -X POST http://localhost:8080/set \
  -H "Content-Type: application/json" \
  -d '{"key":"greeting","value":"\"hello\""}'

# 값 조회
curl "http://localhost:8080/get?key=greeting"
```
`value` 필드는 JSON RawMessage로 저장되므로 문자열, 객체, 숫자 등 어떤 JSON 타입도 그대로 유지됩니다.

## 개발 및 테스트
- 일반 테스트: `go test ./...`
- macOS 등의 권한 문제로 `~/Library/Caches/go-build`에 접근하지 못한다면 프로젝트 루트에 캐시 폴더를 만들고 아래처럼 실행하세요.
  ```bash
  mkdir -p .gocache
  GOCACHE=$(pwd)/.gocache go test ./...
  ```
- API 동작을 빠르게 확인하려면 `cmd/main.go`의 `addr` 값을 수정해 원하는 포트로 서버를 띄울 수 있습니다.
- 핸들러 로직을 수정할 때는 캐시 로직(`internal/core.go`)과 HTTP 계층을 명확히 분리해 유지보수성을 높여 주세요.
