'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Upload, AlertCircle, Sparkles, Languages as LanguagesIcon, FileVideo, Subtitles } from 'lucide-react';
import Link from 'next/link';
import { createTask, SUPPORTED_LANGUAGES, formatFileSize, SubtitleMode } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Alert, AlertDescription } from '@/components/ui/alert';

export default function NewTaskPage() {
  const router = useRouter();

  // 表单状态
  const [file, setFile] = useState<File | null>(null);
  const [title, setTitle] = useState('');
  const [sourceLanguage, setSourceLanguage] = useState('zh');
  const [targetLanguage, setTargetLanguage] = useState('en');
  const [subtitleMode, setSubtitleMode] = useState<SubtitleMode>('burn');

  // UI 状态
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dragActive, setDragActive] = useState(false);

  // 文件选择
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      validateAndSetFile(selectedFile);
    }
  };

  // 文件拖拽
  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    const droppedFile = e.dataTransfer.files?.[0];
    if (droppedFile) {
      validateAndSetFile(droppedFile);
    }
  };

  // 验证并设置文件
  const validateAndSetFile = (selectedFile: File) => {
    setError(null);

    // 检查文件类型
    const allowedTypes = ['video/mp4', 'video/avi', 'video/quicktime', 'video/x-matroska', 'video/x-flv'];
    if (!allowedTypes.includes(selectedFile.type)) {
      setError('不支持的文件格式。请上传 MP4, AVI, MOV, MKV 或 FLV 格式的视频。');
      return;
    }

    // 检查文件大小 (500MB)
    const maxSize = 500 * 1024 * 1024;
    if (selectedFile.size > maxSize) {
      setError('文件过大。最大支持 500MB 的视频文件。');
      return;
    }

    setFile(selectedFile);

    // 自动填充标题
    if (!title) {
      const fileName = selectedFile.name.replace(/\.[^/.]+$/, ''); // 移除扩展名
      setTitle(fileName);
    }
  };

  // 提交任务
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!file) {
      setError('请选择视频文件');
      return;
    }

    if (sourceLanguage === targetLanguage) {
      setError('源语言和目标语言不能相同');
      return;
    }

    try {
      setUploading(true);

      const task = await createTask(file, sourceLanguage, targetLanguage, title, subtitleMode);

      // 跳转到任务详情页
      router.push(`/tasks/${task.id}`);
    } catch (err: any) {
      setError(err.message || '创建任务失败');
      setUploading(false);
    }
  };

  return (
    <div className="container max-w-4xl mx-auto px-4 py-8">
      {/* 返回按钮 */}
      <Button variant="ghost" asChild className="mb-6">
        <Link href="/tasks">
          <ArrowLeft className="w-4 h-4 mr-2" />
          返回任务列表
        </Link>
      </Button>

      {/* 标题 */}
      <div className="mb-8 text-center">
        <h1 className="text-4xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-indigo-600 mb-2">
          创建配音任务
        </h1>
        <p className="text-lg text-muted-foreground">
          上传视频，选择语言，让 AI 为您完成配音
        </p>
      </div>

      {/* 表单 */}
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* 文件上传卡片 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileVideo className="w-5 h-5 text-blue-600" />
              视频文件
            </CardTitle>
            <CardDescription>
              支持 MP4, AVI, MOV, MKV, FLV 格式，最大 500MB
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div
              className={`
                relative border-2 border-dashed rounded-xl p-12 text-center transition-all
                ${dragActive ? 'border-blue-500 bg-blue-50' : 'border-border hover:border-blue-300'}
                ${file ? 'bg-green-50 border-green-500' : 'bg-background'}
              `}
              onDragEnter={handleDrag}
              onDragOver={handleDrag}
              onDragLeave={handleDrag}
              onDrop={handleDrop}
            >
              <input
                type="file"
                accept="video/*"
                onChange={handleFileChange}
                disabled={uploading}
                className="absolute inset-0 w-full h-full opacity-0 cursor-pointer disabled:cursor-not-allowed"
              />

              {file ? (
                <div className="space-y-3">
                  <div className="w-16 h-16 mx-auto bg-green-100 rounded-full flex items-center justify-center">
                    <FileVideo className="w-8 h-8 text-green-600" />
                  </div>
                  <div>
                    <p className="text-lg font-semibold text-green-900">{file.name}</p>
                    <p className="text-sm text-green-700 mt-1">{formatFileSize(file.size)}</p>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      setFile(null);
                    }}
                    disabled={uploading}
                  >
                    重新选择
                  </Button>
                </div>
              ) : (
                <div className="space-y-3">
                  <div className="w-16 h-16 mx-auto bg-blue-100 rounded-full flex items-center justify-center">
                    <Upload className="w-8 h-8 text-blue-600" />
                  </div>
                  <div>
                    <p className="text-lg font-medium text-foreground">
                      <span className="text-blue-600 font-semibold">点击选择</span> 或拖拽文件到此处
                    </p>
                    <p className="text-sm text-muted-foreground mt-1">
                      支持 MP4, AVI, MOV, MKV, FLV（最大 500MB）
                    </p>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* 任务标题卡片 */}
        <Card>
          <CardHeader>
            <CardTitle>任务信息</CardTitle>
            <CardDescription>为您的配音任务设置标题（可选）</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <Label htmlFor="title">任务标题</Label>
              <Input
                type="text"
                id="title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                disabled={uploading}
                placeholder="默认使用文件名"
                className="text-base"
              />
            </div>
          </CardContent>
        </Card>

        {/* 语言选择卡片 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <LanguagesIcon className="w-5 h-5 text-indigo-600" />
              语言配置
            </CardTitle>
            <CardDescription>选择视频的原始语言和目标配音语言</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label htmlFor="source-lang">源语言</Label>
                <Select
                  value={sourceLanguage}
                  onValueChange={setSourceLanguage}
                  disabled={uploading}
                >
                  <SelectTrigger id="source-lang">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SUPPORTED_LANGUAGES.map((lang) => (
                      <SelectItem key={lang.code} value={lang.code}>
                        {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="target-lang">目标语言</Label>
                <Select
                  value={targetLanguage}
                  onValueChange={setTargetLanguage}
                  disabled={uploading}
                >
                  <SelectTrigger id="target-lang">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SUPPORTED_LANGUAGES.map((lang) => (
                      <SelectItem key={lang.code} value={lang.code}>
                        {lang.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 字幕配置卡片 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Subtitles className="w-5 h-5 text-emerald-600" />
              字幕配置
            </CardTitle>
            <CardDescription>选择是否生成字幕以及字幕模式</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <Label htmlFor="subtitle-mode">字幕模式</Label>
              <Select
                value={subtitleMode}
                onValueChange={(value) => setSubtitleMode(value as SubtitleMode)}
                disabled={uploading}
              >
                <SelectTrigger id="subtitle-mode">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="burn">
                    <div className="flex flex-col">
                      <span>烧录字幕（推荐）</span>
                      <span className="text-xs text-muted-foreground">将字幕嵌入视频画面，无需单独加载</span>
                    </div>
                  </SelectItem>
                  <SelectItem value="external">
                    <div className="flex flex-col">
                      <span>外挂字幕</span>
                      <span className="text-xs text-muted-foreground">生成独立 .ass 字幕文件，可单独下载</span>
                    </div>
                  </SelectItem>
                  <SelectItem value="none">
                    <div className="flex flex-col">
                      <span>不生成字幕</span>
                      <span className="text-xs text-muted-foreground">仅配音，不添加任何字幕</span>
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* 错误提示 */}
        {error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* 提交按钮 */}
        <div className="flex items-center gap-4">
          <Button
            type="submit"
            disabled={!file || uploading}
            className="flex-1 py-6 text-lg"
          >
            {uploading ? (
              <>
                <div className="mr-2 h-5 w-5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                上传中...
              </>
            ) : (
              <>
                <Sparkles className="w-5 h-5 mr-2" />
                创建任务
              </>
            )}
          </Button>

          <Button type="button" variant="outline" asChild className="py-6 text-lg">
            <Link href="/tasks">取消</Link>
          </Button>
        </div>
      </form>

      {/* 提示信息 */}
      <Card className="mt-8 border-blue-200 bg-blue-50/50">
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2 text-blue-900">
            <Sparkles className="w-5 h-5" />
            处理流程说明
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ol className="space-y-2 text-sm text-blue-900">
            <li className="flex items-start gap-2">
              <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-bold">1</span>
              <span><strong>提取音频：</strong>从视频中提取音轨</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-bold">2</span>
              <span><strong>语音识别：</strong>使用 ASR 识别语音内容</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-bold">3</span>
              <span><strong>文本翻译：</strong>将识别的文本翻译成目标语言</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-bold">4</span>
              <span><strong>语音合成：</strong>使用 TTS 生成配音</span>
            </li>
            <li className="flex items-start gap-2">
              <span className="flex-shrink-0 w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-bold">5</span>
              <span><strong>视频合成：</strong>将新音轨合并到视频中</span>
            </li>
          </ol>
          <p className="text-sm text-blue-700 mt-4 pt-4 border-t border-blue-200">
            ⏱️ 处理时间取决于视频长度，通常需要 3-10 分钟。您可以在任务列表中实时查看处理进度。
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
