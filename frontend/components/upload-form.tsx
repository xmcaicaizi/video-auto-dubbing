'use client';

import { useState, useCallback } from 'react';
import { useForm } from 'react-hook-form';
import { useRouter } from 'next/navigation';
import { Upload, FileVideo, X, Loader2, AlertCircle } from 'lucide-react';
import { useCreateTask } from '@/lib/hooks/use-tasks';
import { TaskCreatePayload } from '@/lib/types';
import { useDropzone } from 'react-dropzone';

// 语言选项
const LANGUAGES = [
  { value: 'zh', label: '中文 (Chinese)' },
  { value: 'en', label: '英语 (English)' },
  { value: 'ja', label: '日语 (Japanese)' },
  { value: 'ko', label: '韩语 (Korean)' },
  { value: 'es', label: '西班牙语 (Spanish)' },
  { value: 'fr', label: '法语 (French)' },
  { value: 'de', label: '德语 (German)' },
  { value: 'ru', label: '俄语 (Russian)' },
];

export default function UploadForm() {
  const router = useRouter();
  const [uploadError, setUploadError] = useState<string | null>(null);
  const createTaskMutation = useCreateTask();

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<TaskCreatePayload>({
    defaultValues: {
      source_language: 'zh',
      target_language: 'en',
      subtitle_mode: 'external',
    },
  });

  const selectedFile = watch('video');

  // 处理文件拖拽
  const onDrop = useCallback(
    (acceptedFiles: File[]) => {
      if (acceptedFiles.length > 0) {
        setValue('video', acceptedFiles[0], { shouldValidate: true });
        setUploadError(null);
      }
    },
    [setValue]
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'video/*': ['.mp4', '.mov', '.avi', '.mkv', '.webm'],
    },
    maxFiles: 1,
    multiple: false,
  });

  // 移除文件
  const removeFile = (e: React.MouseEvent) => {
    e.stopPropagation();
    setValue('video', null as any, { shouldValidate: true });
  };

  // 提交表单
  const onSubmit = async (data: TaskCreatePayload) => {
    try {
      setUploadError(null);
      await createTaskMutation.mutateAsync(data);
      router.push('/tasks'); // 跳转到列表页
    } catch (error: any) {
      console.error('Upload failed:', error);
      setUploadError(
        error.message || '创建任务失败，请重试'
      );
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-8 max-w-2xl mx-auto p-6 bg-white rounded-lg shadow-sm border border-slate-100">

      {/* 1. 视频上传区域 */}
      <div className="space-y-4">
        <label className="block text-sm font-medium text-slate-700">
          上传视频 <span className="text-red-500">*</span>
        </label>

        <div
          {...getRootProps()}
          className={`
            relative border-2 border-dashed rounded-lg p-8 transition-colors text-center cursor-pointer
            ${isDragActive ? 'border-blue-500 bg-blue-50' : 'border-slate-300 hover:border-slate-400'}
            ${errors.video ? 'border-red-500 bg-red-50' : ''}
          `}
        >
          <input
            {...getInputProps()}
            // 必须手动注册 input 的 onChange，否则 react-hook-form 无法接管 file input
            onChange={(e) => {
                if (e.target.files && e.target.files.length > 0) {
                    setValue('video', e.target.files[0], { shouldValidate: true });
                }
            }}
          />

          {selectedFile ? (
            <div className="flex items-center justify-center space-x-4">
              <FileVideo className="w-10 h-10 text-blue-600" />
              <div className="text-left">
                <p className="font-medium text-slate-900 line-clamp-1 max-w-[200px]">
                  {selectedFile.name}
                </p>
                <p className="text-xs text-slate-500">
                  {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                </p>
              </div>
              <button
                type="button"
                onClick={removeFile}
                className="p-1 hover:bg-slate-200 rounded-full transition-colors"
              >
                <X className="w-5 h-5 text-slate-500" />
              </button>
            </div>
          ) : (
            <div className="space-y-2">
              <Upload className="w-10 h-10 mx-auto text-slate-400" />
              <p className="text-sm text-slate-600">
                {isDragActive ? '释放以上传' : '点击或拖拽视频到此处'}
              </p>
              <p className="text-xs text-slate-400">支持 MP4, MOV, AVI (最大 500MB)</p>
            </div>
          )}
        </div>
        {errors.video && (
          <p className="text-sm text-red-500">{errors.video.message || '请上传视频文件'}</p>
        )}
      </div>

      {/* 2. 语言选择区域 */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* 源语言 */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700">
            源语言 (Source) <span className="text-red-500">*</span>
          </label>
          <select
            {...register('source_language', { required: '请选择源语言' })}
            className="w-full rounded-md border-slate-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm py-2.5"
          >
            {LANGUAGES.map((lang) => (
              <option key={lang.value} value={lang.value}>
                {lang.label}
              </option>
            ))}
          </select>
        </div>

        {/* 目标语言 */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700">
            目标语言 (Target) <span className="text-red-500">*</span>
          </label>
          <select
            {...register('target_language', { required: '请选择目标语言' })}
            className="w-full rounded-md border-slate-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm py-2.5"
          >
            {LANGUAGES.map((lang) => (
              <option key={lang.value} value={lang.value}>
                {lang.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* 3. 字幕模式 */}
      <div className="space-y-2">
        <label className="block text-sm font-medium text-slate-700">
          字幕模式
        </label>
        <select
          {...register('subtitle_mode')}
          className="w-full rounded-md border-slate-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm py-2.5"
        >
          <option value="external">外挂字幕（生成 .ass 文件，可单独下载）</option>
          <option value="burn">烧录字幕（嵌入视频画面，不可关闭）</option>
          <option value="none">不生成字幕</option>
        </select>
        <p className="text-xs text-slate-400">
          外挂字幕可在播放器中开关，烧录字幕会永久嵌入视频（处理较慢）
        </p>
      </div>

      {/* 4. 任务标题（可选） */}
      <div className="space-y-2">
        <label className="block text-sm font-medium text-slate-700">
          任务标题 (可选)
        </label>
        <input
          type="text"
          {...register('title')}
          placeholder="默认为文件名"
          className="w-full rounded-md border-slate-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm py-2.5"
        />
      </div>

      {/* 错误提示 */}
      {uploadError && (
        <div className="bg-red-50 border-l-4 border-red-500 p-4 rounded-r flex items-start">
          <AlertCircle className="w-5 h-5 text-red-500 mr-2 mt-0.5" />
          <p className="text-sm text-red-700">{uploadError}</p>
        </div>
      )}

      {/* 提交按钮 */}
      <button
        type="submit"
        disabled={createTaskMutation.isPending || !selectedFile}
        className={`
          w-full flex justify-center py-3 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white
          transition-all duration-200
          ${
            createTaskMutation.isPending || !selectedFile
              ? 'bg-slate-400 cursor-not-allowed'
              : 'bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
          }
        `}
      >
        {createTaskMutation.isPending ? (
          <>
            <Loader2 className="w-5 h-5 mr-2 animate-spin" />
            正在创建任务...
          </>
        ) : (
          '开始配音'
        )}
      </button>
    </form>
  );
}
