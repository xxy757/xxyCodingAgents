import { useMutation, useQueryClient } from '@tanstack/react-query';
import { tasksApi } from '@/lib/api';

export function useRetryTask(runId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: string) => tasksApi.retry(taskId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs', runId, 'tasks'] }),
  });
}

export function useCancelTask(runId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: string) => tasksApi.cancel(taskId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs', runId, 'tasks'] }),
  });
}
