'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import useSWR from 'swr';
import {
  ArrowLeft,
  Download,
  Trash2,
  RefreshCw,
  Clock,
  MessageSquare,
  Mic,
  AlertCircle,
  CheckCircle,
  Subtitles,
} from 'lucide-react';
import {
  getTask,
  deleteTask,
  getDownloadUrl,
  getStatusLabel,
  getStatusColor,
  formatDateTime,
  formatDuration,
  getLanguageName,
} from '@/lib/api';

export default function TaskDetailPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const [deleting, setDeleting] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [downloadingSubtitle, setDownloadingSubtitle] = useState(false);

  // 使用 SWR 自动刷新任务详情
  const { data: task, error, isLoading, mutate } = useSWR(
    ['task', params.id],
    () => getTask(params.id),
    {
      refreshInterval: (data) => {
        // 任务未完成时每 2 秒刷新一次
        if (data && data.status !== 'completed' && data.status !== 'failed') {
          return 2000;
        }
        // 已完成或失败时停止自动刷新
        return 0;
      },
      revalidateOnFocus: true,
    }
  );

  // 删除任务
  const handleDelete = async () => {
    if (!confirm('确定要删除这个任务吗？此操作不可恢复。')) {
      return;
    }

    try {
      setDeleting(true);
      await deleteTask(params.id);
      router.push('/tasks');
    } catch (err: any) {
      alert(`删除失败：${err.message}`);
      setDeleting(false);
    }
  };

  // 下载文件通用方法（兼容移动端）
  // 注意：download 属性对跨域 URL 无效，需后端配置 Content-Disposition 响应头
  const triggerDownload = (url: string) => {
    const link = document.createElement('a');
    link.href = url;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  // 下载结果
  const handleDownload = async () => {
    try {
      setDownloading(true);
      const { download_url } = await getDownloadUrl(params.id);
      // 使用 a 标签触发下载，兼容移动端
      triggerDownload(download_url);
    } catch (err: any) {
      alert(`获取下载链接失败：${err.message}`);
    } finally {
      setDownloading(false);
    }
  };

  // 下载字幕
  const handleDownloadSubtitle = async () => {
    try {
      setDownloadingSubtitle(true);
      const { subtitle_url } = await getDownloadUrl(params.id);
      if (subtitle_url) {
        // 使用 a 标签触发下载，兼容移动端
        triggerDownload(subtitle_url);
      } else {
        alert('该任务没有字幕文件');
      }
    } catch (err: any) {
      alert(`获取字幕链接失败：${err.message}`);
    } finally {
      setDownloadingSubtitle(false);
    }
  };

  if (error) {
    return (
      <div className="max-w-4xl mx-auto text-center py-12">
        <AlertCircle className="w-16 h-16 text-red-500 mx-auto mb-4" />
        <h2 className="text-2xl font-bold text-slate-900 mb-2">加载失败</h2>
        <p className="text-slate-600 mb-6">{error.message}</p>
        <Link
          href="/tasks"
          className="inline-flex items-center gap-2 text-blue-600 hover:text-blue-700"
        >
          <ArrowLeft className="w-4 h-4" />
          返回任务列表
        </Link>
      </div>
    );
  }

  if (isLoading || !task) {
    return (
      <div className="max-w-4xl mx-auto text-center py-12">
        <div className="inline-block animate-spin rounded-full h-12 w-12 border-4 border-blue-600 border-t-transparent mb-4" />
        <p className="text-slate-600">加载中...</p>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto px-4 py-8">
      {/* 返回按钮 */}
      <Link
        href="/tasks"
        className="inline-flex items-center gap-2 text-slate-600 hover:text-slate-900 mb-6 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        返回任务列表
      </Link>

      {/* 头部信息 */}
      <div className="bg-white rounded-lg border border-slate-200 p-6 mb-6">
        <div className="flex items-start justify-between gap-4 mb-4">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-2xl font-bold text-slate-900">
                {task.title || '未命名任务'}
              </h1>
              <span
                className={`px-3 py-1 text-sm font-medium rounded-full ${getStatusColor(
                  task.status
                )}`}
              >
                {getStatusLabel(task.status)}
              </span>
            </div>

            <div className="flex items-center gap-4 text-sm text-slate-600">
              <span>
                {getLanguageName(task.source_language)} → {getLanguageName(task.target_language)}
              </span>
              <span>•</span>
              <span>任务 ID: {task.id}</span>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => mutate()}
              disabled={isLoading}
              className="p-2 border border-slate-300 text-slate-700 rounded-lg hover:bg-slate-50 disabled:opacity-50 transition-colors"
              title="刷新"
            >
              <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            </button>

            {task.status === 'completed' && (
              <>
                <button
                  onClick={handleDownload}
                  disabled={downloading}
                  className="flex items-center gap-2 bg-green-600 text-white px-4 py-2 rounded-lg font-medium hover:bg-green-700 disabled:opacity-50 transition-colors"
                >
                  <Download className="w-4 h-4" />
                  {downloading ? '获取中...' : '下载视频'}
                </button>

                {task.subtitle_file_path && task.subtitle_mode === 'EXTERNAL' && (
                  <button
                    onClick={handleDownloadSubtitle}
                    disabled={downloadingSubtitle}
                    className="flex items-center gap-2 bg-emerald-600 text-white px-4 py-2 rounded-lg font-medium hover:bg-emerald-700 disabled:opacity-50 transition-colors"
                  >
                    <Subtitles className="w-4 h-4" />
                    {downloadingSubtitle ? '获取中...' : '下载字幕'}
                  </button>
                )}
              </>
            )}

            <button
              onClick={handleDelete}
              disabled={deleting}
              className="flex items-center gap-2 bg-red-600 text-white px-4 py-2 rounded-lg font-medium hover:bg-red-700 disabled:opacity-50 transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              {deleting ? '删除中...' : '删除'}
            </button>
          </div>
        </div>

        {/* 进度条 */}
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-slate-600">{task.current_step || '等待处理'}</span>
            <span className="font-medium text-slate-900">{task.progress}%</span>
          </div>
          <div className="w-full h-3 bg-slate-200 rounded-full overflow-hidden">
            <div
              className={`h-full transition-all duration-500 ${
                task.status === 'completed'
                  ? 'bg-green-500'
                  : task.status === 'failed'
                  ? 'bg-red-500'
                  : 'bg-blue-500'
              }`}
              style={{ width: `${task.progress}%` }}
            />
          </div>
        </div>

        {/* 错误信息 */}
        {task.error_message && (
          <div className="mt-4 flex items-start gap-3 p-4 bg-red-50 border border-red-200 rounded-lg">
            <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-medium text-red-900">处理失败</p>
              <p className="text-sm text-red-800 mt-1">{task.error_message}</p>
            </div>
          </div>
        )}

        {/* 完成提示 */}
        {task.status === 'completed' && (
          <div className="mt-4 flex items-start gap-3 p-4 bg-green-50 border border-green-200 rounded-lg">
            <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-medium text-green-900">处理完成</p>
              <p className="text-sm text-green-800 mt-1">
                视频已成功配音，点击上方按钮下载结果
                {task.subtitle_file_path && task.subtitle_mode === 'EXTERNAL' && (
                  <span>（含外挂字幕文件）</span>
                )}
                {task.subtitle_mode === 'BURN' && (
                  <span>（字幕已烧录到视频中）</span>
                )}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* 详细信息 */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
        <InfoCard
          icon={<Clock className="w-5 h-5 text-blue-600" />}
          title="时长"
          value={task.video_duration_ms ? formatDuration(task.video_duration_ms) : '--'}
        />
        <InfoCard
          icon={<MessageSquare className="w-5 h-5 text-purple-600" />}
          title="分段数量"
          value={task.segment_count.toString()}
        />
        <InfoCard
          icon={<Mic className="w-5 h-5 text-pink-600" />}
          title="创建时间"
          value={formatDateTime(task.created_at)}
        />
      </div>

      {/* 分段列表 */}
      {task.segments.length > 0 && (
        <div className="bg-white rounded-lg border border-slate-200 p-6">
          <h2 className="text-xl font-bold text-slate-900 mb-4">
            分段详情 ({task.segments.length})
          </h2>

          <div className="space-y-3">
            {task.segments
              .slice()
              .sort((a, b) => a.start_time_ms - b.start_time_ms)
              .map((segment, index) => (
              <div
                key={segment.id}
                className="p-4 bg-slate-50 rounded-lg border border-slate-200"
              >
                <div className="flex items-start justify-between gap-4 mb-2">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <span className="px-2 py-0.5 bg-slate-200 text-slate-700 text-xs font-medium rounded">
                        #{index + 1}
                      </span>
                      <span className="text-xs text-slate-600">
                        {formatDuration(segment.start_time_ms)} -{' '}
                        {formatDuration(segment.end_time_ms)}
                      </span>
                      {segment.speaker_id && (
                        <span className="text-xs text-slate-600">
                          说话人: {segment.speaker_id}
                        </span>
                      )}
                      {segment.voice_id && (
                        <span className="text-xs text-purple-600 font-medium">
                          复刻: {segment.voice_id.substring(0, 12)}...
                        </span>
                      )}
                    </div>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <p className="text-xs text-slate-500 mb-1">原文</p>
                        <p className="text-sm text-slate-900">
                          {segment.original_text || '--'}
                        </p>
                      </div>
                      <div>
                        <p className="text-xs text-slate-500 mb-1">译文</p>
                        <p className="text-sm text-slate-900">
                          {segment.translated_text || '--'}
                        </p>
                      </div>
                    </div>
                  </div>

                  {segment.confidence !== null && (
                    <div className="text-right">
                      <p className="text-xs text-slate-500">置信度</p>
                      <p className="text-sm font-medium text-slate-900">
                        {(segment.confidence * 100).toFixed(1)}%
                      </p>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 元数据 */}
      <div className="mt-6 bg-slate-50 rounded-lg border border-slate-200 p-6">
        <h3 className="text-sm font-medium text-slate-700 mb-3">元数据</h3>
        <dl className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
          <div>
            <dt className="text-slate-500">任务 ID</dt>
            <dd className="font-mono text-slate-900 break-all">{task.id}</dd>
          </div>
          {task.celery_task_id && (
            <div>
              <dt className="text-slate-500">Celery 任务 ID</dt>
              <dd className="font-mono text-slate-900 break-all">{task.celery_task_id}</dd>
            </div>
          )}
          <div>
            <dt className="text-slate-500">创建时间</dt>
            <dd className="text-slate-900">{formatDateTime(task.created_at)}</dd>
          </div>
          <div>
            <dt className="text-slate-500">更新时间</dt>
            <dd className="text-slate-900">{formatDateTime(task.updated_at)}</dd>
          </div>
          {task.completed_at && (
            <div>
              <dt className="text-slate-500">完成时间</dt>
              <dd className="text-slate-900">{formatDateTime(task.completed_at)}</dd>
            </div>
          )}
        </dl>
      </div>
    </div>
  );
}

// 信息卡片组件
function InfoCard({
  icon,
  title,
  value,
}: {
  icon: React.ReactNode;
  title: string;
  value: string;
}) {
  return (
    <div className="bg-white rounded-lg border border-slate-200 p-4">
      <div className="flex items-center gap-3 mb-2">
        {icon}
        <p className="text-sm text-slate-600">{title}</p>
      </div>
      <p className="text-2xl font-bold text-slate-900">{value}</p>
    </div>
  );
}
