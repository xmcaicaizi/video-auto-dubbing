# 閮ㄧ讲鎸囧崡

## Docker Compose 閮ㄧ讲

### 鍓嶇疆瑕佹眰

- Docker Engine 20.10+
- Docker Compose 2.0+
- 鑷冲皯 8GB 鍙敤鍐呭瓨
- 鑷冲皯 50GB 鍙敤纾佺洏绌洪棿

### 蹇€熷紑濮?

1. **鍏嬮殕椤圭洰骞惰繘鍏ョ洰褰?*
```bash
cd vedio
```

2. **閰嶇疆鐜鍙橀噺**

鏈粨搴撴彁渚?`env.example` 浣滀负鐜鍙橀噺绀轰緥锛堥儴鍒嗙幆澧冧細闄愬埗浣跨敤 dotfile锛屼緥濡?`.env.example`锛夈€?

濡傛灉浣犵殑鐜鏀寔 `.env` 鏂囦欢锛屽彲浠ュ皢绀轰緥澶嶅埗涓?`.env`锛?

```bash
cp env.example .env
```

缂栬緫 `.env` 鏂囦欢锛岃缃繀瑕佺殑閰嶇疆锛?

```env
# 鏁版嵁搴撻厤缃?
POSTGRES_DB=dubbing
POSTGRES_USER=dubbing
POSTGRES_PASSWORD=your_secure_password
POSTGRES_PORT=5432

# MinIO 閰嶇疆
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=your_secure_password
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001
MINIO_BUCKET=videos

# RabbitMQ 閰嶇疆
RABBITMQ_USER=rabbitmq
RABBITMQ_PASSWORD=your_secure_password
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672

# API 鏈嶅姟閰嶇疆
API_PORT=8080

# TTS 鏈嶅姟閰嶇疆
TTS_PORT=8000
TTS_BACKEND=index_tts2
INDEXTTS_MODEL_DIR=/app/models/IndexTTS-2
INDEXTTS_CFG_PATH=/app/models/IndexTTS-2/config.yaml
INDEXTTS_PROMPT_AUDIO=/app/assets/voice_01.wav
INDEXTTS_DEVICE=auto
INDEXTTS_USE_FP16=true
INDEXTTS_USE_TORCH_COMPILE=false
INDEXTTS_USE_CUDA_KERNEL=false
HF_ENDPOINT=https://hf-mirror.com
HF_HUB_CACHE=/app/models/IndexTTS-2/hf_cache
STRICT_DURATION=false  # true/false
MAX_CONCURRENT_REQUESTS=10  # max concurrent requests
MAX_RETRIES=3  # max retries
RETRY_DELAY_SECONDS=1.0  # retry delay in seconds

# 澶栭儴 API 閰嶇疆锛堟帹鑽愰€氳繃鍓嶇璁剧疆椤甸潰閰嶇疆锛岀幆澧冨彉閲忎綔涓哄悗澶囷級
# Moonshine ASR 鏈嶅姟閰嶇疆
| ASR Service | http://localhost:8002 | Moonshine ASR |
ASR_MODEL_ID=moonshine-base
ASR_DEVICE=cuda
ASR_COMPUTE_TYPE=float16
ASR_BACKEND=moonshine_onnx

# 鏅鸿氨 GLM 缈昏瘧 API 閰嶇疆
GLM_API_KEY=your_glm_api_key
GLM_API_URL=https://open.bigmodel.cn/api/paas/v4/chat/completions
GLM_MODEL=glm-4.5
GLM_RPS=5

# MinIO 鍏綉璁块棶鍦板潃锛堝彲閫夛紝濡傛灉 ASR 鏈嶅姟涓嶅湪鍚屼竴缃戠粶锛?# 渚嬪锛歮inio.example.com:9000 鎴栧弽鍚戜唬鐞嗗湴鍧€
MINIO_PUBLIC_ENDPOINT=

# 缃戝叧閰嶇疆
GATEWAY_HTTP_PORT=80
GATEWAY_HTTPS_PORT=443
```

