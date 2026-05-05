// utils.ts - 通用工具函数

import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

/** 格式化日期为 YYYY-MM-DD HH:mm:ss */
export function formatDate(dateStr: string): string {
  return dayjs(dateStr).format('YYYY-MM-DD HH:mm:ss');
}

/** 格式化为相对时间（如"3分钟前"） */
export function formatRelative(dateStr: string): string {
  return dayjs(dateStr).fromNow();
}

/** 截取 ID 前 8 位 */
export function shortId(id: string): string {
  return id.slice(0, 8);
}

/** 解析仓库 URL 为可显示路径 */
export function repoDisplayPath(url: string): string {
  try {
    return new URL(url).pathname.slice(1).replace(/\.git$/, '');
  } catch {
    const ssh = url.match(/[^:]+:(.+)/);
    if (ssh) return ssh[1].replace(/\.git$/, '');
    return url;
  }
}

/** 合并 CSS 类名 */
export function cn(...classes: (string | false | undefined | null)[]): string {
  return classes.filter(Boolean).join(' ');
}
