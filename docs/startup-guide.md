# 鍚姩鎸囧崡

鏈枃妗ｈ褰曡棰戞湰鍦板寲鑷姩閰嶉煶绯荤粺鐨勫惎鍔ㄦ祦绋嬪拰甯歌闂瑙ｅ喅鏂规銆?

## 蹇€熷惎鍔?

### 鍓嶇疆瑕佹眰

1. **Docker Desktop** 宸插畨瑁呭苟杩愯
   - Windows: 纭繚 `com.docker.service` 鏈嶅姟宸插惎鍔紙鍙兘闇€瑕佺鐞嗗憳鏉冮檺锛?
   - 楠岃瘉锛氳繍琛?`docker version` 搴旇兘鐪嬪埌 Server 淇℃伅

2. **鐜鍙橀噺閰嶇疆**锛堝彲閫夛紝鎺ㄨ崘閫氳繃鍓嶇璁剧疆锛?
| ASR Service | http://localhost:8002 | Moonshine ASR |
   - `GLM_API_URL`: GLM API 鍦板潃锛堥粯璁わ細https://open.bigmodel.cn/api/paas/v4/chat/completions锛?
   - `GLM_MODEL`: GLM 妯″瀷鍚嶏紙榛樿锛歡lm-4.5锛?
   - `MODELSCOPE_TOKEN`: ModelScope API Token
   - `MINIO_PUBLIC_ENDPOINT`: MinIO 鍏綉璁块棶鍦板潃锛堝鏋?ASR 鏈嶅姟涓嶅湪鍚屼竴缃戠粶锛?
### 鍚姩姝ラ

```bash
# 1. 杩涘叆椤圭洰鐩綍
cd vedio

# 2. 鍚姩鎵€鏈夋湇鍔?
docker compose up -d

# 3. 鏌ョ湅鏈嶅姟鐘舵€?
docker compose ps

# 4. 鏌ョ湅鏃ュ織锛堝彲閫夛級
docker compose logs -f
```

### 鏈嶅姟璁块棶鍦板潃

| 鏈嶅姟 | 鍦板潃 | 璇存槑 |
|------|------|------|
| 鍓嶇鐣岄潰 | http://localhost | 閫氳繃 NGINX 缃戝叧璁块棶 |
| API 鏈嶅姟 | http://localhost:8080 | 鍚庣 API |
| ASR Service | http://localhost:8002 | Moonshine ASR |
| TTS 鏈嶅姟 | http://localhost:8001 | 璇煶鍚堟垚鏈嶅姟锛堥粯璁?8001锛岄伩鍏嶄笌 8000 鍐茬獊锛?|
| MinIO 鎺у埗鍙?| http://localhost:9001 | 瀵硅薄瀛樺偍绠＄悊 |
| RabbitMQ 绠＄悊鍙?| http://localhost:15672 | 娑堟伅闃熷垪绠＄悊 |
| PostgreSQL | localhost:5432 | 鏁版嵁搴?|

## 甯歌闂

### 1. Docker Engine 鏈繍琛?

**鐥囩姸**锛?
```
error during connect: open //./pipe/docker_engine: The system cannot find the file specified
```

**瑙ｅ喅鏂规**锛?
1. 鍚姩 Docker Desktop 搴旂敤绋嬪簭
2. 浠ョ鐞嗗憳韬唤杩愯 PowerShell锛屾墽琛岋細
   ```powershell
   Start-Service com.docker.service
   ```
3. 楠岃瘉锛歚docker version` 搴旀樉绀?Server 淇℃伅

### 2. 绔彛鍐茬獊

**鐥囩姸**锛?
```
Bind for 0.0.0.0:8000 failed: port is already allocated
```

**瑙ｅ喅鏂规**锛?
- 妫€鏌ュ崰鐢ㄧ鍙ｇ殑瀹瑰櫒锛歚docker ps --format "table {{.Names}}\t{{.Ports}}"`
- 淇敼 `docker-compose.yml` 涓殑绔彛鏄犲皠锛屾垨鍋滄鍗犵敤绔彛鐨勫鍣?
- TTS 鏈嶅姟榛樿浣跨敤 8001 绔彛浠ラ伩鍏嶅啿绐?

### 3. 鏋勫缓澶辫触锛氱己灏?go.sum

**鐥囩姸**锛?
```
failed to calculate checksum: "/go.sum": not found
```

