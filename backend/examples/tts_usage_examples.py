"""
TTS 客户端使用示例
演示系统音色和声音复刻两种模式
"""

import os
from pathlib import Path

from app.integrations.dashscope import TTSClient


def example_1_system_voice():
    """示例 1: 使用系统音色（cosyvoice-v1）"""
    print("\n" + "=" * 60)
    print("示例 1: 系统音色模式（cosyvoice-v1）")
    print("=" * 60)

    # 初始化客户端
    client = TTSClient(
        model="cosyvoice-v1",
        voice="longxiaochun",  # 龙小春（女声）
        format="mp3",
    )

    # 查看可用音色
    voices = client.list_voices()
    print(f"\n可用音色: {', '.join(voices)}")

    # 合成语音
    text = "你好，我是通义千问的语音合成系统。"
    print(f"\n正在合成: {text}")

    audio = client.synthesize(text)

    # 保存音频
    output_file = "output_system_voice.mp3"
    with open(output_file, "wb") as f:
        f.write(audio)

    print(f"✅ 合成成功，已保存为: {output_file}")
    print(f"   音频大小: {len(audio):,} bytes")


def example_2_voice_cloning_manual():
    """示例 2: 声音复刻 - 手动注册音色"""
    print("\n" + "=" * 60)
    print("示例 2: 声音复刻模式 - 手动注册")
    print("=" * 60)

    # 初始化客户端（声音复刻模型）
    client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")

    # Step 1: 上传样本音频并复刻音色
    sample_audio = "sample_voice.wav"  # 10-20 秒的音频样本

    if not Path(sample_audio).exists():
        print(f"⚠️  样本音频不存在: {sample_audio}")
        print("   请准备一个 10-20 秒的音频文件（wav/mp3）")
        return

    print(f"\nStep 1: 复刻音色...")
    print(f"   样本音频: {sample_audio}")

    voice_id = client.enroll_voice(sample_audio, prefix="my_custom_voice")

    if not voice_id:
        print("❌ 音色复刻失败")
        return

    print(f"✅ 复刻成功!")
    print(f"   Voice ID: {voice_id}")

    # Step 2: 使用复刻的音色合成语音
    text = "这是使用我的声音复刻模型合成的音频。"
    print(f"\nStep 2: 合成语音...")
    print(f"   文本: {text}")
    print(f"   Voice ID: {voice_id}")

    audio = client.synthesize(text, voice=voice_id)

    # 保存音频
    output_file = f"output_cloned_{voice_id}.mp3"
    with open(output_file, "wb") as f:
        f.write(audio)

    print(f"✅ 合成成功，已保存为: {output_file}")
    print(f"   音频大小: {len(audio):,} bytes")


def example_3_voice_cloning_auto():
    """示例 3: 声音复刻 - 自动复刻（一次性使用）"""
    print("\n" + "=" * 60)
    print("示例 3: 声音复刻模式 - 自动复刻")
    print("=" * 60)

    # 初始化客户端
    client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")

    # 样本音频
    sample_audio = "sample_voice.wav"

    if not Path(sample_audio).exists():
        print(f"⚠️  样本音频不存在: {sample_audio}")
        print("   请准备一个 10-20 秒的音频文件（wav/mp3）")
        return

    # 自动复刻 + 合成（一步完成）
    text = "这是使用自动复刻功能合成的音频。"
    print(f"\n正在自动复刻并合成...")
    print(f"   样本音频: {sample_audio}")
    print(f"   文本: {text}")

    audio = client.synthesize(
        text, auto_clone=True, clone_audio_path=sample_audio
    )

    # 保存音频
    output_file = "output_auto_cloned.mp3"
    with open(output_file, "wb") as f:
        f.write(audio)

    print(f"✅ 合成成功，已保存为: {output_file}")
    print(f"   音频大小: {len(audio):,} bytes")


def example_4_batch_synthesis():
    """示例 4: 批量合成"""
    print("\n" + "=" * 60)
    print("示例 4: 批量合成（系统音色）")
    print("=" * 60)

    client = TTSClient(model="cosyvoice-v1", voice="longxiaochun")

    texts = [
        "第一句话：你好，欢迎使用语音合成服务。",
        "第二句话：这是批量合成的示例。",
        "第三句话：每句话都会生成单独的音频文件。",
    ]

    print(f"\n正在批量合成 {len(texts)} 段文本...")

    audios = client.synthesize_batch(texts)

    # 保存音频
    for i, audio in enumerate(audios, 1):
        if audio:
            output_file = f"output_batch_{i}.mp3"
            with open(output_file, "wb") as f:
                f.write(audio)
            print(f"✅ 第 {i} 段合成完成: {output_file} ({len(audio):,} bytes)")
        else:
            print(f"❌ 第 {i} 段合成失败")


def example_5_error_handling():
    """示例 5: 错误处理"""
    print("\n" + "=" * 60)
    print("示例 5: 错误处理")
    print("=" * 60)

    # 错误 1: 复刻模型未提供 voice_id
    print("\n错误 1: 复刻模型未提供 voice_id")
    client = TTSClient(model="qwen3-tts-vc-realtime-2026-01-15")
    try:
        # 尝试使用系统音色名称（错误）
        audio = client.synthesize("测试", voice="longxiaochun")
    except ValueError as e:
        print(f"   ❌ 预期错误: {e}")

    # 错误 2: 系统音色模型调用复刻功能
    print("\n错误 2: 系统音色模型调用复刻功能")
    client = TTSClient(model="cosyvoice-v1")
    voice_id = client.enroll_voice("sample.wav")
    if voice_id is None:
        print("   ⚠️  预期警告: 系统音色模型不支持复刻")


def main():
    """运行所有示例"""
    print("\n🎤 DashScope TTS 客户端使用示例")
    print("=" * 60)

    # 检查 API Key
    if not os.getenv("DASHSCOPE_API_KEY"):
        print("❌ 请设置环境变量 DASHSCOPE_API_KEY")
        print("   export DASHSCOPE_API_KEY='sk-...'")
        return

    try:
        # 示例 1: 系统音色
        example_1_system_voice()

        # 示例 2: 声音复刻（手动）
        # example_2_voice_cloning_manual()

        # 示例 3: 声音复刻（自动）
        # example_3_voice_cloning_auto()

        # 示例 4: 批量合成
        # example_4_batch_synthesis()

        # 示例 5: 错误处理
        example_5_error_handling()

    except Exception as e:
        print(f"\n❌ 执行失败: {e}")
        import traceback

        traceback.print_exc()

    print("\n" + "=" * 60)
    print("示例完成!")
    print("=" * 60)


if __name__ == "__main__":
    main()