3. **鍚姩鎵€鏈夋湇鍔?*

```bash
docker compose up -d
```

4. **鏌ョ湅鏈嶅姟鐘舵€?*

```bash
docker compose ps
```

5. **鏌ョ湅鏃ュ織**

```bash
# 鏌ョ湅鎵€鏈夋湇鍔℃棩蹇?
docker compose logs -f

# 鏌ョ湅鐗瑰畾鏈嶅姟鏃ュ織
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f tts_service
```

### 鏈嶅姟璁块棶鍦板潃

| 鏈嶅姟 | 鍦板潃 | 璇存槑 |
|------|------|------|
| API 鏈嶅姟 | http://localhost:8080 | 鍚庣 API |
| ASR Service | http://localhost:8002 | Moonshine ASR |
| TTS 鏈嶅姟 | http://localhost:8000 | TTS 鏈嶅姟 |
| MinIO 鎺у埗鍙?| http://localhost:9001 | 瀵硅薄瀛樺偍绠＄悊 |
| RabbitMQ 绠＄悊 | http://localhost:15672 | 娑堟伅闃熷垪绠＄悊 |
| 缃戝叧 | http://localhost:80 | 缁熶竴鍏ュ彛 |

**榛樿璐﹀彿瀵嗙爜**:
- MinIO: `minioadmin` / `minioadmin123` (闇€淇敼)
- RabbitMQ: `rabbitmq` / `rabbitmq123` (闇€淇敼)

### 鎵╁睍 Worker 瀹炰緥

**姘村钩鎵╁睍 worker 鏈嶅姟**:

```bash
# 鍚姩 3 涓?worker 瀹炰緥
docker compose up -d --scale worker=3

# 鏌ョ湅 worker 瀹炰緥
docker compose ps worker
```

**鍔ㄦ€佽皟鏁?worker 鏁伴噺**:

```bash
# 澧炲姞鍒?5 涓?
docker compose up -d --scale worker=5

# 鍑忓皯鍒?2 涓?
docker compose up -d --scale worker=2
```

**娉ㄦ剰浜嬮」**:
- Worker 鏄棤鐘舵€佺殑锛屽彲浠ュ畨鍏ㄥ湴鎵╁睍
- 姣忎釜 worker 浼氫粠鍚屼竴涓槦鍒楁秷璐逛换鍔?
- RabbitMQ 浼氳嚜鍔ㄨ繘琛岃礋杞藉潎琛?

### 鏈嶅姟渚濊禆鍏崇郴

```
gateway -> api -> db, minio, rabbitmq
gateway -> tts_service
worker -> db, minio, rabbitmq, tts_service
```

鍚姩椤哄簭鐢?`depends_on` 鍜?`healthcheck` 鑷姩绠＄悊銆?

### 鏁版嵁鎸佷箙鍖?

鎵€鏈夋暟鎹瓨鍌ㄥ湪 Docker volumes 涓細

| Volume | 鐢ㄩ€?| 浣嶇疆 |
|--------|------|------|
| `postgres_data` | PostgreSQL 鏁版嵁 | `/var/lib/postgresql/data` |
| `minio_data` | MinIO 瀵硅薄瀛樺偍 | `/data` |
| `rabbitmq_data` | RabbitMQ 鏁版嵁 | `/var/lib/rabbitmq` |
| `tts_temp` | TTS 涓存椂鏂囦欢 | `/app/temp` |
| `api_logs` | API 鏈嶅姟鏃ュ織 | `/app/logs` |
| `worker_logs` | Worker 鏈嶅姟鏃ュ織 | `/app/logs` |

**澶囦唤鏁版嵁**:

