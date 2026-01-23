# 快速参数修复 - 替换api_server_v2.py文件最后的main部分

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", type=str, default="0.0.0.0")
    parser.add_argument("--port", type=int, default=6006)
    parser.add_argument("--model_dir", type=str, default="checkpoints/IndexTTS-2-vLLM", help="Model checkpoints directory")
    parser.add_argument("--is_fp16", action="store_true", default=False, help="Fp16 infer")  # 添加这一行
    parser.add_argument("--gpu_memory_utilization", type=float, default=0.25)
    parser.add_argument("--qwenemo_gpu_memory_utilization", type=float, default=0.10)  # 添加这一行
    parser.add_argument("--verbose", action="store_true", default=False, help="Enable verbose mode")  # 添加这一行
    args = parser.parse_args()

    if not os.path.exists("outputs"):
        os.makedirs("outputs")

    # 确保日志目录存在
    if not os.path.exists("logs"):
        os.makedirs("logs")

    logger.info(f"Starting IndexTTS VLLM server with voice cloning")
    logger.info(f"Model dir: {args.model_dir}")
    logger.info(f"Upload dir: {UPLOAD_DIR}")
    logger.info(f"Server: http://{args.host}:{args.port}")

    uvicorn.run(app=app, host=args.host, port=args.port)