**瑙ｅ喅鏂规**锛?
- Dockerfile 宸蹭慨澶嶏細鍦ㄦ瀯寤洪樁娈佃繍琛?`go mod tidy` 鑷姩鐢熸垚 go.sum
- 濡傛灉鏈湴闇€瑕侊紝鍙繍琛岋細`cd api && go mod tidy` 鍜?`cd worker && go mod tidy`

### 4. 鏋勫缓澶辫触锛欸o 缂栬瘧閿欒

**鐥囩姸**锛?
```
"time" imported and not used
"os" imported and not used
```

**瑙ｅ喅鏂规**锛?
- 宸蹭慨澶嶏細绉婚櫎浜嗘墍鏈夋湭浣跨敤鐨?import
- 濡傛灉閬囧埌绫讳技闂锛岃繍琛?`go vet ./...` 妫€鏌?

### 5. TTS 鏈嶅姟鍚姩澶辫触锛氭壘涓嶅埌 uvicorn

**鐥囩姸**锛?
```
exec: "uvicorn": executable file not found in $PATH
```

**瑙ｅ喅鏂规**锛?
- Dockerfile 宸蹭慨澶嶏細浣跨敤缁濆璺緞 `/app/.venv/bin/uvicorn`
- 纭繚 `docker-compose.yml` 涓病鏈夎鐩?`/app` 鐨?bind mount锛堜細瑕嗙洊铏氭嫙鐜锛?

### 6. TTS 鍋ュ悍妫€鏌ュけ璐ワ細鎵句笉鍒?curl

**鐥囩姸**锛?
```
exec: "curl": executable file not found in $PATH
```

**瑙ｅ喅鏂规**锛?
- 宸蹭慨澶嶏細鍋ュ悍妫€鏌ユ敼涓轰娇鐢?Python `urllib`锛屼笉渚濊禆 curl
- 妫€鏌?`docker-compose.yml` 涓殑 healthcheck 閰嶇疆

### 7. TTS 鏈嶅姟鍚姩澶辫触锛氱己灏?MODELSCOPE_TOKEN

**鐥囩姸**锛?
```
MODELSCOPE_TOKEN is required but not set
```

**瑙ｅ喅鏂规**锛?
- 宸蹭慨澶嶏細TTS 鏈嶅姟鏀寔寤惰繜鍔犺浇锛屽惎鍔ㄦ椂涓嶅啀寮哄埗瑕佹眰 token
- 鍙互閫氳繃鍓嶇璁剧疆椤甸潰閰嶇疆 token锛屾垨璁剧疆鐜鍙橀噺 `MODELSCOPE_TOKEN`

### 8. Debian apt 婧愯繛鎺ュけ璐?

**鐥囩姸**锛?
```
Failed to fetch http://deb.debian.org/... Unable to connect
```

**瑙ｅ喅鏂规**锛?
- Dockerfile 宸蹭慨澶嶏細鑷姩灏?apt 婧愬垏鎹负 HTTPS
- 濡傛灉浠嶆湁闂锛屾鏌ョ綉缁滆繛鎺ユ垨浣跨敤浠ｇ悊

## 楠岃瘉鏈嶅姟杩愯

### 妫€鏌ュ鍣ㄧ姸鎬?

```bash
docker compose ps
```

鎵€鏈夋湇鍔″簲鏄剧ず `Up (healthy)` 鐘舵€併€?

### 妫€鏌ユ湇鍔℃棩蹇?

```bash
# 鏌ョ湅鎵€鏈夋湇鍔℃棩蹇?
docker compose logs -f

# 鏌ョ湅鐗瑰畾鏈嶅姟鏃ュ織
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f tts_service
```

### 娴嬭瘯 API 绔偣

```bash
# 娴嬭瘯 API 鍋ュ悍妫€鏌?
curl http://localhost:8080/health

# 娴嬭瘯 TTS 鍋ュ悍妫€鏌?
curl http://localhost:8001/health
```

## 鍋滄鏈嶅姟

```bash
# 鍋滄鎵€鏈夋湇鍔★紙淇濈暀鏁版嵁锛?
docker compose stop

# 鍋滄骞跺垹闄ゅ鍣紙淇濈暀鏁版嵁鍗凤級
docker compose down

# 鍋滄骞跺垹闄ゆ墍鏈夛紙鍖呮嫭鏁版嵁鍗凤級
docker compose down -v
```

## 閲嶅惎鏈嶅姟