```bash
# 澶囦唤 PostgreSQL
docker compose exec db pg_dump -U dubbing dubbing > backup.sql

# 澶囦唤 MinIO锛堜娇鐢?mc 瀹㈡埛绔級
docker compose exec minio mc mirror /data /backup
```

### 鍋ュ悍妫€鏌?

鎵€鏈夋湇鍔￠兘閰嶇疆浜嗗仴搴锋鏌ワ細

```bash
# 妫€鏌ユ墍鏈夋湇鍔″仴搴风姸鎬?
docker compose ps

# 鎵嬪姩妫€鏌ユ湇鍔″仴搴?
curl http://localhost:8080/health
curl http://localhost:8000/health
```

### 鍋滄鍜屾竻鐞?

**鍋滄鏈嶅姟**:
```bash
docker compose stop
```

**鍋滄骞跺垹闄ゅ鍣?*:
```bash
docker compose down
```

**鍋滄骞跺垹闄ゅ鍣ㄣ€佺綉缁溿€乿olumes**:
```bash
docker compose down -v
```

鈿狅笍 **璀﹀憡**: `-v` 閫夐」浼氬垹闄ゆ墍鏈夋暟鎹紝鍖呮嫭鏁版嵁搴撳拰瀵硅薄瀛樺偍涓殑鏁版嵁锛?

### 鏇存柊鏈嶅姟

1. **鎷夊彇鏈€鏂颁唬鐮?*
```bash
git pull
```

2. **閲嶆柊鏋勫缓闀滃儚**
```bash
docker compose build
```

3. **閲嶅惎鏈嶅姟**
```bash
docker compose up -d
```

**闆跺仠鏈烘洿鏂?*锛堟帹鑽愶級:
```bash
# 鍏堝惎鍔ㄦ柊瀹瑰櫒
docker compose up -d --no-deps --build api

# 绛夊緟鏂板鍣ㄥ仴搴峰悗锛屽仠姝㈡棫瀹瑰櫒
docker compose stop api
docker compose rm -f api
docker compose up -d api
```

### 鐢熶骇鐜寤鸿

1. **浣跨敤 HTTPS**
   - 閰嶇疆 SSL 璇佷功
   - 淇敼 `gateway/nginx.conf` 鍚敤 HTTPS
   - 灏嗚瘉涔︽斁鍦?`gateway/ssl/` 鐩綍

2. **淇敼榛樿瀵嗙爜**
   - 鎵€鏈夋湇鍔＄殑榛樿瀵嗙爜蹇呴』淇敼
   - 浣跨敤寮哄瘑鐮佺瓥鐣?

3. **璧勬簮闄愬埗**
   - 鍦?`docker-compose.yml` 涓坊鍔犺祫婧愰檺鍒讹細
   ```yaml
   deploy:
     resources:
       limits:
         cpus: '2'
         memory: 4G
       reservations:
         cpus: '1'
         memory: 2G
   ```

4. **鏃ュ織绠＄悊**
   - 閰嶇疆鏃ュ織杞浆
   - 浣跨敤闆嗕腑寮忔棩蹇楁敹闆嗭紙濡?ELK銆丩oki锛?

5. **鐩戞帶鍛婅**
   - 閰嶇疆 Prometheus + Grafana
   - 璁剧疆鍏抽敭鎸囨爣鍛婅

6. **澶囦唤绛栫暐**
   - 瀹氭湡澶囦唤鏁版嵁搴?
   - 瀹氭湡澶囦唤瀵硅薄瀛樺偍
   - 娴嬭瘯鎭㈠娴佺▼

### 鏁呴殰鎺掓煡

**鏈嶅姟鏃犳硶鍚姩**:
```bash
# 鏌ョ湅璇︾粏鏃ュ織
docker compose logs service_name

# 妫€鏌ユ湇鍔″仴搴风姸鎬?
docker compose ps

# 妫€鏌ョ鍙ｅ崰鐢?
netstat -tulpn | grep :8080
```

