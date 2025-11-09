# go-cache-server-mini

## 개요
`go-cache-server-mini`는 Gin 기반의 HTTP API를 통해 키-값 쌍을 저장·조회할 수 있는 초경량 인메모리 캐시 서버입니다. 학습용 혹은 간단한 PoC(Service mock) 환경에서 빠르게 상태 저장이 필요할 때 사용할 수 있도록 최소한의 러닝 커브로 설계되었습니다.

## 주요 기능
- `/set` 엔드포인트로 JSON 페이로드를 받아 캐시에 저장
- `/get` 엔드포인트로 키에 해당하는 값을 즉시 반환
- `sync.Map` 기반의 경쟁 안전한 저장소와 명확한 오류 응답(`ErrBadRequest`, `ErrNotFound` 등)

## 프로젝트 구조
```
cmd/main.go          # 서버 엔트리포인트, 포트 설정
internal/api.go      # Gin 라우터 및 HTTP 핸들러 정의
internal/core.go     # Cache 타입과 Set/Get 구현
internal/errors.go   # 공용 에러 정의
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
- 테스트 실행: `go test ./...`
- API 동작을 빠르게 확인하려면 `cmd/main.go`의 `addr` 값을 수정해 원하는 포트로 서버를 띄울 수 있습니다.
- 핸들러 로직을 수정할 때는 캐시 로직(`internal/core.go`)과 HTTP 계층을 명확히 분리해 유지보수성을 높여 주세요.