```bash
# 閲嶅惎鎵€鏈夋湇鍔?
docker compose restart

# 閲嶅惎鐗瑰畾鏈嶅姟
docker compose restart api
docker compose restart worker
```

## 鏇存柊浠ｇ爜鍚庨噸鏂版瀯寤?

```bash
# 閲嶆柊鏋勫缓骞跺惎鍔?
docker compose up -d --build

# 浠呴噸鏂版瀯寤虹壒瀹氭湇鍔?
docker compose up -d --build api
docker compose up -d --build worker
docker compose up -d --build tts_service
```

## 寮€鍙戞ā寮?

濡傛灉闇€瑕佷慨鏀逛唬鐮佸苟瀹炴椂鐢熸晥锛屽彲浠ワ細

1. **API/Worker锛圙o锛?*锛?
   - 淇敼浠ｇ爜鍚庨噸鏂版瀯寤猴細`docker compose up -d --build api`
   - 鎴栦娇鐢?bind mount锛堜笉鎺ㄨ崘锛屽彲鑳借鐩栨瀯寤轰骇鐗╋級

2. **TTS锛圥ython锛?*锛?
   - 淇敼浠ｇ爜鍚庨噸鏂版瀯寤猴細`docker compose up -d --build tts_service`
   - 鎴栦娇鐢?bind mount锛堟敞鎰忎笉瑕佽鐩?`.venv`锛?

## 鏁呴殰鎺掓煡

### 鏌ョ湅瀹瑰櫒璇︾粏淇℃伅

```bash
docker inspect video-dubbing-api
docker inspect video-dubbing-worker
docker inspect video-dubbing-tts
```

### 杩涘叆瀹瑰櫒璋冭瘯

```bash
# 杩涘叆 API 瀹瑰櫒
docker exec -it video-dubbing-api sh

# 杩涘叆 Worker 瀹瑰櫒
docker exec -it video-dubbing-worker sh

# 杩涘叆 TTS 瀹瑰櫒
docker exec -it video-dubbing-tts bash
```

### 妫€鏌ョ綉缁滆繛鎺?

```bash
# 鏌ョ湅 Docker 缃戠粶
docker network ls
docker network inspect vedio_dubbing-network
```

### 妫€鏌ユ暟鎹嵎

```bash
# 鏌ョ湅鏁版嵁鍗?
docker volume ls
docker volume inspect vedio_postgres_data
docker volume inspect vedio_minio_data
```

## 鎬ц兘浼樺寲

### 鎵╁睍 Worker 瀹炰緥

```bash
# 鍚姩 3 涓?worker 瀹炰緥
docker compose up -d --scale worker=3
```

### 璧勬簮闄愬埗

鍦?`docker-compose.yml` 涓负鏈嶅姟娣诲姞璧勬簮闄愬埗锛?

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

## 瀹夊叏寤鸿

1. **鐢熶骇鐜**锛?
   - 淇敼鎵€鏈夐粯璁ゅ瘑鐮?
   - 浣跨敤鐜鍙橀噺鏂囦欢锛坄.env`锛夌鐞嗘晱鎰熶俊鎭?
   - 鍚敤 HTTPS锛堥厤缃?SSL 璇佷功锛?
   - 闄愬埗绔彛鏆撮湶锛堜粎鏆撮湶蹇呰鐨勭鍙ｏ級

2. **API Key 绠＄悊**锛?
   - 褰撳墠 MVP 闃舵锛欰PI Key 瀛樺偍鍦ㄦ暟鎹簱锛堟槑鏂囷級
   - 鐢熶骇鐜锛氬簲鍔犲瘑瀛樺偍鎴栦娇鐢ㄥ瘑閽ョ鐞嗘湇鍔?

3. **缃戠粶闅旂**锛?
   - 浣跨敤 Docker 缃戠粶闅旂鏈嶅姟
   - 浠呯綉鍏虫湇鍔℃毚闇插埌鍏綉
   - 鍐呴儴鏈嶅姟閫氳繃鏈嶅姟鍚嶉€氫俊

## 鐩稿叧鏂囨。

- [閮ㄧ讲鎸囧崡](deployment.md)
- [鏋舵瀯鏂囨。](architecture.md)
- [API 鎺ュ彛鏂囨。](api-contracts.md)
- [ASR 服务文档](asr-service.md)
- [TTS 鏈嶅姟鏂囨。](tts-service.md)

