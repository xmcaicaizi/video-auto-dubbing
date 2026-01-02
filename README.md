# 瑙嗛鏈湴鍖栬嚜鍔ㄩ厤闊崇郴缁?

闈㈠悜瑙嗛鏈湴鍖栫殑鑷姩閰嶉煶绯荤粺锛岄€氳繃缁撳悎鍏堣繘鐨勮闊宠瘑鍒€佹満鍣ㄧ炕璇戜笌鍙帶璇煶鍚堟垚鎶€鏈紝瀹炵幇瑙嗛涓鐧界殑鑷姩璇嗗埆銆佽法璇█缈昏瘧鍜岀洰鏍囪绉嶉厤闊炽€?

## 绯荤粺鏋舵瀯

鏈郴缁熼噰鐢ㄥ井鏈嶅姟 + 寮傛浠诲姟闃熷垪鏂瑰紡鏋勫缓锛屽皢璇煶璇嗗埆銆佺炕璇戙€佸彲鎺?TTS 鍒嗗竷鎴愮嫭绔嬫ā鍧楋紝骞剁粨鍚堝璞″瓨鍌ㄤ笌鏁版嵁搴撶粺涓€绠＄悊璧勬簮锛屼粠鑰屽疄鐜伴珮骞跺彂銆佹槗鎵╁睍銆佸彲闈犲彲閮ㄧ讲鐨勮棰戞湰鍦板寲鑷姩閰嶉煶骞冲彴銆?

### 鏍稿績鐗规€?

- 鉁?**ASR 璇煶璇嗗埆**: 浣跨敤 Moonshine 寮€婧愭ā鍨嬭繘琛屽甫鏃堕棿鎴崇殑璇煶璇嗗埆
- 鉁?**鏈哄櫒缈昏瘧**: 鍒╃敤 GLM 妯″瀷杩涜璺ㄨ瑷€缈昏瘧
- 鉁?**鍙帶 TTS**: 閫氳繃榄旀惌(ModelScope)绀惧尯鐨?IndexTTS-2 API 瀹炵幇鍙楁椂闂磋酱绾︽潫鐨勫彲鎺ц闊冲悎鎴?
- 鉁?**闊崇敾鍚屾**: 閫氳繃鏃堕棿杞寸害鏉熻В鍐宠法璇█閰嶉煶鏃堕煶鐢讳笉鍚屾鐨勯棶棰?
- 鉁?**寰湇鍔℃灦鏋?*: Go 璇█鍚庣锛屾ā鍧楄В鑰︼紝鏄撲簬鎵╁睍
- 鉁?**寮傛澶勭悊**: RabbitMQ 浠诲姟闃熷垪锛屾敮鎸侀珮骞跺彂鍜屼换鍔￠噸璇?
- 鉁?**瀹瑰櫒鍖栭儴缃?*: Docker Compose 涓€閿儴缃诧紝鏀寔妯悜鎵╁睍

## 鎶€鏈爤

### 鍚庣鏈嶅姟
- **璇█**: Go 1.21+
- **妗嗘灦**: Gin/Echo
- **鏁版嵁搴?*: PostgreSQL 15
- **娑堟伅闃熷垪**: RabbitMQ
- **瀵硅薄瀛樺偍**: MinIO (S3 鍏煎)

### TTS 鏈嶅姟
- **璇█**: Python 3.11+
- **妗嗘灦**: FastAPI
- **渚濊禆绠＄悊**: uv
- **妯″瀷**: IndexTTS-2锛堥€氳繃 ModelScope API 璋冪敤锛?

