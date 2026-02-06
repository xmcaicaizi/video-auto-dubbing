export type TaskStatus =
  | 'pending'
  | 'uploading'
  | 'extracting'
  | 'transcribing'
  | 'translating'
  | 'synthesizing'
  | 'muxing'
  | 'completed'
  | 'failed';

export interface Segment {
  id: string;
  task_id: string;
  segment_index: number;
  start_time_ms: number;
  end_time_ms: number;
  original_text: string;
  translated_text: string | null;
  speaker_id: string | null;
  emotion: string | null;
  confidence: number | null;
  audio_path: string | null;
  voice_id: string | null;
}

export type SubtitleMode = 'none' | 'external' | 'burn';

export interface Task {
  id: string;
  title: string;
  source_language: string;
  target_language: string;
  status: TaskStatus;
  subtitle_mode: SubtitleMode;

  // 进度相关
  current_step: string | null;
  progress: number;
  error_message: string | null;

  // 文件路径
  input_video_path: string | null;
  extracted_audio_path: string | null;
  output_video_path: string | null;
  subtitle_file_path: string | null;

  // 元数据
  video_duration_ms: number | null;
  segment_count: number;

  created_at: string;
  updated_at: string;

  // 关联数据
  segments?: Segment[];
}

export interface TaskCreatePayload {
  video: File;
  source_language: string;
  target_language: string;
  title?: string;
  subtitle_mode?: SubtitleMode;
}

export interface TaskListResponse {
  items: Task[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface TaskResultResponse {
  download_url: string;
  subtitle_url?: string;
  expires_in: number;
}