**鏁版嵁搴撹繛鎺ュけ璐?*:
```bash
# 妫€鏌ユ暟鎹簱鏄惁杩愯
docker compose ps db

# 妫€鏌ユ暟鎹簱鏃ュ織
docker compose logs db

# 娴嬭瘯鏁版嵁搴撹繛鎺?
docker compose exec db psql -U dubbing -d dubbing
```

**闃熷垪娑堟伅鍫嗙Н**:
```bash
# 鏌ョ湅 RabbitMQ 绠＄悊鐣岄潰
# http://localhost:15672

# 妫€鏌ラ槦鍒楃姸鎬?
docker compose exec rabbitmq rabbitmqctl list_queues
```

**Worker 澶勭悊鎱?*:
```bash
# 澧炲姞 worker 瀹炰緥
docker compose up -d --scale worker=5

# 妫€鏌?worker 鏃ュ織
docker compose logs -f worker
```

### 镜像 / 长视频调优

- **硬件建议**：CPU ≥ 8 核、内存 ≥ 24GB，GPU 显存 ≥ 12GB；MinIO/磁盘预留 ≥ 200GB 以容纳长视频拆分音频、合成中间产物。
- **环境变量示例**（可运行 `make long-video-mode-env` 查看）：`TRANSLATE_BATCH_SIZE=30`、`TTS_BATCH_SIZE=30`、`TTS_MAX_CONCURRENCY=6`、`TTS_MAX_RETRIES=4`、`TTS_RETRY_DELAY_SECONDS=3.0`、`TIMEOUT_EXTRACT_AUDIO_SECONDS=1800`、`TIMEOUT_ASR_SECONDS=2400`、`TIMEOUT_TTS_SECONDS=2400`、`TIMEOUT_MUX_SECONDS=1800`。
- **补偿/定时任务**：启用 cron 或 CI 定时运行 `go run ./worker/cmd/tts_requeue`，扫描 `tts_audio_key` 为空的 segment 并重新投递 `task.tts`，避免单段失败阻塞 mux。
- **存储与带宽**：监控 MinIO bucket 空间，必要时开启生命周期策略清理历史 dub/中间件；确保 Worker、ASR、TTS 与 MinIO 在同一可用区降低传输时延。
- **FFmpeg 分段示例**：长视频可先按时间切片降低 ASR 超时风险，例如 `ffmpeg -ss 00:10:00 -to 00:20:00 -i input.mp4 -ac 1 -ar 16000 -y clip_10_20.wav`。

### 鎬ц兘璋冧紭

1. **鏁版嵁搴撲紭鍖?*
   - 璋冩暣 PostgreSQL 閰嶇疆
   - 娣诲姞閫傚綋鐨勭储寮?
   - 瀹氭湡 VACUUM

2. **闃熷垪浼樺寲**
   - 璋冩暣 RabbitMQ 棰勫彇鏁伴噺
   - 浼樺寲娑堟伅澶у皬
   - 浣跨敤娑堟伅鍘嬬缉

3. **瀵硅薄瀛樺偍浼樺寲**
   - 浣跨敤 CDN 鍔犻€?
   - 閰嶇疆鐢熷懡鍛ㄦ湡绛栫暐
   - 浼樺寲瀛樺偍妗剁粨鏋?

4. **TTS 鏈嶅姟浼樺寲**
   - 璋冩暣 `MAX_CONCURRENT_REQUESTS` 閬垮厤瑙﹀彂 API 闄愭祦
   - 鏍规嵁闇€姹傞€夋嫨 `STRICT_DURATION` 妯″紡锛堣川閲?vs 鏃堕暱绮剧‘搴︼級
   - 鐩戞帶 IndexTTS2 ?? 璋冪敤閰嶉鍜岄檺娴佹儏鍐?
   - 浼樺寲鎵瑰鐞嗗ぇ灏忥紙鍒嗘鍚堟垚锛?