### 鍩虹璁炬柦
- **瀹瑰櫒鍖?*: Docker + Docker Compose
- **缃戝叧**: NGINX
- **鍓嶇**: (寰呭疄鐜?

## 蹇€熷紑濮?

### 鍓嶇疆瑕佹眰

- Docker Engine 20.10+
- Docker Compose 2.0+
- 鑷冲皯 8GB 鍙敤鍐呭瓨
- 鑷冲皯 50GB 鍙敤纾佺洏绌洪棿

### 閮ㄧ讲姝ラ

1. **鍏嬮殕椤圭洰**
```bash
git clone <repository-url>
cd vedio
```

2. **閰嶇疆鐜鍙橀噺**
```bash
# 鍒涘缓鐜鍙橀噺鏂囦欢锛堝鏋滀笉瀛樺湪锛?
# 璁剧疆蹇呰鐨勯厤缃紝鐗瑰埆鏄閮?API 瀵嗛挜锛?
# - GLM_API_KEY: 鏅鸿氨 GLM 缈昏瘧 API Key锛堟帹鑽愰€氳繃鍓嶇璁剧疆锛?# - GLM_API_URL: GLM API 鍦板潃锛堥粯璁?https://open.bigmodel.cn/api/paas/v4/chat/completions锛?# - GLM_MODEL: GLM 妯″瀷鍚嶏紙榛樿 glm-4.5锛?# - MODELSCOPE_TOKEN: ModelScope API 璁块棶浠ょ墝锛堟帹鑽愰€氳繃鍓嶇璁剧疆锛?
# - ASR_SERVICE_URL: Moonshine ASR 服务地址（默认 http://localhost:8002）
# - ASR_MODEL_ID: Moonshine 模型 ID（默认 moonshine-base）
# - ASR_DEVICE: 运行设备（默认 cuda）
# - ASR_COMPUTE_TYPE: 推理精度（默认 float16）
# - ASR_BACKEND: ASR 后端（默认 moonshine）
# - MINIO_PUBLIC_ENDPOINT: MinIO 外部可访问地址（ASR 服务不在同一网络时使用）
```

**閲嶈**: 
- **鎺ㄨ崘鏂瑰紡**锛氬湪鍓嶇"璁剧疆"椤甸潰杈撳叆 API Key锛岀郴缁熶細闅忎换鍔′繚瀛樺苟浣跨敤銆傜幆澧冨彉閲忎粎浣滀负鍚庡銆?
- API Key 鑾峰彇鏂瑰紡锛?
  - Moonshine ASR: 浣跨敤鏈湴 Python 鏈嶅姟锛屾棤闇€澶栭儴 API Key
  - 鏅鸿氨 GLM: 鐧诲綍 [鏅鸿氨 AI 寮€鏀惧钩鍙癩(https://open.bigmodel.cn/) 鑾峰彇 API Key
  - ModelScope: 鐧诲綍 [ModelScope 瀹樼綉](https://modelscope.cn)锛屽湪涓汉璁剧疆涓敓鎴?API Token
- **MINIO_PUBLIC_ENDPOINT**: 若 ASR 服务不在同一网络，可配置可访问的 MinIO 地址（如 minio.example.com:9000）

> 璇存槑锛氭湰浠撳簱鎻愪緵 `env.example` 浣滀负鐜鍙橀噺绀轰緥锛堥儴鍒嗙幆澧冧細闄愬埗浣跨敤 dotfile锛屼緥濡?`.env.example`锛夈€備綘鍙互灏?`env.example` 澶嶅埗涓?`.env`锛堝鏋滅幆澧冩敮鎸侊級锛屾垨鎸夐渶瀵煎叆涓虹郴缁熺幆澧冨彉閲忋€?

3. **鍚姩鏈嶅姟**
```bash
docker compose up -d
```

4. **鏌ョ湅鏈嶅姟鐘舵€?*
```bash
docker compose ps
```

5. **鏌ョ湅鏃ュ織**
```bash
docker compose logs -f
```

### 鏈嶅姟璁块棶

- **API 鏈嶅姟**: http://localhost:8080
- **TTS 鏈嶅姟**: http://localhost:8000
- **MinIO 鎺у埗鍙?*: http://localhost:9001
- **RabbitMQ 绠＄悊**: http://localhost:15672
- **缃戝叧**: http://localhost:80

璇︾粏閮ㄧ讲璇存槑璇峰弬鑰?[閮ㄧ讲鎸囧崡](docs/deployment.md)

## 椤圭洰缁撴瀯

```
vedio/
鈹溾攢鈹€ api/                    # API 鏈嶅姟锛圙o锛?
鈹溾攢鈹€ worker/                  # Worker 鏈嶅姟锛圙o锛?
鈹溾攢鈹€ tts_service/            # TTS 鏈嶅姟锛圥ython + uv锛?
├── asr_service/            # ASR service (Python + Moonshine)
鈹溾攢鈹€ gateway/                # NGINX 缃戝叧閰嶇疆
鈹溾攢鈹€ web/                    # 鍓嶇锛堝緟瀹炵幇锛?
鈹溾攢鈹€ docs/                   # 鏂囨。
鈹?  鈹溾攢鈹€ architecture.md    # 鏋舵瀯璁捐
鈹?  鈹溾攢鈹€ data-model.md      # 鏁版嵁妯″瀷
鈹?  鈹溾攢鈹€ api-contracts.md   # API 鎺ュ彛瑙勮寖
鈹?  鈹溾攢鈹€ queue-messages.md  # 闃熷垪娑堟伅瑙勮寖
鈹?  鈹溾攢鈹€ tts-service.md     # TTS 鏈嶅姟瑙勮寖
鈹?  鈹溾攢鈹€ coding-standards.md # 浠ｇ爜瑙勮寖
鈹?  鈹溾攢鈹€ code-review.md     # 浠ｇ爜瀹℃煡瑙勮寖
鈹?  鈹溾攢鈹€ commit-message.md  # 鎻愪氦淇℃伅瑙勮寖
鈹?  鈹斺攢鈹€ deployment.md      # 閮ㄧ讲鎸囧崡
鈹溾攢鈹€ .github/workflows/     # GitHub Actions CI 閰嶇疆
鈹溾攢鈹€ Makefile              # 缁熶竴寮€鍙戝懡浠ゅ叆鍙?
鈹溾攢鈹€ .golangci.yml         # Go 闈欐€佹鏌ラ厤缃?
鈹溾攢鈹€ .editorconfig         # 缂栬緫鍣ㄩ厤缃?
鈹斺攢鈹€ CONTRIBUTING.md       # 璐＄尞鎸囧崡
鈹溾攢鈹€ docker-compose.yml     # Docker Compose 閰嶇疆
鈹溾攢鈹€ .env.example           # 鐜鍙橀噺绀轰緥
鈹斺攢鈹€ README.md             # 鏈枃浠?
```

## 鏍稿績娴佺▼

1. **涓婁紶瑙嗛**: 鐢ㄦ埛閫氳繃 API 涓婁紶瑙嗛鏂囦欢
2. **鎻愬彇闊抽**: Worker 浣跨敤 ffmpeg 浠庤棰戞彁鍙栭煶棰?
3. **璇煶璇嗗埆**: 璋冪敤 Moonshine ASR 鏈嶅姟锛岃幏鍙栧甫鏃堕棿鎴崇殑璇嗗埆缁撴灉
4. **鏈哄櫒缈昏瘧**: 璋冪敤 GLM API锛屽皢璇嗗埆鏂囨湰缈昏瘧涓虹洰鏍囪瑷€
5. **璇煶鍚堟垚**: 璋冪敤 TTS 鏈嶅姟锛屾牴鎹椂闂磋酱绾︽潫鍚堟垚閰嶉煶闊抽
6. **瑙嗛鍚堟垚**: 浣跨敤 ffmpeg 灏嗛厤闊抽煶棰戞浛鎹㈠師瑙嗛闊抽
7. **涓嬭浇缁撴灉**: 鐢ㄦ埛涓嬭浇澶勭悊瀹屾垚鐨勮棰?

璇︾粏娴佺▼璇峰弬鑰?[鏋舵瀯璁捐鏂囨。](docs/architecture.md)

## API 鏂囨。

### 鍒涘缓浠诲姟
```bash
POST /api/v1/tasks
Content-Type: multipart/form-data

video: <file>
source_language: zh
target_language: en
```

### 鏌ヨ浠诲姟鐘舵€?
```bash
GET /api/v1/tasks/:task_id
```

### 鑾峰彇浠诲姟缁撴灉
```bash
GET /api/v1/tasks/:task_id/result
```

璇︾粏 API 鏂囨。璇峰弬鑰?[API 鎺ュ彛瑙勮寖](docs/api-contracts.md)

## 鎵╁睍 Worker

绯荤粺鏀寔姘村钩鎵╁睍 worker 瀹炰緥锛?

```bash
# 鍚姩 3 涓?worker 瀹炰緥
docker compose up -d --scale worker=3
```

## 寮€鍙戞寚鍗?

### 浠ｇ爜瑙勮寖

鏈」鐩伒寰?Google Style Guides 鍜?Engineering Practices 鐨勬渶浣冲疄璺碉細

- 馃摉 [浠ｇ爜瑙勮寖](docs/coding-standards.md) - Go/Python/Shell 缂栫爜瑙勮寖
- 馃攳 [浠ｇ爜瀹℃煡瑙勮寖](docs/code-review.md) - 浠ｇ爜瀹℃煡娴佺▼鍜屾鏌ユ竻鍗?
- 馃挰 [鎻愪氦淇℃伅瑙勮寖](docs/commit-message.md) - Git 鎻愪氦淇℃伅鏍煎紡瑕佹眰
- 馃 [璐＄尞鎸囧崡](CONTRIBUTING.md) - 瀹屾暣鐨勫紑鍙戝伐浣滄祦鍜岃础鐚祦绋?

### 浠ｇ爜妫€鏌?

鎻愪氦浠ｇ爜鍓嶏紝**蹇呴』**杩愯浠ヤ笅妫€鏌ワ細

```bash
# 浣跨敤 Makefile锛堟帹鑽愶級
make format    # 鏍煎紡鍖栨墍鏈変唬鐮?
make lint      # 杩愯鎵€鏈夐潤鎬佹鏌?
make test      # 杩愯鎵€鏈夋祴璇?
make ci        # 杩愯瀹屾暣 CI 妫€鏌ワ紙format + lint + test锛?

# 鏌ョ湅鎵€鏈夊彲鐢ㄥ懡浠?
make help
```

**CI 鐘舵€侊細** 鎵€鏈変唬鐮佹彁浜ら兘浼氶€氳繃 [GitHub Actions](.github/workflows/ci.yml) 鑷姩妫€鏌ワ紝鍖呮嫭锛?
- 鉁?Go 浠ｇ爜鏍煎紡鍖栧拰闈欐€佹鏌ワ紙gofmt, goimports, golangci-lint锛?
- 鉁?Python 浠ｇ爜鏍煎紡鍖栧拰闈欐€佹鏌ワ紙black, ruff锛?
- 鉁?Shell 鑴氭湰妫€鏌ワ紙shfmt, shellcheck锛?
- 鉁?鎵€鏈夋祴璇曞繀椤婚€氳繃

### Go 鏈嶅姟寮€鍙?

```bash
cd api  # 鎴?worker
go mod download
go run main.go
```

### Python TTS 鏈嶅姟寮€鍙?

```bash
cd tts_service
uv venv
source .venv/bin/activate  # Windows: .venv\Scripts\activate
uv sync
uv run uvicorn app.main:app --host 0.0.0.0 --port 8000
```

## 鏂囨。绱㈠紩

### 绯荤粺鏂囨。
- [鏋舵瀯璁捐](docs/architecture.md) - 绯荤粺鏋舵瀯鍜屾湇鍔″垝鍒?
- [鏁版嵁妯″瀷](docs/data-model.md) - 鏁版嵁搴撹〃鍜屽璞″瓨鍌ㄨ璁?
- [API 鎺ュ彛瑙勮寖](docs/api-contracts.md) - REST API 璇︾粏璇存槑
- [闃熷垪娑堟伅瑙勮寖](docs/queue-messages.md) - RabbitMQ 娑堟伅鏍煎紡鍜岄噸璇曠瓥鐣?- [ASR 鏈嶅姟瑙勮寖](docs/asr-service.md) - Moonshine ASR 鏈嶅姟鎺ュ彛鍜岄厤缃?- [TTS 鏈嶅姟瑙勮寖](docs/tts-service.md) - TTS 鏈嶅姟鎺ュ彛鍜岄厤缃?- [閮ㄧ讲鎸囧崡](docs/deployment.md) - 璇︾粏閮ㄧ讲璇存槑鍜屾晠闅滄帓鏌?

### 寮€鍙戣鑼?
- [浠ｇ爜瑙勮寖](docs/coding-standards.md) - Go/Python/Shell 缂栫爜瑙勮寖鍜屾渶浣冲疄璺?
- [浠ｇ爜瀹℃煡瑙勮寖](docs/code-review.md) - 浠ｇ爜瀹℃煡娴佺▼鍜屾鏌ユ竻鍗?
- [鎻愪氦淇℃伅瑙勮寖](docs/commit-message.md) - Git 鎻愪氦淇℃伅鏍煎紡瑕佹眰
- [璐＄尞鎸囧崡](CONTRIBUTING.md) - 瀹屾暣鐨勫紑鍙戝伐浣滄祦鍜岃础鐚祦绋?

## 璐＄尞鎸囧崡

鎴戜滑娆㈣繋鎵€鏈夊舰寮忕殑璐＄尞锛佽鍏堥槄璇讳互涓嬫枃妗ｏ細

1. 馃摉 [璐＄尞鎸囧崡](CONTRIBUTING.md) - 璇︾粏鐨勫紑鍙戝伐浣滄祦
2. 馃摑 [浠ｇ爜瑙勮寖](docs/coding-standards.md) - 缂栫爜瑙勮寖鍜屾渶浣冲疄璺?
3. 馃攳 [浠ｇ爜瀹℃煡瑙勮寖](docs/code-review.md) - 浠ｇ爜瀹℃煡娴佺▼
4. 馃挰 [鎻愪氦淇℃伅瑙勮寖](docs/commit-message.md) - 鎻愪氦淇℃伅鏍煎紡

**蹇€熷紑濮嬶細**

1. Fork 鏈」鐩?
2. 鍒涘缓鐗规€у垎鏀?(`git checkout -b feature/AmazingFeature`)
3. 寮€鍙戜唬鐮佸苟杩愯妫€鏌?(`make ci`)
4. 鎻愪氦鏇存敼锛堥伒寰彁浜や俊鎭鑼冿級
5. 鎺ㄩ€佸埌鍒嗘敮骞跺紑鍚?Pull Request

**閲嶈锛?* 鎵€鏈変唬鐮佸繀椤婚€氳繃 CI 妫€鏌ユ墠鑳藉悎骞讹紒

## 璁稿彲璇?

鏈」鐩噰鐢?MIT 璁稿彲璇併€?

## 鑱旂郴鏂瑰紡

濡傛湁闂鎴栧缓璁紝璇锋彁浜?Issue 鎴栬仈绯婚」鐩淮鎶よ€呫€?




## IndexTTS2 ??

- ??? TTS ????? IndexTTS2 ?????????????? `tts_service/indextts`?
- `_vendor/` ????????????? Git?
