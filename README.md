# go-cache-server-mini

## 개요
`go-cache-server-mini`는 Gin 기반 HTTP API로 키-값을 읽고 쓰는 초경량 인메모리 캐시 서버입니다. 학습·실험·간단한 통합 테스트에서 상태를 빠르게 저장할 수 있도록 설계되었으며, TTL·숫자 연산·벌크 작업 등 Redis에서 자주 쓰는 최소 기능을 제공합니다.

## 주요 기능
- **확장된 엔드포인트**: 단건(`set`, `get`, `del`)뿐 아니라 `setnx`, `getset`, `mget`, `mset`과 같은 멱등·벌크 연산까지 제공해 테스트 시나리오를 유연하게 구성할 수 있습니다.
- **TTL & 영구 키**: TTL을 생략하면 기본 TTL을 사용하고, 음수를 넣으면 `persist` 상태(-1 TTL)로 저장됩니다. 만료 워커가 1초 간격으로 캐시를 스캔합니다.
- **숫자 연산**: `incr`, `decr`가 문자열로 저장된 정수 값을 원자적으로 갱신합니다.
- **동시성 안전**: RWMutex로 보호된 맵과 중앙 집중 에러(`internal/errors.go`)를 사용해 단순하면서도 예측 가능한 동작을 유지합니다.
- **Graceful shutdown**: `cmd/main.go`가 SIGINT/SIGTERM을 받아 API 서버와 만료 워커를 순차 종료합니다.

## API 한눈에 보기
| Method | Path | Body / Query | 설명 |
| --- | --- | --- | --- |
| GET | `/ping` | - | Liveness/Health 체크 |
| POST | `/set` | `{"key","value","ttl?"}` | 값을 저장, TTL은 초 단위 |
| GET | `/get` | `?key=` | 값을 JSON 그대로 반환 |
| DELETE | `/del` | `?key=` | 키 삭제 |
| GET | `/exists` | `?key=` | 존재 여부(boolean) |
| GET | `/keys` | - | 현재 키 목록 |
| POST | `/expire` | `{"key","ttl"}` | TTL 재설정, 0 이하이면 삭제 |
| GET | `/ttl` | `?key=` | 남은 TTL(초). 영구 키는 -1 |
| POST | `/persist` | `?key=` | 만료 시간을 제거 |
| POST | `/flush` | - | 모든 키 제거 |
| POST | `/incr` | `?key=` | 정수 값 +1 후 값 반환 |
| POST | `/decr` | `?key=` | 정수 값 -1 후 값 반환 |
| POST | `/setnx` | `{"key","value","ttl?"}` | 키가 없을 때만 저장 |
| POST | `/getset` | `{"key","value"}` | 새 값으로 교체하고 이전 값을 반환 |
| POST | `/mget` | `{"keys":[]}` | 여러 키를 한 번에 조회 |
| POST | `/mset` | `{"kv":{},"ttl?"}` | 여러 키를 동일 TTL로 저장 |

### TTL 규칙
1. `ttl`이 0이거나 누락되면 `config.yml`의 `ttl.default`를 사용합니다.
2. `ttl`이 `ttl.max`를 넘으면 자동으로 잘립니다.
3. `ttl`에 음수를 주면 `persist` 상태(-1)로 저장되며 만료 워커의 대상에서 제외됩니다.

### 요청 예시
```bash
# 1시간 TTL로 저장
curl -X POST http://localhost:8080/set \
  -H "Content-Type: application/json" \
  -d '{"key":"greeting","value":"\"hello\"","ttl":3600}'

# 벌크 저장 후 조회
curl -X POST http://localhost:8080/mset \
  -H "Content-Type: application/json" \
  -d '{"kv":{"foo":1,"bar":2}}'
curl -X POST http://localhost:8080/mget \
  -H "Content-Type: application/json" \
  -d '{"keys":["foo","bar"]}'

# 숫자 연산
curl -X POST "http://localhost:8080/incr?key=counter"
```
`value`는 `json.RawMessage`로 저장되므로 문자열, 객체, 숫자 등 어떤 JSON 타입도 변형 없이 round-trip 됩니다.

## 프로젝트 구조
```
cmd/main.go                  # 엔트리포인트, 시그널 처리, graceful shutdown
internal/api/api.go          # Gin 서버 부트스트랩 및 라우트 매핑
internal/api/handler/*.go    # 각 HTTP 엔드포인트 구현
internal/api/dto/*.go        # 요청/응답 DTO
internal/core/core.go        # 캐시 구현, TTL/만료 워커, 숫자/벌크 연산
internal/core/cache_interface.go
internal/util/convert.go     # TTL 정규화, int<->[]byte 변환
internal/config/config.go    # YAML 설정 로더 (env 확장 지원)
internal/errors.go           # 공용 에러 정의
config.yml                   # 기본 TTL, HTTP 바인딩 등 런타임 설정
```

## 빠른 시작
1. Go 1.24.5 이상을 설치합니다.
2. 서버 실행:
   ```bash
   go run ./cmd
   ```
3. 배포용 바이너리 생성:
   ```bash
   go build -o bin/cache-server ./cmd
   ```

## 설정
`config.yml`을 수정하거나 `${PORT}`처럼 환경 변수를 넣어두면 런타임에 `os.ExpandEnv`로 치환됩니다.

```yaml
persistent:
  type: memory        # 추후 외부 스토리지 추가 예정
ttl:
  default: 86400      # TTL 미지정 시 1일
  max: 604800         # TTL 상한 7일
http:
  enabled: true
  address: ":8080"
```

## Graceful shutdown & 오류 전파
- `cmd/main.go`가 SIGINT/SIGTERM을 수신하면 컨텍스트를 취소하고 API 서버 · TTL 워커를 기다린 후 종료합니다.
- `api.StartAPIServer`가 포트를 잡지 못하면 즉시 에러를 반환하고, 메인은 에러 로그를 남긴 뒤 종료 코드 1로 프로세스를 종료합니다.

## 개발 및 테스트
- 단위 테스트: `go test ./...`
- macOS 등에서 `$HOME/Library/Caches/go-build` 접근이 막히면:
  ```bash
  mkdir -p .gocache
  GOCACHE=$(pwd)/.gocache go test ./...
  ```
- `internal/api/handler/handler_test.go`가 엔드포인트 대부분을 커버하므로 신규 기능 추가 시 여기에 케이스를 확장해주세요.
- 여러 인스턴스를 띄워야 한다면 `config.yml`의 `http.address`를 바꾸거나 환경 변수를 주입해 포트를 조정하세요.
